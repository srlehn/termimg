package term

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"sort"
	"strings"
	"syscall"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/linux"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/wndws"
	"github.com/srlehn/termimg/wm"
)

// TermChecker must implement at least one of CheckExclude, CheckIs, CheckWindow.
// For passing on properties create a new Proprietor with NewProprietor.
// CheckExclude is a preliminary check if the to be checked terminal might be
// the matching terminal of the TermChecker. CheckExclude can set properties
// for later exclusion of CheckIs checks, etc, no ANSI querying shall be done during
// this stage.
// CheckIs is the final check, ANSI querying is allowed, if not prohibited by the
// TerminalCheckerInput properties.
// Alternatively a Window can be compared against with CheckWindow.
//
//   - CheckExclude(TerminalCheckerInput) (mightBe bool, p Proprietor)
//   - CheckIs(Querier, TTY, TerminalCheckerInput) (is bool, p Proprietor)
//   - CheckWindow(Window) (is bool, p Proprietor)
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
//   - Window(environ.Proprietor) (Window, error)
//   - Args(environ.Proprietor) []string
//   - Exe(environ.Proprietor) string
type TermChecker interface {
	// TODO: implement all optional methods through the core and check for nil?
	// The following methods are implemented by embedded *termCheckerCore.
	Name() string
	CheckExclude(Properties) (mightBe bool, p Properties)
	CheckIsQuery(Querier, TTY, Properties) (is bool, p Properties)
	CheckIsWindow(Window) (is bool, p Properties)
	Check(qu Querier, tty TTY, inp Properties) (is bool, p Properties)
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
func (c *termCheckerCore) Check(qu Querier, tty TTY, inp Properties) (is bool, p Properties) {
	// TODO include CheckIsWindow?
	if c == nil || c.parent == nil {
		return false, nil
	}

	pr := environ.NewProperties()
	mightBe, prCE := c.parent.CheckExclude(inp)
	if !mightBe {
		return false, nil
	}
	pr.MergeProperties(prCE)

	_, avoidANSI := pr.Property(propkeys.AvoidANSI)
	if !avoidANSI {
		isTerm, prCI := c.parent.CheckIsQuery(qu, tty, inp)
		if !isTerm {
			return false, nil
		}
		pr.MergeProperties(prCI)
	}

	return true, pr
}

func (c *termCheckerCore) CheckExclude(Properties) (mightBe bool, p Properties) {
	p = environ.NewProperties()
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+c.Name(), consts.CheckTermDummy)
	return false, p
}
func (c *termCheckerCore) CheckIsQuery(Querier, TTY, Properties) (is bool, p Properties) {
	p = environ.NewProperties()
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+c.Name(), consts.CheckTermDummy)
	return false, p
}
func (c *termCheckerCore) CheckIsWindow(Window) (is bool, p Properties) {
	p = environ.NewProperties()
	p.SetProperty(propkeys.CheckTermWindowIsPrefix+c.Name(), consts.CheckTermDummy)
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
		return nil, errors.NilReceiver()
	}
	tm := newDummyTerminal()
	overwriteEnv := false // likely already set by caller function (NewTerminal())
	opts = append(opts, setInternalDefaults, setEnvAndMuxers(overwriteEnv))
	if err := tm.SetOptions(opts...); logx.IsErr(err, tm, slog.LevelInfo) {
		return nil, err
	}
	logx.Info(`terminal auto-detection`, tm, `terminal`, c.Name())
	composeManuallyStr, composeManually := tm.Property(propkeys.ManualComposition)
	if composeManually && composeManuallyStr == `true` {
		// manual composition
		tm.SetOptions(setTTYAndQuerier(nil))
		tm.surveyor = getSurveyor(tm.partialSurveyor, tm.properties)
		if tm.closer == nil {
			tm.closer = internal.NewCloser()
		}
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
		Exe(Properties) string
	}); okExe {
		exe = ex.Exe(tm.properties)
	}
	var ar internal.Arger
	if arg, okArger := c.parent.(interface {
		Args(Properties) []string
	}); okArger {
		ar = newArger(arg.Args(tm.properties))
	}
	if err := tm.SetOptions(setTTYAndQuerier(c)); logx.IsErr(err, tm, slog.LevelInfo) {
		return nil, err
	}
	var w Window
	if tm.windowProvider != nil {
		w = tm.windowProvider(c.parent.CheckIsWindow, tm.properties)
	}
	if w == nil {
		if wdwer, okWdwer := c.parent.(interface {
			Window(Properties) (Window, error)
		}); okWdwer {
			wChk, err := wdwer.Window(tm.properties)
			if err == nil && wChk != nil {
				w = wChk
			}
		}
		if w == nil && tm.windowProviderDefault != nil {
			w = tm.windowProviderDefault(c.parent.CheckIsWindow, tm.properties)
		}
	}
	if tm.partialSurveyor == nil {
		if surver, okSurver := c.parent.(interface {
			Surveyor(Properties) PartialSurveyor
		}); okSurver {
			tm.partialSurveyor = surver.Surveyor(tm.properties)
		}
		if tm.partialSurveyor == nil {
			tm.partialSurveyor = tm.partialSurveyorDefault
			if tm.partialSurveyor == nil {
				tm.partialSurveyor = DefaultSurveyor()
			}
		}
	}

	drCkInp := &drawerCheckerInput{
		Properties: tm.properties,
		Querier:    tm.querier,
		TTY:        tm.tty,
		w:          w,
		name:       c.parent.Name(),
	}
	drawers, drProps, err := drawersFor(drCkInp)
	if logx.IsErr(err, tm, slog.LevelInfo) {
		return nil, err
	}
	if len(drawers) == 0 {
		// TODO one drawer should always be provided!
		// drawers = []Drawer{&generic.DrawerGeneric{}} // TODO import cycle
		return nil, errors.New(`no drawers found`) // TODO rm
	}
	tm.MergeProperties(drProps)
	var lessFn func(i, j int) bool
	drawerMap := make(map[string]struct{})
	if _, isRemote := tm.properties.Property(propkeys.IsRemote); isRemote {
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

	tm.surveyor = getSurveyor(tm.partialSurveyor, tm.properties)
	tm.arger = ar
	tm.window = w
	tm.drawers = drawers
	if tm.closer == nil {
		tm.closer = internal.NewCloser()
	}
	tm.SetProperty(propkeys.TerminalName, c.parent.Name())
	if len(exe) > 0 {
		tm.SetProperty(propkeys.Executable, exe)
	}
	if tm.Logger() != nil {
		drs := make([]string, 0, len(tm.Drawers()))
		for _, dr := range tm.Drawers() {
			drs = append(drs, dr.Name())
		}
		drsStr := strings.Join(drs, `,`)
		logx.Info(`drawer auto-detection`, tm, `drawers`, drsStr)
	}

	_ = tm.watchWINCHStart()

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
	tm.addClosers(tm.tty, tm.querier, tm.window)
	for _, dr := range drawers {
		tm.addClosers(dr)
	}
	runtime.SetFinalizer(tm, func(tc *Terminal) { _ = tc.Close() })
	if noCleanUpOnInterrupt, ok := tm.Property(propkeys.NoCleanUpOnInterrupt); !ok || noCleanUpOnInterrupt != `true` {
		if internal.IsDefaultTTY(tm.ptyName()) {
			sigs := make(chan os.Signal, 1)
			signal.Notify(
				sigs,
				syscall.SIGHUP,  // 1 // TODO does tmux pass this?
				os.Interrupt,    // 2
				syscall.SIGQUIT, // 3
				syscall.SIGABRT, // 6
				syscall.SIGTERM, // 15
			)
			go func() {
				sig := <-sigs
				logx.Info(`terminal was interrupted`, tm, `signal`, sig.String())
				tm.Close()
				signal.Stop(sigs)
				close(sigs)
			}()
		}
	}

	for _, f := range tm.afterSetupFuncs {
		if f == nil {
			continue
		}
		f(tm)
	}

	return tm, nil
}

