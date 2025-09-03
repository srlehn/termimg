package term

import (
	"fmt"
	"image"
	"log/slog"
	"maps"
	"math"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/queries"
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
	Window
}
*/

var _ interface {
	internal.Closer
	Surveyor
	Properties
	TTY
} = (*Terminal)(nil)

type (
	Properties     = environ.Properties
	Window         = wm.Window
	WindowProvider = wm.WindowProvider
	// unexport for struct embedding
	tty        = TTY
	querier    = Querier
	properties = Properties
	arger      = internal.Arger
	closer     = internal.Closer
)

type Terminal struct {
	tty
	querier
	properties // Data
	surveyor   SurveyorLight
	arger
	window Window
	closer
	drawers         []Drawer
	resizer         Resizer
	passages        mux.Muxers
	printMu         *sync.Mutex
	afterSetupFuncs []func(*Terminal)

	ttyDefault                              TTY
	querierDefault                          Querier
	ttyProv, ttyProvDefault                 ttyProvider
	partialSurveyor, partialSurveyorDefault PartialSurveyor
	windowProvider, windowProviderDefault   WindowProvider

	// resolution
	resTermInCellsW, resTermInCellsH uint
	resTermInPxlsW, resTermInPxlsH   uint
	resCellInPxlsW, resCellInPxlsH   float64
	// last window change
	// timeLastSizeCheck time.Time
	timeLastSizeCheck time.Time
	// window change directly prior to last resolution query
	timeResTermInCells time.Time
	timeResTermInPxls  time.Time
	timeResCellInPxls  time.Time

	logger *slog.Logger
}

// NewTerminal tries to recognize the terminal that manages the device ptyName.
// It will use suggestions provided by the optional TerminalChecker methods:
//
//   - TTY(ptyName string, ci environ.Proprietor) (TTY, error)
//   - Querier(environ.Proprietor) Querier
//   - Surveyor(environ.Proprietor) PartialSurveyor
//   - Window(environ.Proprietor) (Window, error)
//   - Args(environ.Proprietor) []string
//   - Exe(environ.Proprietor) string // alternative executable name if it differs from Name()
//
// "Enforced" Options have precedence over the TermCheckers suggestion.
func NewTerminal(opts ...Option) (*Terminal, error) {
	tm := newDummyTerminal()
	opts = append(opts, setInternalDefaults, setEnvAndMuxers(true))
	if err := tm.SetOptions(opts...); logx.IsErr(err, tm, slog.LevelError) {
		return nil, err
	}

	ttyTmp, quTmp, err := getTTYAndQuerier(tm, nil)
	if logx.IsErr(err, tm, slog.LevelError) {
		return nil, err
	}

	var checker TermChecker
	composeManuallyStr, composeManually := tm.Property(propkeys.ManualComposition)
	if !composeManually || composeManuallyStr != `true` {
		// find terminal checker
		chk, prChecker, err := findTermChecker(tm.properties, ttyTmp, quTmp)
		if logx.IsErr(err, tm, slog.LevelInfo) {
			return nil, err
		}
		tm.properties.MergeProperties(prChecker)
		checker = chk
	} else {
		checker = &termCheckerCore{} // dummy
	}
	// terminal specific settings
	tm, err = checker.NewTerminal(replaceTerminal(tm))
	if logx.IsErr(err, tm, slog.LevelInfo) {
		return nil, err
	}

	return tm, nil
}
func newDummyTerminal() *Terminal {
	tm := &Terminal{
		properties: environ.NewProperties(),
		printMu:    &sync.Mutex{},
		closer:     internal.NewCloser(),
	}
	tm.SetProperty(propkeys.EnvIsLoaded, `merge`)
	return tm
}

