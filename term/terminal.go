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

	"github.com/srlehn/termimg/env/advanced"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
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
	proprietor
	surveyor SurveyorLight
	arger
	w wm.Window
	closer
	drawers  []Drawer
	rsz      Resizer // TODO
	passages mux.Muxers

	name    string
	ptyName string
	exe     string
	tempDir string
}

// ResetTerminalCheckerList
func ResetTerminalCheckerList() {
	termCheckerListIsInit = false
	_ = AllTerminalCheckers()
}

// NewTerminal tries to recognize the terminal that manages the device ptyName and matches w.
// It will use non-zero implementations provided by the optional TerminalChecker methods:
//
//   - TTY(pytName string, ci environ.Proprietor) (TTY, error)
//   - Querier(environ.Proprietor) Querier
//   - Surveyor(environ.Proprietor) PartialSurveyor
//   - Window(environ.Proprietor) (wm.Window, error)
//   - Args(environ.Proprietor) []string
//   - Exe(environ.Proprietor) string // alternative executable name if it differs from Name()
//
// Fallback options: ttyProvDefault (mandatory), quDefault, survPartDefault (optional)
func NewTerminal(cr *Creator) (*Terminal, error) {
	if cr == nil {
		return nil, errorsGo.New(internal.ErrNilParam)
	}
	if len(cr.PTYName) == 0 {
		// default if w == nil
		cr.PTYName = internal.DefaultTTYDevice()
	}
	if cr.TTYProvFallback == nil {
		return nil, errorsGo.New(`missing tty provider func`)
	}
	// set some package internal defaults
	if cr.PartialSurveyorFallback == nil {
		cr.PartialSurveyorFallback = &SurveyorDefault{}
	}
	if cr.WindowProviderFallback == nil {
		cr.WindowProviderFallback = wm.NewWindow
	}

	pr, passages, err := advanced.GetEnv(cr.PTYName)
	if err != nil {
		return nil, err
	}
	env := environ.CloneProprietor(pr)

	var tty TTY
	setDefaultTTY := func(t TTY) (TTY, error) {
		if t != nil {
			return t, nil
		}
		if cr.TTYProvFallback != nil {
			tt, err := cr.TTYProvFallback(cr.PTYName)
			if err != nil {
				util.Println(err) // TODO log error
				return nil, err
			}
			if tt != nil {
				return tt, nil
			}
		}
		return nil, errorsGo.New(`nil tty provider`)
	}
	tty, _ = setDefaultTTY(tty)

	quTemp := cr.Querier
	if quTemp == nil {
		quTemp = cr.QuerierFallback
	}
	if quTemp == nil {
		return nil, errorsGo.New(`both Creator.Querier and Creator.QuerierFallback are nil`)
	}
	checker, prChecker, err := findTermChecker(env, tty, quTemp)
	_ = err                       // TODO log error
	tty, err = setDefaultTTY(tty) // making sure...
	if err != nil {
		return nil, errorsGo.New(err)
	}

	pr.Merge(prChecker)

	cr.TTY = tty
	cr.Proprietor = pr
	tm, err := checker.NewTerminal(cr)
	if err != nil {
		return nil, err
	}

	tm.passages = passages

	return tm, nil
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

// ComposeTerminal manually composes a Terminal ignoring any incongruities.
func ComposeTerminal(cr *Creator) (*Terminal, error) {
	pr, passages, err := advanced.GetEnv(cr.PTYName)
	if err != nil {
		return nil, err
	}
	tm := &Terminal{
		tty:        cr.TTY,
		querier:    cr.Querier,
		surveyor:   getSurveyor(cr.PartialSurveyor, pr),
		proprietor: pr,
		arger:      cr.Arger,
		w:          cr.Window,
		closer:     internal.NewCloser(),
		passages:   passages,
		drawers:    cr.Drawers,
		name:       cr.TerminalName,
		ptyName:    cr.PTYName,
		exe:        cr.Exe,
	}
	tm.OnClose(func() error {
		if tm == nil || len(tm.tempDir) == 0 {
			return nil
		}
		return os.RemoveAll(tm.tempDir)
	})
	tm.addClosers(cr.TTY, cr.Querier, cr.Window)
	for _, dr := range cr.Drawers {
		tm.addClosers(dr)
	}
	runtime.SetFinalizer(tm, func(tc *Terminal) {
		// if tc == nil {return}
		_ = tc.Close()
	})

	return tm, nil
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
	return t.drawers[0].Draw(img, bounds, t.rsz, t)
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

var _ internal.Arger = (*args)(nil)

type args []string

func (a *args) Args() []string {
	if a == nil {
		return nil
	}
	return *a
}

func newArger(arguments []string) internal.Arger {
	a := args(arguments)
	return &a
}
