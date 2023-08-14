package term

import (
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/exp/maps"

	errorsGo "github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/wm"
)

/*
type Terminal interface {
	Name() string
	// TTYDevName() string // from TTY

	Drawers() []Drawer
	Closer

	Query(qs string, p Parser) (string, error)
	Printf(format string, a ...any) (int, error)
	Draw(img image.Image, bounds image.Rectangle) error

	CreateTemp(pattern string) (*os.File, error) // bound to Terminal lifetime
	WriteString(s string) (n int, err error)

	// optional
	// Args() []string

	Surveyor
	environ.Proprietor
	TTY
	wm.Window
}
*/

var _ interface {
	internal.Closer
	Surveyor
	environ.Proprietor
	TTY
} = (*Terminal)(nil)

type (
	tty        = TTY
	querier    = Querier
	proprietor = environ.Proprietor
	arger      = internal.Arger
	closer     = internal.Closer
)

type Terminal struct {
	tty
	querier
	proprietor // Data
	surveyor   SurveyorLight
	arger
	w wm.Window
	closer
	drawers  []Drawer
	resizer  Resizer
	passages mux.Muxers

	ttyDefault                              TTY
	querierDefault                          Querier
	ttyProv, ttyProvDefault                 TTYProvider
	partialSurveyor, partialSurveyorDefault PartialSurveyor
	windowProvider, windowProviderDefault   wm.WindowProvider

	name    string
	ptyName string
	exe     string
	tempDir string

	// TODO add logger
}

// NewTerminal tries to recognize the terminal that manages the device ptyName and matches w.
// It will use non-zero implementations provided by the optional TerminalChecker methods:
//
//   - TTY(ptyName string, ci environ.Proprietor) (TTY, error)
//   - Querier(environ.Proprietor) Querier
//   - Surveyor(environ.Proprietor) PartialSurveyor
//   - Window(environ.Proprietor) (wm.Window, error)
//   - Args(environ.Proprietor) []string
//   - Exe(environ.Proprietor) string // alternative executable name if it differs from Name()
//
// The optional Options.…Fallback fields are applied in case the enforced Options fields are nil and
// the TermChecker also doesn't return a suggestion.
func NewTerminal(opts ...Option) (*Terminal, error) {
	// TODO update comment
	tm := &Terminal{proprietor: environ.NewProprietor()}
	if err := tm.SetOptions(append(opts, setInternalDefaults, setEnvAndMuxers(true))...); err != nil {
		return nil, err
	}

	ttyTmp, quTmp, err := getTTYAndQuerier(tm, nil)
	if err != nil {
		return nil, err
	}

	var checker TermChecker
	composeManuallyStr, composeManually := tm.Property(propkeys.ManualComposition)
	if !composeManually || composeManuallyStr != `true` {
		// find terminal checker
		chk, prChecker, err := findTermChecker(tm.proprietor, ttyTmp, quTmp)
		if err != nil {
			return nil, err
		}
		tm.proprietor.Merge(prChecker)
		checker = chk
	} else {
		checker = &termCheckerCore{}
	}
	// terminal specific settings
	tm, err = checker.NewTerminal(replaceTerminal(tm))
	if err != nil {
		return nil, err
	}

	return tm, nil
}