func getTTYAndQuerier(tm *Terminal, tc *termCheckerCore) (TTY, Querier, error) {
	// tty order: tm.tty, tm.ttyProv, tc.TTY, tm.ttyDefault, tm.ttyProvDefault
	// querier order: tm.querier, tc.Querier, tm.querierDefault

	setDefaultTTY := func(t TTY, ttyProv ttyProvider) (TTY, error) {
		if t != nil {
			return t, nil
		}
		if ttyProv == nil {
			return nil, errors.New(`nil tty provider`)
		}
		tt, err := ttyProv(tm.ptyName())
		if err != nil {
			return nil, err
		}
		if tt == nil {
			return nil, errors.New(`nil tty received from tty provider`)
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
				TTY(pytName string, ci Properties) (TTY, error)
			}); okTTYProv {
				tty, err := ttyProv.TTY(tm.ptyName(), tm.properties)
				if err != nil {
					errs = append(errs, err)
				} else if tty != nil {
					if tm.properties != nil {
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
			errTTYRet = errors.WrapPrefix(errTTYRet, `no/failed tty provision;`, 0)
		} else {
			errTTYRet = errors.New(`no tty provided`)
		}
		return nil, nil, errTTYRet
	}

	var quTemp Querier
	if tm.querier != nil {
		quTemp = tm.querier
	} else {
		if tc != nil {
			if querier, okQuerier := tc.parent.(interface {
				Querier(Properties) Querier
			}); okQuerier {
				quTemp = querier.Querier(tm.properties)
			}
		}
		if quTemp == nil && tm.querierDefault != nil {
			quTemp = tm.querierDefault
		} else {
			return nil, nil, errors.New(`nil querier`)
		}
	}

	return ttyTemp, quTemp, nil
}

func findTermChecker(env Properties, tty TTY, qu Querier) (tc TermChecker, _ Properties, e error) {
	var ttyTemp TTY
	if tty == nil || qu == nil {
		return RegisteredTermChecker(consts.TermGenericName), nil, errors.NilParam()
	}
	if env == nil {
		return RegisteredTermChecker(consts.TermGenericName), nil, nil
	}
	pr := environ.NewProperties()
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
		pr.MergeProperties(prE)
	}

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		var errs []error
		err := errors.New(r)
		errs = append(errs, err)
		if ttyTemp != nil {
			err := ttyTemp.Close()
			errs = append(errs, err)
		}
		if tty != nil {
			err := tty.Close()
			errs = append(errs, err)
		}
		tc = RegisteredTermChecker(consts.TermGenericName)
		e = errors.New(errors.Join(errs...))
	}()
	termGenericCheck(tty, qu, pr)
	env.MergeProperties(pr)
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
				case consts.CheckTermPassed:
					break
				case consts.CheckTermDummy:
					exclEnvSkipped = true
				case consts.CheckTermFailed:
					fallthrough
				default:
					pr.SetProperty(propkeys.CheckTermCompletePrefix+tchk.Name(), consts.CheckTermFailed)
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
					prI.ExportProperties()
					passedIs, okPassedIs := prI.Property(propkeys.CheckTermQueryIsPrefix + tchkName)
					exclSolePassed = okPassedIs && passedIs == consts.CheckTermDummy
				}
				if !exclSolePassed {
					pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, consts.CheckTermFailed)
					continue
				}
			}
			pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, consts.CheckTermPassed)
			pr.MergeProperties(prI)
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
			case consts.CheckTermPassed, consts.CheckTermDummy:
				pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, consts.CheckTermPassed)
			case consts.CheckTermFailed:
				fallthrough
			default:
				pr.SetProperty(propkeys.CheckTermCompletePrefix+tchkName, consts.CheckTermFailed)
				continue
			}
		}
	}
	for _, tchk := range terminalCheckersAll {
		tchkName := tchk.Name()
		passed, completed := pr.Property(propkeys.CheckTermCompletePrefix + tchkName)
		if completed && passed == consts.CheckTermPassed {
			prTmChkMap[tchkName] = struct{}{}
		}
	}

	// single out a specific terminal checker if possible
	var termMatchName string
	var useGenericTermIfUncertain bool = true
	if useStr, okUse := pr.Property(propkeys.UseFallback); okUse && useStr == `false` {
		useGenericTermIfUncertain = false
	}
	switch l := len(prTmChkMap); {
	case l == 0:
		termMatchName = consts.TermGenericName
	case l == 1:
		termMatchName = slices.Collect(maps.Keys(prTmChkMap))[0]
	case l == 2:
		// assume `generic` is contained
		for k := range prTmChkMap {
			if k != consts.TermGenericName {
				termMatchName = k
				break
			}
		}
	case l > 2:
		// TODO: if windowQuery == nil then try to find corresponding window and recheck
		if useGenericTermIfUncertain {
			termMatchName = consts.TermGenericName
		} else {
			termNames := slices.Collect(maps.Keys(prTmChkMap))
			sort.Strings(termNames)
			termNamesStr := strings.Join(termNames, ` `)
			return RegisteredTermChecker(consts.TermGenericName), nil, errors.Errorf(`more than 1 terminal check matched: %s`, termNamesStr)
		}
	}
	_, okMatch := prTmChkMap[termMatchName]
	if !okMatch {
		termMatchName = consts.TermGenericName
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
		return RegisteredTermChecker(consts.TermGenericName), nil, errors.New(`no matching terminal was found`)
	}

	return checker, pr, nil
}

