package term

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/go-errors/errors"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/wndws"
	"github.com/srlehn/termimg/wm"
	"golang.org/x/exp/slices"
)

// TermChecker must implement at least one of CheckExclude, CheckIs, CheckWindow.
// For passing on properties create a new Proprietor with NewProprietor.
// CheckExclude is a preliminary check if the to be checked terminal might be
// the matching terminal of the TermChecker. CheckExclude can set properties
// for later exclusion of CheckIs checks, etc, no ANSI querying shall be done during
// this stage.
// CheckIs is the final check, ANSI querying is allowed, if not prohibited by the
// TerminalCheckerInput properties.
// Alternatively a wm.Window can be compared against with CheckWindow.
//
//   - CheckExclude(TerminalCheckerInput) (mightBe bool, p Proprietor)
//   - CheckIs(Querier, TTY, TerminalCheckerInput) (is bool, p Proprietor)
//   - CheckWindow(wm.Window) (is bool, p Proprietor)
//
// A new TermChecker has to embed NewTermCheckerCore(name string) providing the
// TermChecker type identity and the Name() method.
//
// RegisterTermChecker(TermChecker) is used for the registration of the new TermChecker.
//
// Optional methods for setting specific implementations:
//
//   - TTY(pytName string, ci environ.Proprietor) (TTY, error)
//   - Querier(environ.Proprietor) Querier
//   - Surveyor(environ.Proprietor) PartialSurveyor
//   - Window(environ.Proprietor) (wm.Window, error)
//   - Args(environ.Proprietor) []string
//   - Exe(environ.Proprietor) string
type TermChecker interface {
	// TODO: implement all optional methods through the core and check for nil?
	// The following methods are implemented by embedded *termCheckerCore.
	Name() string
	CheckExclude(environ.Proprietor) (mightBe bool, p environ.Proprietor)
	CheckIsQuery(Querier, TTY, environ.Proprietor) (is bool, p environ.Proprietor)
	CheckIsWindow(wm.Window) (is bool, p environ.Proprietor)
	Check(qu Querier, tty TTY, inp environ.Proprietor) (is bool, p environ.Proprietor)
	NewTerminal(...Option) (*Terminal, error)
	Init(tc TermChecker) // called during registration
}

type termCheckerCore struct {
	name   string
	parent TermChecker
}

func (c *termCheckerCore) Name() string {
	if c == nil {
		return ``
	}
	return c.name
}

// combines CheckExclude and CheckIs
func (c *termCheckerCore) Check(qu Querier, tty TTY, inp environ.Proprietor) (is bool, p environ.Proprietor) {
	// TODO include CheckIsWindow?
	if c == nil || c.parent == nil {
		return false, nil
	}

	pr := environ.NewProprietor()
	mightBe, prCE := c.parent.CheckExclude(inp)
	if !mightBe {
		return false, nil
	}
	pr.Merge(prCE)

	_, avoidANSI := pr.Property(propkeys.AvoidANSI)
	if !avoidANSI {
		isTerm, prCI := c.parent.CheckIsQuery(qu, tty, inp)
		if !isTerm {
			return false, nil
		}
		pr.Merge(prCI)
	}

	return true, pr
}

const (
	CheckTermPassed = `passed`
	CheckTermFailed = `failed`
	CheckTermDummy  = `dummy` // promoted dummy core method
)

func (c *termCheckerCore) CheckExclude(environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+c.Name(), CheckTermDummy)
	return false, p
}
func (c *termCheckerCore) CheckIsQuery(Querier, TTY, environ.Proprietor) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+c.Name(), CheckTermDummy)
	return false, p
}
func (c *termCheckerCore) CheckIsWindow(wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	p.SetProperty(propkeys.CheckTermWindowIsPrefix+c.Name(), CheckTermDummy)
	return false, p
}

func NewTermCheckerCore(name string) TermChecker {
	if len(name) == 0 {
		return nil
	}
	return &termCheckerCore{name: name}
}

func (c *termCheckerCore) Init(tc TermChecker) {
	if c == nil {
		return
	}
	c.parent = tc
}

