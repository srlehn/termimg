//go:build !windows && !android && !darwin && !js

package x11

import (
	"image"
	"strings"
	"time"

	"github.com/go-errors/errors"

	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xevent"
	"github.com/srlehn/xgbutil/xgraphics"
	"github.com/srlehn/xgbutil/xwindow"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/x11"
)

func init() { term.RegisterDrawer(&drawerX11{}) }

type drawerX11 struct{}

func (d *drawerX11) Name() string     { return `x11` }
func (d *drawerX11) New() term.Drawer { return &drawerX11{} }

func (d *drawerX11) IsApplicable(inp term.DrawerCheckerInput) bool {
	if d == nil || inp == nil {
		return false
	}

	tn := inp.Name()
	for _, n := range []string{
		`conhost`, // Windows
		`foot`,    // wayland
	} {
		if tn == n {
			return false
		}
	}
	switch inp.Name() {
	case `conhost`, // Windows
		`foot`,   // wayland
		`edexui`: // lots of frills around terminal area in window
		return false
	}

	// systemd: XDG_SESSION_TYPE == x11
	sessionType, okST := inp.LookupEnv(`XDG_SESSION_TYPE`)
	if okST && sessionType != `x11` {
		// TODO `wayland`
		return false
	}

	display, okD := inp.LookupEnv(`DISPLAY`)
	if !okD || len(display) == 0 {
		return false
	}
	host := strings.Split(display, `:`)[0]
	if len(host) > 0 && host != `localhost` {
		return false
	}

	// TODO close
	c, err := wm.NewConn()
	if err != nil {
		return false
	}
	defer c.Close()

	// w, err := c.Window(inp) // TODO -> inp.WindowFind()
	w := wm.NewWindow(nil, inp)
	err = w.WindowFind()
	if err != nil || w == nil {
		return false
	}

	return true
}

func (d *drawerX11) Draw(img image.Image, bounds image.Rectangle, rsz term.Resizer, tm *term.Terminal) error {
	if d == nil {
		return errors.New(`nil receiver or receiver with nil values`)
	}
	if tm == nil || img == nil {
		return errors.New(`nil parameter`)
	}
	tmw := tm.Window()
	if tmw == nil {
		return errors.New(`nil window`)
	}
	if tmw.WindowType() != `x11` {
		return errors.New(`wrong window type`)
	}
	if err := tmw.WindowFind(); err != nil {
		return err
	}
	connXU, okConn := tmw.WindowConn().Conn().(*xgbutil.XUtil) // TODO make generic Conn
	if !okConn {
		return errors.New(`not a X11 connection`)
	}
	if connXU == nil {
		return errors.New(`nil X11 connection`)
	}

	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errors.New(internal.ErrNilImage)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return err
	}

	cw, ch, err := tm.CellSize()
	if err != nil {
		return err
	}

	// X11

	w := xwindow.New(connXU, xproto.Window(tmw.WindowID()))
	ximg := xgraphics.NewConvert(connXU, timg.Cropped)
	boundsPx := image.Rect(int(cw)*bounds.Min.X, int(ch)*bounds.Min.Y, int(cw)*bounds.Max.X, int(ch)*bounds.Max.Y)

	go xevent.Main(connXU)

	wCanvas, err := x11.AttachWindow(connXU, w, boundsPx)
	if err != nil {
		return err
	}
	if err := ximg.XSurfaceSet(wCanvas.Id); err != nil {
		return errors.New(err)
	}
	if err := ximg.XDrawChecked(); err != nil {
		return errors.New(err)
	}
	ximg.XPaint(wCanvas.Id)
	wCanvas.Map()
	time.Sleep(2 * time.Second)
	// wCanvas.Unmap()
	// ximg.Destroy()
	// wCanvas.Destroy()
	return nil
}