func (t *Terminal) Name() string {
	if t == nil || t.properties == nil {
		return ``
	}
	termName, _ := t.Property(propkeys.TerminalName)
	return termName
}

func (t *Terminal) ptyName() string {
	if t == nil || t.properties == nil {
		return ``
	}
	ptyName, _ := t.Property(propkeys.PTYName)
	return ptyName
}

func (t *Terminal) tempDir() string {
	if t == nil || t.properties == nil {
		return ``
	}
	tempDir, _ := t.Property(propkeys.TempDir)
	return tempDir
}

func (t *Terminal) Query(qs string, p Parser) (string, error) {
	repl, err := t.query(qs, p)
	args := []any{`query`, qs, `reply`, repl}
	if err != nil {
		args = append(args, `error`, err)
	}
	logx.Info(`terminal query`, t, args...)
	return repl, err
}

func (t *Terminal) query(qs string, p Parser) (_ string, err error) {
	if t == nil {
		return ``, errors.NilReceiver()
	}
	if t.tty == nil {
		return ``, errors.New(`nil tty`)
	}
	if t.querier == nil {
		return ``, errors.New(`nil querier`)
	}
	if t.properties == nil {
		return ``, errors.New(`nil proprietor`)
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(r)
		}
	}()
	if _, avoidANSI := t.Property(propkeys.AvoidANSI); avoidANSI {
		return ``, errors.New(consts.ErrPlatformNotSupported)
	}
	return t.querier.Query(qs, t.tty, p)
}

// CreateTemp ...
func (t *Terminal) CreateTemp(pattern string) (*os.File, error) {
	if t == nil {
		return nil, errors.NilReceiver()
	}
	tempDir := t.tempDir()
	if len(tempDir) == 0 {
		dir, err := os.MkdirTemp(``, consts.LibraryName+`.`)
		if logx.IsErr(err, t, slog.LevelInfo) {
			return nil, err
		}
		t.SetProperty(propkeys.TempDir, tempDir)
		onCloseFunc := func() error {
			return os.RemoveAll(dir)
		}
		t.OnClose(onCloseFunc)
	}

	f, err := os.CreateTemp(tempDir, pattern)
	if logx.IsErr(err, t, slog.LevelInfo) {
		return nil, err
	}
	onCloseFunc := func() error { return errors.New(errors.Join(f.Close(), os.Remove(f.Name()))) }
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
		return 0, errors.NilReceiver()
	}
	n, err := t.WriteString(t.passages.Wrap(fmt.Sprintf(format, a...)))
	if logx.IsErr(err, t, slog.LevelInfo) {
		return n, errors.New(err)
	}
	return n, nil
}

func (t *Terminal) Write(p []byte) (n int, err error) {
	if t == nil {
		return 0, errors.NilReceiver()
	}
	if t.tty == nil {
		return 0, errors.New(`nil tty`)
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(r)
		}
	}()
	t.printMu.Lock()
	defer t.printMu.Unlock()
	return t.tty.Write(p)
}
func (t *Terminal) WriteString(s string) (n int, err error) {
	b := unsafe.Slice(unsafe.StringData(s), len(s))
	return t.Write(b)
}

func (t *Terminal) Draw(img image.Image, bounds image.Rectangle) error {
	return Draw(img, bounds, t, nil)
}