func (c *termCheckerCore) NewTerminal(opts ...Option) (*Terminal, error) {
	if c == nil {
		return nil, errors.New(internal.ErrNilReceiver)
	}
	tm := newDummyTerminal()
	overwriteEnv := false // likely already set by caller function (NewTerminal())
	if err := tm.SetOptions(append(opts, setInternalDefaults, setEnvAndMuxers(overwriteEnv))...); err != nil {
		return nil, err
	}
	composeManuallyStr, composeManually := tm.Property(propkeys.ManualComposition)
	if composeManually && composeManuallyStr == `true` {
		// manual composition
		tm.SetOptions(setTTYAndQuerier(nil))
		tm.surveyor = getSurveyor(tm.partialSurveyor, tm.proprietor)
		tm.closer = internal.NewCloser()
		tm.OnClose(func() error {
			if tm == nil || len(tm.tempDir()) == 0 {
				return nil
			}
			return os.RemoveAll(tm.tempDir())
		})
		tm.addClosers(tm.tty, tm.querier, tm.window)
		for _, dr := range tm.drawers {
			tm.addClosers(dr)
		}
		runtime.SetFinalizer(tm, func(t *Terminal) { _ = t.Close() })
		return tm, nil
	}
	if c.parent == nil {
		return nil, errors.New(`*termCheckerCore.Init was not called`)
	}
	var exe string
	if ex, okExe := c.parent.(interface {
		Exe(environ.Proprietor) string
	}); okExe {
		exe = ex.Exe(tm.proprietor)
	}
	var ar internal.Arger
	if arg, okArger := c.parent.(interface {
		Args(environ.Proprietor) []string
	}); okArger {
		ar = newArger(arg.Args(tm.proprietor))
	}
	if err := tm.SetOptions(setTTYAndQuerier(c)); err != nil {
		return nil, err
	}
	if tm.partialSurveyor == nil {
		if surver, okSurver := c.parent.(interface {
			Surveyor(environ.Proprietor) PartialSurveyor
		}); okSurver {
			tm.partialSurveyor = surver.Surveyor(tm.proprietor)
		}
		if tm.partialSurveyor == nil {
			tm.partialSurveyor = tm.partialSurveyorDefault
		}
	}
	var w wm.Window
	if tm.windowProvider != nil {
		w = tm.windowProvider(c.parent.CheckIsWindow, tm.proprietor)
	}
	if w == nil {
		if wdwer, okWdwer := c.parent.(interface {
			Window(environ.Proprietor) (wm.Window, error)
		}); okWdwer {
			wChk, err := wdwer.Window(tm.proprietor)
			if err == nil && wChk != nil {
				w = wChk
			}
		}
		if w == nil && tm.windowProviderDefault != nil {
			w = tm.windowProviderDefault(c.parent.CheckIsWindow, tm.proprietor)
		}
	}

	drCkInp := &drawerCheckerInput{
		Proprietor: tm.proprietor,
		Querier:    tm.querier,
		TTY:        tm.tty,
		w:          w,
		name:       c.parent.Name(),
	}
	drawers, err := DrawersFor(drCkInp)
	if err != nil {
		return nil, err
	}
	if len(drawers) == 0 {
		// TODO one drawer should always be provided!
		// drawers = []Drawer{&generic.DrawerGeneric{}} // TODO import cycle
		return nil, errors.New(`no drawers found`) // TODO rm
	}
	var lessFn func(i, j int) bool
	drawerMap := make(map[string]struct{})
	if _, isRemote := tm.proprietor.Property(propkeys.IsRemote); isRemote {
		lessFn = func(i, j int) bool {
			return slices.Index(drawersPriorityOrderedRemote, drawers[i].Name()) < slices.Index(drawersPriorityOrderedRemote, drawers[j].Name())
		}
		for _, drName := range drawersPriorityOrderedRemote {
			if len(drName) == 0 {
				continue
			}
			drawerMap[drName] = struct{}{}
		}
	} else {
		lessFn = func(i, j int) bool {
			return slices.Index(drawersPriorityOrderedLocal, drawers[i].Name()) < slices.Index(drawersPriorityOrderedLocal, drawers[j].Name())
		}
		for _, drName := range drawersPriorityOrderedLocal {
			if len(drName) == 0 {
				continue
			}
			drawerMap[drName] = struct{}{}
		}
	}
	drawersPrunedAndNew := make([]Drawer, 0, len(drawers))
	for _, dr := range drawers {
		if dr == nil {
			continue
		}
		if _, ok := drawerMap[dr.Name()]; ok {
			drNew := dr.New() // create new drawer instances
			if drNew == nil {
				continue
			}
			drawersPrunedAndNew = append(drawersPrunedAndNew, drNew)
		}
	}
	drawers = drawersPrunedAndNew
	sort.SliceStable(drawers, lessFn)

	if tm.resizer == nil {
		tm.resizer = ResizerDefault()
	}

	tm.surveyor = getSurveyor(tm.partialSurveyor, tm.proprietor)
	tm.arger = ar
	tm.window = w
	tm.drawers = drawers
	tm.closer = internal.NewCloser()
	tm.SetProperty(propkeys.TerminalName, c.parent.Name())
	tm.SetProperty(propkeys.Executable, exe)
	tm.OnClose(func() error {
		// last closer function
		tm = nil
		return nil
	})
	tm.OnClose(func() error {
		tempDir := tm.tempDir()
		if tm == nil || len(tempDir) == 0 {
			return nil
		}
		return os.RemoveAll(tempDir)
	})
	tm.addClosers(tm.tty, tm.querier, tm.windowProvider)
	for _, dr := range drawers {
		tm.addClosers(dr)
	}
	runtime.SetFinalizer(tm, func(tc *Terminal) { _ = tc.Close() })

	return tm, nil
}