////////////////////////////////////////////////////////////////////////////////

func termGenericPreCheck(pr Properties) {
	if isRemotePreCheck(pr) {
		pr.SetProperty(propkeys.IsRemote, `true`)
	}
	// TODO store map keys as exported internal vars
	pr.SetProperty(propkeys.RunsOnWine, fmt.Sprintf(`%v`, wndws.RunsOnWine()))
	tty, ok := pr.Property(propkeys.TerminalTTY)
	if ok && len(tty) > 0 {
		f, err := os.Open(tty)
		if err == nil && f != nil {
			mode, isLinuxConsole, _ := linux.KDGetMode(f.Fd())
			pr.SetProperty(propkeys.IsLinuxConsole, fmt.Sprintf(`%v`, isLinuxConsole))
			if isLinuxConsole {
				pr.SetProperty(propkeys.LinuxConsoleMode, fmt.Sprintf(`%v`, mode))
			}
		}
	}
}

func termGenericCheck(tty TTY, qu Querier, pr Properties) {
	if _, avoidANSI := pr.Property(propkeys.AvoidANSI); avoidANSI {
		return
	}
	_ = QueryDeviceAttributes(qu, tty, pr, pr)
	if _, avoidTCap := pr.Property(propkeys.AvoidTCap); !avoidTCap {
		for _, spcl := range xtGetTCapSpecialStrs {
			_, _ = xtGetTCap(spcl, qu, tty, pr, pr)
		}
	}
	_, _ = xtVersion(qu, tty, pr, pr)
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
	Properties
	Querier
	TTY
	Window // TODO remove Close()
}

var _ DrawerCheckerInput = (*drawerCheckerInput)(nil)

type drawerCheckerInput struct {
	Properties
	Querier
	TTY
	w    Window
	name string
}

func (in *drawerCheckerInput) Name() string {
	if in == nil {
		return ``
	}
	return in.name
}

var _ Window = (*drawerCheckerInput)(nil)

func (in *drawerCheckerInput) WindowConn() wm.Connection {
	if in == nil {
		return nil
	}
	return in.w.WindowConn()
}
func (in *drawerCheckerInput) WindowFind() error {
	if in == nil {
		return errors.NilReceiver()
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
		return nil, errors.NilReceiver()
	}
	return in.w.Screenshot()
}