// CellScale returns a cell size for pixel size <ptSrcPx> to
// the cell size <ptDstCl> while maintaining the scale.
// With no passed 0 side length values, the largest subarea is returned.
// With one passed 0 side length value, the other side length will be fixed.
// With two passed 0 side length values, pixels in source and destination area at the same position correspond to each other.
func (t *Terminal) CellScale(ptSrcPx, ptDstCl image.Point) (ptSrcCl image.Point, _ error) {
	if err := errors.NilReceiver(t); err != nil {
		return image.Point{}, err
	}
	cpw, cph, err := t.CellSize()
	if logx.IsErr(err, t, slog.LevelInfo) {
		return image.Point{}, err
	}
	if cpw < 1 || cph < 1 {
		return image.Point{}, errors.New(`received invalid terminal cell size`)
	}
	if ptDstCl.X == 0 {
		if ptDstCl.Y == 0 {
			ptSrcCl = image.Point{
				X: roundInf(float64(ptSrcPx.X) / cpw),
				Y: roundInf(float64(ptSrcPx.Y) / cph),
			}
		} else {
			ptSrcCl.Y = ptDstCl.Y
			ptSrcCl.X = roundInf((float64(ptSrcPx.X) * float64(cph) * float64(ptDstCl.Y)) / (float64(ptSrcPx.Y) * float64(cpw)))
		}
	} else {
		ptSrcCl.X = ptDstCl.X
		yScaled := roundInf((float64(ptSrcPx.Y) * float64(cpw) * float64(ptDstCl.X)) / (float64(ptSrcPx.X) * float64(cph)))
		if ptDstCl.Y == 0 {
			ptSrcCl.Y = yScaled
		} else {
			if yScaled <= ptDstCl.Y {
				ptSrcCl.Y = yScaled
			} else {
				ptSrcCl.Y = ptDstCl.Y
				ptSrcCl.X = roundInf((float64(ptSrcPx.X) * float64(cph) * float64(ptDstCl.Y)) / (float64(ptSrcPx.Y) * float64(cpw)))
			}
		}
	}
	return ptSrcCl, nil
}

func (t *Terminal) watchWINCHStart() error {
	if t == nil || t.surveyor == nil {
		return errors.NilReceiver()
	}
	winch, closeFunc, err := t.surveyor.WatchResizeEventsStart(t.tty, t.querier, t.window, t.properties)
	if logx.IsErr(err, t, slog.LevelInfo) {
		return err
	}
	t.closer.OnClose(closeFunc)
	go func() {
		for {
			if winch == nil {
				return
			}
			res, ok := <-winch
			if !ok {
				return
			}
			tmNow := time.Now()
			t.timeLastSizeCheck = tmNow
			if res.TermInCellsW > 0 && res.TermInCellsH > 0 {
				t.resTermInCellsW = res.TermInCellsW
				t.resTermInCellsH = res.TermInCellsH
				t.timeResTermInCells = tmNow
			}
			if res.TermInPxlsW > 0 && res.TermInPxlsH > 0 {
				t.resTermInPxlsW = res.TermInPxlsW
				t.resTermInPxlsH = res.TermInPxlsH
				t.timeResTermInPxls = tmNow
			}
			if t.resCellInPxlsW >= 1 && t.resCellInPxlsH >= 1 {
				t.resCellInPxlsW = res.CellInPxlsW
				t.resCellInPxlsH = res.CellInPxlsH
				t.timeResCellInPxls = tmNow
			}
			// TODO perhaps calculate cell resolution if missing based on terminal resolution (cells & pixels)
		}
	}()
	return nil
}
func (t *Terminal) watchWINCHStop() error {
	if t == nil || t.surveyor == nil {
		return errors.NilReceiver()
	}
	return t.surveyor.WatchResizeEventsStop()
}

func (t *Terminal) Logger() *slog.Logger {
	if t == nil {
		return nil
	}
	return t.logger
}