////////////////////////////////////////////////////////////////////////////////

func termGenericPreCheck(pr environ.Proprietor) {
	if isRemotePreCheck(pr) {
		pr.SetProperty(propkeys.IsRemote, ``)
	}
	// TODO store map keys as exported internal vars
	pr.SetProperty(propkeys.RunsOnWine, fmt.Sprintf(`%v`, wndws.RunsOnWine()))
}

func termGenericCheck(tty TTY, qu Querier, pr environ.Proprietor) {
	if _, avoidANSI := pr.Property(propkeys.AvoidANSI); avoidANSI {
		return
	}
	_ = QueryDeviceAttributes(qu, tty, pr, pr)
	for _, spcl := range xtGetTCapSpecialStrs {
		_, _ = XTGetTCap(spcl, qu, tty, pr, pr)
	}
}

func isRemotePreCheck(e environ.Enver) bool {
	// TODO return Proprietor?
	if e == nil {
		return false
	}

	display, ok := e.LookupEnv(`DISPLAY`)
	if ok && len(display) > 0 {
		// TODO X11 doesn't use a unix socket for "localhost" only for "" (?)
		if host := strings.Split(display, `:`)[0]; len(host) == 0 || host == `localhost` {
			return false
		} else {
			return true
		}
	}
	sshConn, ok := e.LookupEnv(`SSH_CONNECTION`)
	if ok && len(sshConn) > 0 {
		return true
	}
	sshClient, ok := e.LookupEnv(`SSH_CLIENT`)
	if ok && len(sshClient) > 0 {
		return true
	}
	sshTTY, ok := e.LookupEnv(`SSH_TTY`)
	if ok && len(sshTTY) > 0 {
		return true
	}

	return false
}

////////////////////////////////////////////////////////////////////////////////

type DrawerCheckerInput interface {
	Name() string
	environ.Proprietor
	Querier
	TTY
	wm.Window // TODO remove Close()
}

var _ DrawerCheckerInput = (*drawerCheckerInput)(nil)

type drawerCheckerInput struct {
	environ.Proprietor
	Querier
	TTY
	w    wm.Window
	name string
}

func (in *drawerCheckerInput) Name() string {
	if in == nil {
		return ``
	}
	return in.name
}

var _ wm.Window = (*drawerCheckerInput)(nil)

func (in *drawerCheckerInput) WindowConn() wm.Connection {
	if in == nil {
		return nil
	}
	return in.w.WindowConn()
}
func (in *drawerCheckerInput) WindowFind() error {
	if in == nil {
		return errors.New(internal.ErrNilReceiver)
	}
	return in.w.WindowFind()
}
func (in *drawerCheckerInput) WindowType() string {
	if in == nil {
		return ``
	}
	return in.w.WindowType()
}
func (in *drawerCheckerInput) WindowName() string {
	if in == nil {
		return ``
	}
	return in.w.WindowName()
}
func (in *drawerCheckerInput) WindowClass() string {
	if in == nil {
		return ``
	}
	return in.w.WindowClass()
}
func (in *drawerCheckerInput) WindowInstance() string {
	if in == nil {
		return ``
	}
	return in.w.WindowInstance()
}
func (in *drawerCheckerInput) WindowID() uint64 {
	if in == nil {
		return 0
	}
	return in.w.WindowID()
}
func (in *drawerCheckerInput) WindowPID() uint64 {
	if in == nil {
		return 0
	}
	return in.w.WindowPID()
}
func (in *drawerCheckerInput) DeviceContext() uintptr {
	if in == nil {
		return 0
	}
	return in.w.DeviceContext()
}
func (in *drawerCheckerInput) Screenshot() (image.Image, error) {
	if in == nil {
		return nil, errors.New(internal.ErrNilReceiver)
	}
	return in.w.Screenshot()
}
