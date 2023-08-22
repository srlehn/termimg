//go:build !windows && !android && !darwin && !js

package x11

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xevent"
	"github.com/srlehn/xgbutil/xgraphics"
	"github.com/srlehn/xgbutil/xwindow"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	. "github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/x11"
)

var _ = Println // TODO rm util import

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
	c, err := wm.NewConn(inp)
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

func (d *drawerX11) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
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

	timg := term.NewImage(img)
	if timg == nil {
		return errors.New(consts.ErrNilImage)
	}
	rsz := tm.Resizer()
	if rsz == nil {
		return errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return err
	}

	cw, ch, err := tm.CellSize()
	if err != nil {
		return err
	}

	// X11

	fmt.Println()
	tpw, tph, err := tm.SizeInPixels()
	hasTermSize := err == nil && tpw > 0 && tph > 0
	_ = hasTermSize
	tpw97, tph97 := int(0.97*float64(tpw)), int(0.97*float64(tph))
	_, _ = tpw, tph
	_, _ = tpw97, tph97
	w := xwindow.New(connXU, xproto.Window(tmw.WindowID()))
	var xOffset, yOffset int
	if hasTermSize {
		treeRepl, err := xproto.QueryTree(connXU.Conn(), w.Id).Reply()
		if err == nil && treeRepl != nil {
			for _, child := range treeRepl.Children {
				geomChild, err := xwindow.New(connXU, child).Geometry()
				if err == nil && geomChild != nil {
					wGeomChild := geomChild.Width()
					hGeomChild := geomChild.Height()
					if wGeomChild >= tpw97 && hGeomChild >= tph97 {
						// xOffset = wGeomChild + geomChild.X() - int(tpw)
						if hGeomChild >= int(tph) {
							yOffset = hGeomChild + geomChild.Y() - int(tph)
						} else {
							yo := int(tph) - hGeomChild - geomChild.Y()
							if yo > 0 && yo <= int(0.1*float64(tph)) {
								yOffset = yo
							}
						}
						break
					}
				}
				// Println(geomChild, tpw, tph, xOffset, yOffset)
			}
		}
	}
	if xOffset == 0 && yOffset == 0 {
		geom, err := w.Geometry()
		if err == nil && geom != nil {
			// geom.X(), geom.Y() offset to parent window
			// xOffset = geom.X()
			// yOffset = geom.Y()
			wGeom := geom.Width()
			hGeom := geom.Height()
			if wGeom >= tpw97 && hGeom >= tph97 {
				// x might contain scrollbar width
				// xOffset = wGeom - int(tpw)
				if hGeom >= int(tph) {
					yOffset = hGeom - int(tph)
				} else {
					yo := int(tph) - hGeom
					if yo > 0 && yo <= int(0.1*float64(tph)) {
						yOffset = yo
					}
				}
			}
			// Println(geom, tpw, tph, xOffset, yOffset)
		}
	}
	// Println("xOffset", xOffset, "yOffset", yOffset)
	boundsPx := image.Rect(int(cw)*bounds.Min.X+xOffset, int(ch)*bounds.Min.Y+yOffset,
		int(cw)*bounds.Max.X+xOffset, int(ch)*bounds.Max.Y+yOffset)

	//

	if err := draw(connXU, w, timg, boundsPx); err != nil {
		return err
	}

	return nil
}

type drawFunc func(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error

var draw drawFunc = drawCurrentImpl

func drawCurrentImpl(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error {
	ximg := xgraphics.NewConvert(connXU, timg.Cropped)

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