func (t *Terminal) logDebug(msg string, args ...any) {
	if t == nil {
		return
	}
	logx.Log(msg, t.logger, slog.LevelDebug, 3, args...)
}
func (t *Terminal) logInfo(msg string, args ...any) {
	if t == nil {
		return
	}
	logx.Log(msg, t.logger, slog.LevelInfo, 3, args...)
}
func (t *Terminal) logWarn(msg string, args ...any) {
	if t == nil {
		return
	}
	logx.Log(msg, t.logger, slog.LevelWarn, 3, args...)
}
func (t *Terminal) logError(msg string, args ...any) {
	if t == nil {
		return
	}
	logx.Log(msg, t.logger, slog.LevelError, 3, args...)
}

// round away from zero (toward infinity)
func roundInf(f float64) int {
	if f > 0 {
		return int(math.Ceil(f))
	}
	return int(math.Floor(f))
}

// If lineCnt == 0 scroll until cursor is out of view.
func (t *Terminal) Scroll(lineCnt int) error {
	if t == nil {
		return errors.NilParam()
	}
	_, tch, err := t.SizeInCells()
	if logx.IsErr(err, t, slog.LevelInfo) {
		return err
	}
	if tch == 0 {
		return errors.New(`received null terminal size in cells`)
	}
	var avoidCS1IndexSuffix bool = true
	{
		prop, ok := t.Property(propkeys.TerminalPrefix + t.Name() + propkeys.TerminalAvoidCS1IndexSuffix)
		avoidCS1IndexSuffix = ok && prop == `true`
		if avoidCS1IndexSuffix && lineCnt < 0 {
			return errors.New(`C1 RI (Reverse Index) not supported`)
		}
	}
	switch {
	case lineCnt == 0:
		_, y, err := t.Cursor()
		if logx.IsErr(err, t, slog.LevelInfo) {
			return err
		}
		lineCnt = int(y)
		fallthrough
	case lineCnt > 0:
		if err := t.SetCursor(0, tch); logx.IsErr(err, t, slog.LevelInfo) {
			return err
		}
		if avoidCS1IndexSuffix {
			t.Printf(`%s`, strings.Repeat("\n", lineCnt))
		} else {
			t.Printf(`%s`, strings.Repeat(queries.IND, lineCnt)) // C1 Index - moves down same column
			// tm2.Printf(queries.CSI+`%dB`, lineCnt) // CUD - Cursor Down
			// tm2.Printf(queries.CSI+`%dT`, lineCnt) // SD - Scroll Down
		}
	case lineCnt < 0:
		if err := t.SetCursor(0, 0); logx.IsErr(err, t, slog.LevelInfo) {
			return err
		}
		t.Printf(`%s`, strings.Repeat(queries.RI, -lineCnt)) // C1 Index - moves up same column
		// tm2.Printf(queries.CSI+`%dA`, lineCnt) // CUU - Cursor Up
		// tm2.Printf(queries.CSI+`%dS`, lineCnt) // SU - Scroll Up
	}
	return nil
}

func (t *Terminal) CellSize() (width, height float64, _ error) {
	if t == nil {
		return 0, 0, errors.NilReceiver()
	}
	if t.surveyor == nil {
		return 0, 0, errors.New(`nil surveyor`)
	}
	logMsg := `(*term.Terminal).CellSize()`
	logArgs := []any{`last-sigwinch`, t.timeLastSizeCheck}
	var cpw, cph float64
	var query bool
	var err error
	// TODO add prop key for disabling cache
	if !t.timeResCellInPxls.IsZero() && t.timeResCellInPxls.Equal(t.timeLastSizeCheck) {
		cpw, cph = t.resCellInPxlsW, t.resCellInPxlsH
	} else {
		cpw, cph, err = t.surveyor.CellSize(t.tty, t.querier, t.window, t.properties)
		query = true
		if logx.IsErr(err, t, slog.LevelInfo, logArgs...) {
			logArgs = append(logArgs, []any{"cpw", cpw, "cph", cph, "query", query, "err", err}...)
			t.logError(logMsg, logArgs...)
			return 0, 0, err
		}
		tmNow := time.Now()
		t.timeLastSizeCheck = tmNow
		t.timeResCellInPxls = tmNow
	}
	logArgs = append(logArgs, []any{"cpw", cpw, "cph", cph, "query", query}...)
	if cpw < 1 || cph < 1 {
		err := errors.New(`CellSize failed`)
		logArgs = append(logArgs, []any{"err", err}...)
		t.logError(logMsg, logArgs...)
		return 0, 0, err
	}
	if !t.timeLastSizeCheck.IsZero() {
		t.resCellInPxlsW = cpw
		t.resCellInPxlsH = cph
		t.timeResCellInPxls = t.timeLastSizeCheck
	}
	if query {
		t.logInfo(logMsg, logArgs...)
	}
	return cpw, cph, nil
}