func getTTYAndQuerier(tm *Terminal, tc *termCheckerCore) (TTY, Querier, error) {
	// tty order: tm.tty, tm.ttyProv, tc.TTY, tm.ttyDefault, tm.ttyProvDefault
	// querier order: tm.querier, tc.Querier, tm.querierDefault

	setDefaultTTY := func(t TTY, ttyProv TTYProvider) (TTY, error) {
		if t != nil {
			return t, nil
		}
		if ttyProv == nil {
			return nil, errorsGo.New(`nil tty provider`)
		}
		tt, err := ttyProv(tm.ptyName)
		if err != nil {
			return nil, err
		}
		if tt == nil {
			return nil, errorsGo.New(`nil tty received from tty provider`)
		}
		return tt, nil
	}

	var ttyTemp TTY
	var errs []error
	tty, err := setDefaultTTY(tm.tty, tm.ttyProv)
	if err != nil {
		errs = append(errs, err)
	} else if tty != nil {
		tm.tty = tty
	}
	if tm.tty != nil {
		ttyTemp = tm.tty
	} else {
		if tc != nil {
			if ttyProv, okTTYProv := tc.parent.(interface {
				TTY(pytName string, ci environ.Proprietor) (TTY, error)
			}); okTTYProv {
				tty, err := ttyProv.TTY(tm.ptyName, tm.proprietor)
				if err != nil {
					errs = append(errs, err)
				} else if tty != nil {
					if tm.proprietor != nil {
						tm.tty = tty
					}
					ttyTemp = tty
				}
			}
		}
		if ttyTemp == nil {
			ttyDefault, err := setDefaultTTY(tm.ttyDefault, tm.ttyProvDefault)
			if err != nil {
				errs = append(errs, err)
			} else if ttyDefault != nil {
				tm.ttyDefault = ttyDefault
				ttyTemp = ttyDefault
			}
		}
	}
	if ttyTemp == nil {
		errTTYRet := errors.Join(errs...)
		if errTTYRet != nil {
			errTTYRet = errorsGo.WrapPrefix(errTTYRet, `no/failed tty provision;`, 0)
		} else {
			errTTYRet = errorsGo.New(`no tty provided`)
		}
		return nil, nil, errTTYRet
	}

	var quTemp Querier
	if tm.querier != nil {
		quTemp = tm.querier
	} else {
		if tc != nil {
			if querier, okQuerier := tc.parent.(interface {
				Querier(environ.Proprietor) Querier
			}); okQuerier {
				quTemp = querier.Querier(tm.proprietor)
			}
		}
		if quTemp == nil && tm.querierDefault != nil {
			quTemp = tm.querierDefault
		} else {
			return nil, nil, errorsGo.New(`nil querier`)
		}
	}

	return ttyTemp, quTemp, nil
}

