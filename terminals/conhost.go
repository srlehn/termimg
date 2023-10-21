//go:build windows

package terminals

import (
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/lxn/win"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/internal/wndws"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// ConHost
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerConHost{TermChecker: term.NewTermCheckerCore(termNameConHost)})
}

const termNameConHost = `conhost`

var _ term.TermChecker = (*termCheckerConHost)(nil)

type termCheckerConHost struct {
	term.TermChecker
}

func (t *termCheckerConHost) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameConHost, consts.CheckTermFailed)
		return false, p
	}
	// cmd.exe probably runs in conhost
	v, ok := pr.LookupEnv(`PROMPT`)
	if ok && len(v) > 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameConHost, consts.CheckTermPassed)
		p.SetProperty(propkeys.AvoidANSI, ``)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameConHost, consts.CheckTermFailed)
	return false, p
}

func (t *termCheckerConHost) Windower(pr environ.Properties) (wm.Window, error) {
	return getConHostWindow()
}
func (t *termCheckerConHost) Surveyor(pr environ.Properties) term.PartialSurveyor {
	return &surveyorConhost{}
}

////////////////////////////////////////////////////////////////////////////////

var _ wm.Window = (*windowConHost)(nil)

type windowConHost struct {
	wminternal.WindowCore
	hwnd uintptr
	hdc  uintptr
	internal.Closer
}

func (w *windowConHost) WindowHandle() uintptr {
	if w == nil {
		return 0
	}
	return w.hwnd
}
func (w *windowConHost) DeviceContext() uintptr {
	if w == nil {
		return 0
	}
	return w.hdc
}
func getConHostWindow() (wm.Window, error) {
	w := &windowConHost{
		Closer: internal.NewCloser(),
	}
	w.hwnd = uintptr(win.GetConsoleWindow())
	if w.hwnd == 0 {
		return nil, errors.New("GetConsoleWindow failed")
	}
	w.hdc = uintptr(win.GetDC(win.HWND(w.hwnd)))
	if w.hdc == 0 {
		return nil, errors.New("CreateCompatibleDC failed")
	}
	w.OnClose(func() error { _ = win.DeleteDC(win.HDC(w.hdc)); return nil })
	return w, nil
}

////////////////////////////////////////////////////////////////////////////////

type surveyorConhost struct {
	term.SurveyorNoANSI
	conOutHdl windows.Handle
	internal.Closer
}

// CellSize - cell size in pixels
func (s *surveyorConhost) CellSize(term.TTY) (width, height float64, err error) {
	if s == nil {
		return 0, 0, errors.New(consts.ErrNilReceiver)
	}
	if err := s.init(); err != nil {
		return 0, 0, err
	}
	tmInfo, err := wndws.GetCurrentConsoleFont(s.conOutHdl)
	if err != nil {
		return 0, 0, err
	}
	if tmInfo == nil {
		return 0, 0, errors.New(`GetCurrentConsoleFont failed`)
	}
	return float64(tmInfo.DwFontSize.X), float64(tmInfo.DwFontSize.Y), nil
}

// SizeInCells - terminal size in cells
func (s *surveyorConhost) SizeInCells(tty term.TTY) (widthCells, heightCells uint, err error) {
	if tty == nil {
		return 0, 0, errors.New(`nil tty`)
	}
	// implemented by gotty.ttyMattN
	szr, ok := tty.(interface {
		Size() (tcw, tch int, e error)
	})
	if !ok {
		return 0, 0, errors.New(`tty has no Size method`)
	}
	tcw, tch, err := szr.Size()
	if err != nil {
		return 0, 0, err
	}
	if tcw <= 0 || tch <= 0 {
		return 0, 0, errors.New(`tty.Size failed`)
	}
	return uint(tcw), uint(tch), nil
}

func (s *surveyorConhost) init() error {
	if s == nil {
		return errors.New(consts.ErrNilReceiver)
	}
	if s.Closer == nil {
		s.Closer = internal.NewCloser()
	}
	if s.conOutHdl != 0 {
		// already initiated
		return nil
	}
	// TODO use s.filename?
	conOutUTF16Ptr, err := syscall.UTF16PtrFromString("CONOUT$")
	if err != nil {
		return errors.New(err)
	}
	// hConsoleOutput := windows.Stdout
	// https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew#consoles
	conOutHdl, err := windows.CreateFile(
		conOutUTF16Ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		// windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return err
	}
	if conOutHdl == 0 {
		return errors.New(`windows.CreateFile failed`)
	}
	s.conOutHdl = conOutHdl
	s.OnClose(func() error { return windows.CloseHandle(s.conOutHdl) })
	return nil
}