func (t *Terminal) SizeInCells() (width, height uint, err error) {
	if t == nil {
		return 0, 0, errors.NilReceiver()
	}
	if t.surveyor == nil {
		return 0, 0, errors.New(`nil surveyor`)
	}
	// TODO add prop key for disabling cache
	if !t.timeResTermInCells.IsZero() && t.timeResTermInCells.Equal(t.timeLastSizeCheck) {
		return t.resTermInCellsW, t.resTermInCellsH, nil
	}
	w, h, err := t.surveyor.SizeInCells(t.tty, t.querier, t.window, t.properties)
	if !t.timeLastSizeCheck.IsZero() && err == nil && w > 0 && h > 0 {
		t.resTermInCellsW = w
		t.resTermInCellsH = h
		t.timeResTermInCells = t.timeLastSizeCheck
	}
	return w, h, err
}

func (t *Terminal) SizeInPixels() (width, height uint, err error) {
	if t == nil {
		return 0, 0, errors.NilReceiver()
	}
	if t.surveyor == nil {
		return 0, 0, errors.New(`nil surveyor`)
	}
	// TODO add prop key for disabling cache
	if !t.timeResTermInPxls.IsZero() && t.timeResTermInPxls.Equal(t.timeLastSizeCheck) {
		return t.resTermInPxlsW, t.resTermInPxlsH, nil
	}
	w, h, err := t.surveyor.SizeInPixels(t.tty, t.querier, t.window, t.properties)
	if !t.timeLastSizeCheck.IsZero() && err == nil && w > 0 && h > 0 {
		t.resTermInPxlsW = w
		t.resTermInPxlsH = h
		t.timeResTermInPxls = t.timeLastSizeCheck
	}
	return w, h, err
}

func (t *Terminal) Cursor() (xPosCells, yPosCells uint, err error) {
	if t == nil {
		return 0, 0, errors.NilReceiver()
	}
	if t.surveyor == nil {
		return 0, 0, errors.New(`nil surveyor`)
	}
	return t.surveyor.Cursor(t.tty, t.querier, t.window, t.properties)
}

func (t *Terminal) SetCursor(xPosCells, yPosCells uint) (err error) {
	if t == nil {
		return errors.NilReceiver()
	}
	if t.surveyor == nil {
		return errors.New(`nil surveyor`)
	}
	return t.surveyor.SetCursor(xPosCells, yPosCells, t.tty, t.querier, t.window, t.properties)
}

func (t *Terminal) Env() environ.Enver {
	if t == nil || t.properties == nil {
		return nil
	}
	return environ.EnvToProperties(t.properties.Environ())
}

// default
func init() { wm.SetImpl(wminternal.DummyImpl()) }

func (t *Terminal) Window() Window {
	if t == nil {
		return nil
	}
	return t.window
}

func (t *Terminal) Resizer() Resizer { return t.resizer }
func (t *Terminal) NewCanvas(bounds image.Rectangle) (*Canvas, error) {
	err := errors.NilReceiver(t)
	if logx.IsErr(err, t, slog.LevelInfo) {
		return nil, err
	}
	c := &Canvas{
		terminal: t,
		closed:   make(chan struct{}),
	}
	err = c.SetCellArea(bounds)
	if logx.IsErr(err, t, slog.LevelInfo) {
		return nil, err
	}
	t.AddClosers(c)
	return c, nil
}

////////////////////////////////////////////////////////////////////////////////

type ttyProvider func(ptyName string) (TTY, error)

func newTTYProvider[T TTY, F func(ptyName string) (T, error)](ttyProv F) ttyProvider {
	if ttyProv == nil {
		return nil
	}
	return func(ptyName string) (TTY, error) { return ttyProv(ptyName) }
}

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
