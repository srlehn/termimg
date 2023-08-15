//go:build unix && !noX11 && !android && !darwin && !js

// based on xgbutil examples

package wmimpl

import (
	"errors"
	"image"
	"image/color"
	"strconv"

	errorsGo "github.com/go-errors/errors"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/res"
	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/ewmh"
	"github.com/srlehn/xgbutil/icccm"
	"github.com/srlehn/xgbutil/xevent"
	"github.com/srlehn/xgbutil/xgraphics"
	"github.com/srlehn/xgbutil/xwindow"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/wm"
)

// connX11 ...
type connX11 struct {
	*xgbutil.XUtil
}

func newConn() (*connX11, error) {
	displayVar := ``
	return newConnDisplay(displayVar)
}

func newConnDisplay(displayVar string) (*connX11, error) {
	// Connect to the X server using the DISPLAY environment variable.
	conn, err := xgbutil.NewConnDisplay(displayVar)
	if err != nil {
		return nil, errorsGo.New(err)
	}
	return &connX11{conn}, nil
}

func (c *connX11) Close() error {
	if c == nil || c.XUtil == nil {
		return nil
	}
	conn := c.XUtil.Conn()
	if conn == nil {
		return nil
	}
	conn.Close()
	return nil
}

func (c *connX11) Conn() any {
	if c == nil {
		return nil
	}
	return c.XUtil
}

// Windows ...
func (c *connX11) Windows() ([]wm.Window, error) {
	ws, err := c.getWindows()
	if err != nil {
		return nil, err
	}
	var wsRet []wm.Window
	for _, w := range ws {
		if w == nil {
			continue
		}
		wsRet = append(wsRet, w)
	}
	return wsRet, nil
}

// DisplayImage ...
func (c *connX11) DisplayImage(img image.Image, windowName string) {
	ximg := xgraphics.NewConvert(c.XUtil, img)
	_ = ximg.XShowExtra(windowName, true)
	xevent.Main(c.XUtil)
}

func (c *connX11) getWindows() ([]*windowX11, error) {
	var windows []*windowX11

	// Get a list of all client ids.
	clientIDs, err := ewmh.ClientListGet(c.XUtil)
	if err != nil {
		return nil, errorsGo.New(err)
	}

	err = res.Init(c.XUtil.Conn())
	if err != nil {
		return nil, err
	}
	var clientIDSpecs []res.ClientIdSpec

	// Iterate through each client, find its name and find its size.
	for _, window := range clientIDs {
		class, instance, _ := c.getWindowClass(window)
		windows = append(
			windows,
			&windowX11{
				id:       uint64(window),
				name:     util.IgnoreError(c.getWindowName(window)),
				class:    class,
				instance: instance,
			},
		)
		clientIDSpecs = append(clientIDSpecs, res.ClientIdSpec{
			Client: uint32(window),
			Mask:   res.ClientIdMaskLocalClientPID,
		})
	}

	repl, err := res.QueryClientIds(c.XUtil.Conn(), uint32(len(clientIDSpecs)), clientIDSpecs).Reply()
	var xidPIDMap map[uint32]uint32
	if err == nil && repl != nil {
		xidPIDMap = make(map[uint32]uint32)
		for _, id := range repl.Ids {
			if len(id.Value) != 1 {
				continue
			}
			xidPIDMap[id.Spec.Client] = id.Value[0]
		}
	}

	var windowsRet []*windowX11
	for _, w := range windows {
		if xidPIDMap != nil {
			if pid, ok := xidPIDMap[uint32(w.id)]; ok {
				util.Println(w.id, w.pid)
				w.pid = uint64(pid)
			}
		}
		if w.pid == 0 {
			// fallback to unreliable client set _NET_WM_PID
			netWMPID, err := c.getWindowPIDNetWMPID(xproto.Window(w.id))
			if err == nil {
				w.pid = netWMPID
			}
		}

		windowsRet = append(windowsRet, w)
	}

	return windowsRet, nil
}

// getWindowName ...
func (c *connX11) getWindowName(w xproto.Window) (string, error) {
	name, errE := ewmh.WmNameGet(c.XUtil, w)
	if errE == nil {
		return name, nil
	}

	// If there was a problem getting _NET_WM_NAME or if its empty,
	// try the old-school version.
	name, errI := icccm.WmNameGet(c.XUtil, w)
	if errI == nil {
		return name, nil
	}

	return ``, errorsGo.New(errors.Join(errE, errI))
}

// getWindowClass ...
func (c *connX11) getWindowClass(w xproto.Window) (class, instance string, _ error) {
	cl, err := icccm.WmClassGet(c.XUtil, w)
	if err != nil {
		return ``, ``, errorsGo.New(err)
	}
	if cl == nil {
		return ``, ``, errorsGo.New(`nil icccm.WmClassGet() client`)
	}
	return cl.Class, cl.Instance, nil
}

var _ = (*connX11).getWindowPID