func findTermChecker(env environ.Proprietor, tty TTY, qu Querier) (tc TermChecker, _ environ.Proprietor, e error) {
	var ttyTemp TTY
	if tty == nil || qu == nil {
		return GetRegTermChecker(internal.TermGenericName), nil, errorsGo.New(internal.ErrNilParam)
	}
	if env == nil {
		return GetRegTermChecker(internal.TermGenericName), nil, nil
	}
	pr := environ.NewProprietor()
	ptyName := tty.TTYDevName()

	termGenericPreCheck(env)

	prTmChkMap := make(map[string]struct{})
	// run preliminary checks, to see which test should be avoided in the final checks
	// e.g. querying conhost
	terminalCheckersAll := AllTerminalCheckers()
	for _, tchk := range terminalCheckersAll {
		if tchk == nil {
			continue
		}
		mightBe, prE := tchk.CheckExclude(env)
		_ = mightBe
		pr.Merge(prE)
	}

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		log.Println(`panic in findTermEnv`) // TODO
		var errs []error
		err := errorsGo.New(r)
		errs = append(errs, err)
		if ttyTemp != nil {
			log.Println(`closing temporary tty`) // TODO
			err := ttyTemp.Close()
			errs = append(errs, err)
		}
		if tty != nil {
			log.Println(`closing tty`) // TODO
			err := tty.Close()
			errs = append(errs, err)
		}
		tc = GetRegTermChecker(internal.TermGenericName)
		e = errorsGo.New(errors.Join(errs...))
	}()
	termGenericCheck(tty, qu, pr)
	env.Merge(pr)
	if _, avoidANSI := pr.Property(propkeys.AvoidANSI); !avoidANSI {
		// run final checks
		for _, tchk := range terminalCheckersAll {
			if tchk == nil {
				continue
			}
			tchkName := tchk.Name()
			passedExcl, okPassedExcl := pr.Property(propkeys.CheckTermEnvExclPrefix + tchkName)
			var exclEnvSkipped bool
			if okPassedExcl {
				switch passedExcl {
				case CheckTermPassed:
					break
				case CheckTermDummy:
					exclEnvSkipped = true
					fallthrough
				case CheckTermFailed:
					fallthrough
				default:
					pr.SetProperty(propkeys.CheckTermCompletePrefix+tchk.Name(), CheckTermFailed)
					continue
				}
			}
			ttyTemp = tty
			var usingTempTTY bool
			if ttyProv, okTTYProv := tchk.(interface {
				TTY(pytName string) (TTY, error)
			}); okTTYProv {
				t, err := ttyProv.TTY(ptyName)
				if err == nil {
					usingTempTTY = true
					ttyTemp = t
				}
			}
			if ttyTemp == nil {
				continue
			}
			if usingTempTTY {
				defer ttyTemp.Close()
			}
			is, prI := tchk.CheckIsQuery(qu, ttyTemp, env)
			if usingTempTTY {
				_ = ttyTemp.Close() // TODO reuse if same tty
			}
			if !is {
				exclSolePassed := !exclEnvSkipped
				if exclSolePassed {
					prI.Properties()
					passedIs, okPassedIs := prI.Property(propkeys.CheckTermQueryIsPrefix + tchkName)
					exclSolePassed = okPassedIs && passedIs == CheckTermDummy
				}
				if !exclSolePassed {
					pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, CheckTermFailed)
					continue
				}
			}
			pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, CheckTermPassed)
			pr.Merge(prI)
		}
	} else {
		for _, tchk := range terminalCheckersAll {
			if tchk == nil {
				continue
			}
			tchkName := tchk.Name()

			exclPassed, okNotExcl := pr.Property(propkeys.CheckTermEnvExclPrefix + tchkName)
			if !okNotExcl {
				continue
			}
			switch exclPassed {
			case CheckTermPassed, CheckTermDummy:
				pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, CheckTermPassed)
			case CheckTermFailed:
				fallthrough
			default:
				pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, CheckTermFailed)
				continue
			}
		}
	}
	for _, tchk := range terminalCheckersAll {
		tchkName := tchk.Name()
		passed, completed := pr.Property(propkeys.CheckTermCompletePrefix + tchkName)
		if completed && passed == CheckTermPassed {
			prTmChkMap[tchkName] = struct{}{}
		}
	}

	// single out a specific terminal checker if possible
	var termMatchName string
	useGenericTermIfUncertain := true // TODO
	switch l := len(prTmChkMap); {
	case l == 0:
		termMatchName = internal.TermGenericName
	case l == 1:
		for k := range prTmChkMap {
			termMatchName = k
			break
		}
	case l == 2:
		// assume `generic` is contained
		for k := range prTmChkMap {
			if k != internal.TermGenericName {
				termMatchName = k
				break
			}
		}
	case l > 2:
		// TODO: if windowQuery == nil then try to find corresponding window and recheck
		if useGenericTermIfUncertain {
			termMatchName = internal.TermGenericName
		} else {
			termNames := maps.Keys(prTmChkMap)
			sort.Strings(termNames)
			termNamesStr := strings.Join(termNames, ` `)
			return GetRegTermChecker(internal.TermGenericName), nil, errorsGo.Errorf(`more than 1 terminal check matched: %s`, termNamesStr)
		}
	}
	_, okMatch := prTmChkMap[termMatchName]
	if !okMatch {
		termMatchName = internal.TermGenericName
	}
	var checker TermChecker
	for _, tchk := range terminalCheckersAll {
		if tchk != nil && tchk.Name() == termMatchName {
			checker = tchk
			break
		}
	}
	if checker == nil {
		// This should only be possible if the generic TermChecker was removed from the register.
		return GetRegTermChecker(internal.TermGenericName), nil, errorsGo.New(`no matching terminal was found`)
	}

	return checker, pr, nil
}

func (t *Terminal) Name() string {
	if t == nil {
		return ``
	}
	return t.name
}

func (t *Terminal) Exe() string {
	if t == nil {
		return ``
	}
	var suffix string
	if runtime.GOOS == `windows` {
		suffix = `.exe`
	}
	if len(t.exe) > 0 {
		return t.exe + suffix
	}
	return t.name + suffix
}

func (t *Terminal) Query(qs string, p Parser) (string, error) { return t.query(qs, p) }

func (t *Terminal) query(qs string, p Parser) (_ string, err error) {
	if t == nil {
		return ``, errorsGo.New(internal.ErrNilReceiver)
	}
	if t.tty == nil {
		return ``, errorsGo.New(`nil tty`)
	}
	if t.querier == nil {
		return ``, errorsGo.New(`nil querier`)
	}
	if t.proprietor == nil {
		return ``, errorsGo.New(`nil proprietor`)
	}
	defer func() {
		if r := recover(); r != nil {
			err = errorsGo.New(r)
		}
	}()
	if _, avoidANSI := t.Property(propkeys.AvoidANSI); avoidANSI {
		return ``, errorsGo.New(internal.ErrPlatformNotSupported)
	}
	return t.querier.Query(qs, t.tty, p)
}

// CreateTemp ...
func (t *Terminal) CreateTemp(pattern string) (*os.File, error) {
	if t == nil {
		return nil, internal.ErrNilReceiver
	}
	if len(t.tempDir) == 0 {
		dir, err := os.MkdirTemp(``, internal.LibraryName+`.`)
		if err != nil {
			return nil, err
		}
		t.tempDir = dir
		onCloseFunc := func() error {
			return os.RemoveAll(dir)
		}
		t.OnClose(onCloseFunc)
	}

	f, err := os.CreateTemp(t.tempDir, pattern)
	if err != nil {
		return nil, err
	}
	onCloseFunc := func() error { return errorsGo.New(errors.Join(f.Close(), os.Remove(f.Name()))) }
	t.OnClose(onCloseFunc)
	onCloseFin := func(f *os.File) { _ = f.Close(); os.Remove(f.Name()) }
	runtime.SetFinalizer(f, onCloseFin)

	return f, nil
}

func (t *Terminal) Close() error {
	if t == nil || t.closer == nil {
		return nil
	}
	return t.closer.Close()
}

func (t *Terminal) addClosers(objs ...any) {
	if t == nil || t.closer == nil || len(objs) == 0 {
		return
	}
	var closers []interface{ Close() error }
	for _, obj := range objs {
		obj := obj
		if obj == nil {
			continue
		}
		closer, isCloser := any(obj).(interface{ Close() error })
		if isCloser {
			closers = append(closers, closer)
		}
		clearer, isClearer := any(obj).(interface{ Clear(term *Terminal) error }) // Drawer
		if isClearer {
			closers = append(closers, &clearCloser{clearer: clearer, term: t})
		}
	}
	t.closer.AddClosers(closers...)
}

type clearCloser struct {
	clearer interface{ Clear(term *Terminal) error }
	term    *Terminal
}

func (c *clearCloser) Close() error { return c.clearer.Clear(c.term) }

func (t *Terminal) Drawers() []Drawer {
	if t == nil {
		return nil
	}
	return t.drawers
}

func (t *Terminal) Printf(format string, a ...any) (int, error) {
	if t == nil {
		return 0, errorsGo.New(internal.ErrNilReceiver)
	}
	n, err := t.WriteString(t.passages.Wrap(fmt.Sprintf(format, a...)))
	if err != nil {
		return n, errorsGo.New(err)
	}
	return n, nil
}

func (t *Terminal) Write(p []byte) (n int, err error) {
	if t == nil {
		return 0, errorsGo.New(internal.ErrNilReceiver)
	}
	if t.tty == nil {
		return 0, errorsGo.New(`nil tty`)
	}
	defer func() {
		if r := recover(); r != nil {
			err = errorsGo.New(r)
		}
	}()
	return t.tty.Write(p)
}
func (t *Terminal) WriteString(s string) (n int, err error) {
	b := unsafe.Slice(unsafe.StringData(s), len(s))
	return t.Write(b)
}

func (t *Terminal) Draw(img image.Image, bounds image.Rectangle) error { return t.draw(img, bounds) }

func (t *Terminal) draw(img image.Image, bounds image.Rectangle) (e error) {
	if t == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	if len(t.drawers) == 0 {
		return errorsGo.New(`no drawers set`)
	}

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		// TODO log error
		var errs []error
		err := errorsGo.New(r)
		errs = append(errs, err)
		/* if t != nil && t.tty != nil {
			log.Println(`closing tty`) // TODO
			err := t.tty.Close()
			errs = append(errs, err)
		} */
		e = errorsGo.New(errors.Join(errs...))
	}()
	return t.drawers[0].Draw(img, bounds, t.resizer, t)
}