// getWindowPID ...
func (c *connX11) getWindowPID(w xproto.Window) (uint64, error) {
	pid, errQCI := c.getWindowPIDQueryClientIDs(w)
	if errQCI == nil {
		return pid, nil
	}
	pid, errNWP := c.getWindowPIDNetWMPID(w)
	if errNWP == nil {
		return pid, nil
	}
	return 0, errorsGo.New(errors.Join(errQCI, errNWP))
}

func (c *connX11) getWindowPIDQueryClientIDs(w xproto.Window) (uint64, error) {
	err := res.Init(c.XUtil.Conn())
	if err != nil {
		return 0, errorsGo.New(err)
	}

	// Iterate through each client, find its name and find its size.
	clientIDSpecs := []res.ClientIdSpec{{
		Client: uint32(w),
		Mask:   res.ClientIdMaskLocalClientPID,
	}}

	repl, err := res.QueryClientIds(c.XUtil.Conn(), uint32(len(clientIDSpecs)), clientIDSpecs).Reply()
	if err != nil {
		return 0, errorsGo.New(err)
	}
	if repl == nil {
		return 0, errorsGo.New(`nil QueryClientIds reply`)
	}
	var pid uint64
	for _, id := range repl.Ids {
		if len(id.Value) != 1 || id.Spec.Client != uint32(w) {
			continue
		}
		pid = uint64(id.Value[0])
		break
	}
	return pid, nil
}
func (c *connX11) getWindowPIDNetWMPID(w xproto.Window) (uint64, error) {
	if c == nil {
		return 0, errorsGo.New(`nil conn`)
	}
	// some processes set _NET_WM_PID for the X11 window, e.g.: DomTerm
	pid, err := ewmh.WmPidGet(c.XUtil, w)
	if err != nil {
		return 0, errorsGo.New(err)
	}
	return uint64(pid), nil
}

// windowX11 ...
type windowX11 struct {
	wminternal.WindowCore
	id       uint64 // xproto.Window
	name     string
	class    string
	instance string
	pid      uint64 // terminal
	is       wm.IsWindowFunc
	isInit   bool
	errFind  error
	conn     *connX11
}

var _ wm.Window = (*windowX11)(nil)

func (w *windowX11) WindowType() string { return `x11` }
func (w *windowX11) WindowName() string {
	if w == nil || w.WindowFind() != nil {
		return ``
	}
	return w.name
}
func (w *windowX11) WindowClass() string {
	if w == nil || w.WindowFind() != nil {
		return ``
	}
	return w.class
}
func (w *windowX11) WindowInstance() string {
	if w == nil || w.WindowFind() != nil {
		return ``
	}
	return w.instance
}
func (w *windowX11) WindowID() uint64 {
	if w == nil || w.WindowFind() != nil {
		return 0
	}
	return uint64(w.id)
}
func (w *windowX11) WindowPID() uint64 {
	if w == nil || w.WindowFind() != nil {
		return 0
	}
	return w.pid
}
func (w *windowX11) WindowConn() wm.Connection {
	if w == nil || w.WindowFind() != nil {
		return nil
	}
	return w.conn
}
func (w *windowX11) WindowFind() error {
	if w == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	if w.isInit {
		return w.errFind
	}
	w.isInit = true
	if w.conn == nil || w.conn.XUtil == nil {
		c, err := newConn()
		if err != nil {
			w.errFind = err
			return err
		}
		w.conn = c
		if w.conn == nil || w.conn.XUtil == nil {
			err := errorsGo.New(`nil X11 connection`)
			w.errFind = err
			return err
		}
	}
	connXU := w.conn.XUtil
	windows, err := w.conn.getWindows()
	if err != nil {
		return err
	}
	var couldCompare bool
	if w.id > 0 {
		couldCompare = true
		for _, wdw := range windows {
			if wdw == nil || w.id != wdw.id {
				continue
			}
			windows = []*windowX11{wdw}
			break
		}
	}
	if len(windows) > 1 && w.pid > 0 {
		couldCompare = true
		var ws []*windowX11
		for _, wdw := range windows {
			if wdw == nil || w.pid != wdw.pid {
				continue
			}
			ws = append(ws, wdw)
		}
		windows = ws
	}
	if len(windows) > 1 && w.is != nil {
		couldCompare = true
		var ws []*windowX11
		for _, wdw := range windows {
			if wdw == nil {
				continue
			}
			if is, _ := w.is(wdw); !is {
				continue
			}
			ws = append(ws, wdw)
		}
		if len(ws) > 0 {
			windows = ws
		}
	}
	if len(windows) > 1 && (len(w.name) > 0 || len(w.class) > 0 || len(w.instance) > 0) {
		couldCompare = true
		var ws []*windowX11
		for _, wdw := range windows {
			if wdw == nil ||
				w.name != wdw.name ||
				w.class != wdw.class ||
				w.instance != wdw.instance {
				continue
			}
			ws = append(ws, wdw)
		}
		windows = ws
	}
	if len(windows) > 1 {
		// choose
		var errStrMulti = `more than 1 window match`
		// https://github.com/BurntSushi/xgb/blob/deaf085/examples/get-active-window/main.go#L39
		atomNameNetActiveWindow := "_NET_ACTIVE_WINDOW"
		a, err := xproto.InternAtom(connXU.Conn(), true, uint16(len(atomNameNetActiveWindow)), atomNameNetActiveWindow).Reply()
		if err != nil {
			err = errorsGo.New(err)
			w.errFind = err
			return err
		}
		r, err := xproto.GetProperty(connXU.Conn(), false, xproto.Setup(connXU.Conn()).DefaultScreen(connXU.Conn()).Root, a.Atom, xproto.GetPropertyTypeAny, 0, (1<<32)-1).Reply()
		if err != nil {
			err = errorsGo.New(err)
			w.errFind = err
			return err
		}
		wActive := xwindow.New(w.conn.XUtil, xproto.Window(xgb.Get32(r.Value)))
		var lastWParentID xproto.Window
		parent := wActive
	Outer:
		for {
			for _, wdw := range windows {
				if parent.Id != xproto.Window(wdw.id) {
					continue
				}
				windows = []*windowX11{wdw}
				break Outer
			}
			parent, err := wActive.Parent()
			if err != nil {
				err = errorsGo.New(err)
				w.errFind = err
				return err
			}
			if parent == nil || parent.Id == lastWParentID {
				err = errorsGo.New(errStrMulti)
				w.errFind = err
				return err
			}
			lastWParentID = parent.Id
		}
	}
	switch len(windows) {
	case 0:
		err := errorsGo.New(`no window match`)
		w.errFind = err
		return err
	case 1:
		// success
		wdw := windows[0]
		if w.id == 0 {
			w.id = wdw.id
		}
		if w.pid == 0 {
			w.pid = wdw.pid
		}
		if len(w.name) == 0 {
			w.name = wdw.name
		}
		if len(w.class) == 0 {
			w.class = wdw.class
		}
		if len(w.instance) == 0 {
			w.instance = wdw.instance
		}
		w.errFind = nil
		return nil
	}

	if !couldCompare {
		err := errorsGo.New(`window find: nothing to compare against`)
		w.errFind = err
		return err
	}

	err = errorsGo.New(`window find failed`)
	w.errFind = err
	return err
}
func (w *windowX11) Screenshot() (image.Image, error) {
	if w == nil {
		return nil, errorsGo.New(internal.ErrNilReceiver)
	}
	if err := w.WindowFind(); err != nil {
		return nil, err
	}
	if w.conn == nil || w.conn.XUtil == nil {
		return nil, errorsGo.New(`nil conn`)
	}

	wXW := xwindow.New(w.conn.XUtil, xproto.Window(w.id))
	// TODO wezterm doesn't draw until mouse/keyboard event
	wXW.Focus() // doesn't help for wezterm

	// Use the "NewDrawable" constructor to create an xgraphics.Image value
	// from a drawable. (Usually this is done with pixmaps, but drawables
	// can also be windows.)
	// return xgraphics.NewDrawable(c.XUtil, xproto.Drawable(w.WindowID()))
	ximg, err := xgraphics.NewDrawable(w.conn.XUtil, xproto.Drawable(w.id))
	if err != nil {
		return nil, errorsGo.New(err)
	}
	defer ximg.Destroy()

	ximgBounds := ximg.Bounds()
	ximgCopy := image.NewRGBA(ximgBounds)
	for x := 0; x < ximgBounds.Size().X; x++ {
		for y := 0; y < ximgBounds.Size().Y; y++ {
			r, g, b, _ := ximg.At(x, y).RGBA()
			colRGBA := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			ximgCopy.Set(x, y, colRGBA)
		}
	}

	return ximgCopy, nil
}
func (w *windowX11) Close() error {
	if w == nil || w.conn == nil || w.conn.Conn() == nil {
		return nil
	}
	connXU, okXU := w.conn.Conn().(*xgbutil.XUtil)
	if okXU && connXU != nil {
		connXU.Conn().Close()
	}
	return nil
}

func createWindowX11(env environ.Proprietor, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	var windowID, pidTerm uint64
	if env != nil {
		pidTermStr, okPIDTerm := env.Property(propkeys.TerminalPID)
		if okPIDTerm && len(pidTermStr) > 0 {
			pt, err := strconv.ParseUint(pidTermStr, 10, 64)
			if err == nil {
				pidTerm = pt
			}
		}
		windowIDStr, okWindowID := env.LookupEnv(`WINDOWID`)
		if okWindowID && len(windowIDStr) > 0 {
			wi, err := strconv.ParseUint(windowIDStr, 10, 64)
			if err == nil {
				windowID = wi
			}
		}
	}

	if windowID == 0 && pidTerm < 2 && isWindow == nil &&
		len(name) == 0 && len(class) == 0 && len(instance) == 0 {
		return nil
	}

	return &windowX11{
		is:       isWindow,
		id:       windowID,
		pid:      pidTerm,
		name:     name,
		class:    class,
		instance: instance,
	}
}