// CellScale returns a cell size for pixel size <ptPx> to the cell size <ptDest> while maintaining the scale.
// With no passed 0 side length values, the largest subarea is returned.
// With one passed 0 side length value, the other side length will be fixed.
// With two passed 0 side length values, pixels in source and destination area at the same position correspond to each other.
func (t *Terminal) CellScale(ptSrcPx, ptDstCl image.Point) (image.Point, error) {
	var ret image.Point
	if t == nil {
		return image.Point{}, errorsGo.New(internal.ErrNilReceiver)
	}
	cpw, cph, err := t.CellSize()
	if err != nil {
		return image.Point{}, err
	}
	if cpw < 1 || cph < 1 {
		return image.Point{}, errorsGo.New(`received invalid terminal cell size`)
	}
	if ptDstCl.X == 0 {
		if ptDstCl.Y == 0 {
			ret = image.Point{
				X: int(float64(ptSrcPx.X) / cpw),
				Y: int(float64(ptSrcPx.Y) / cph),
			}
		} else {
			ret.Y = ptDstCl.Y
			ret.X = int((float64(ptSrcPx.X) * float64(cph) * float64(ptDstCl.Y)) / (float64(ptSrcPx.Y) * float64(cpw)))
		}
	} else {
		ret.X = ptDstCl.X
		yScaled := int((float64(ptSrcPx.Y) * float64(cpw) * float64(ptDstCl.X)) / (float64(ptSrcPx.X) * float64(cph)))
		if ptDstCl.Y == 0 {
			ret.Y = yScaled
		} else {
			if yScaled <= ptDstCl.Y {
				ret.Y = yScaled
			} else {
				ret.Y = ptDstCl.Y
				ret.X = int((float64(ptSrcPx.X) * float64(cph) * float64(ptDstCl.Y)) / (float64(ptSrcPx.Y) * float64(cpw)))
			}
		}
	}
	return ret, nil
}

func (t *Terminal) CellSize() (width, height float64, err error) {
	if t == nil {
		return 0, 0, errorsGo.New(internal.ErrNilReceiver)
	}
	if t.surveyor == nil {
		return 0, 0, errorsGo.New(`nil surveyor`)
	}
	return t.surveyor.CellSize(t.tty, t.querier, t.w, t.proprietor)
}

func (t *Terminal) SizeInCells() (width, height uint, err error) {
	if t == nil {
		return 0, 0, errorsGo.New(internal.ErrNilReceiver)
	}
	if t.surveyor == nil {
		return 0, 0, errorsGo.New(`nil surveyor`)
	}
	return t.surveyor.SizeInCells(t.tty, t.querier, t.w, t.proprietor)
}

func (t *Terminal) SizeInPixels() (width, height uint, err error) {
	if t == nil {
		return 0, 0, errorsGo.New(internal.ErrNilReceiver)
	}
	if t.surveyor == nil {
		return 0, 0, errorsGo.New(`nil surveyor`)
	}
	return t.surveyor.SizeInPixels(t.tty, t.querier, t.w, t.proprietor)
}

func (t *Terminal) GetCursor() (xPosCells, yPosCells uint, err error) {
	if t == nil {
		return 0, 0, errorsGo.New(internal.ErrNilReceiver)
	}
	if t.surveyor == nil {
		return 0, 0, errorsGo.New(`nil surveyor`)
	}
	return t.surveyor.GetCursor(t.tty, t.querier, t.w, t.proprietor)
}

func (t *Terminal) SetCursor(xPosCells, yPosCells uint) (err error) {
	if t == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	if t.surveyor == nil {
		return errorsGo.New(`nil surveyor`)
	}
	return t.surveyor.SetCursor(xPosCells, yPosCells, t.tty, t.querier, t.w, t.proprietor)
}

// default
func init() { wm.SetImpl(wminternal.DummyImpl()) }

func (t *Terminal) Window() wm.Window {
	if t == nil {
		return nil
	}
	return t.w
}

////////////////////////////////////////////////////////////////////////////////

type TTYProvider func(ptyName string) (TTY, error)

var _ internal.Arger = (*arguments)(nil)

type arguments []string

func (a *arguments) Args() []string {
	if a == nil {
		return nil
	}
	return *a
}

func newArger(args []string) internal.Arger {
	a := arguments(args)
	return &a
}
