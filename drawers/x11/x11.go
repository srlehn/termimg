//go:build !windows && !android && !darwin && !js

package x11

import (
	"context"
	"image"
	"log/slog"
	"strings"
	"time"

	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xgraphics"
	"github.com/srlehn/xgbutil/xwindow"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/x11"
)

func init() { term.RegisterDrawer(&drawerX11{}) }

const drawerNameX11 = `x11`

type drawerX11 struct{}

func (d *drawerX11) Name() string     { return drawerNameX11 }
func (d *drawerX11) New() term.Drawer { return &drawerX11{} }

func (d *drawerX11) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	if d == nil || inp == nil {
		return false, nil
	}

	tn := inp.Name()
	for _, n := range []string{
		`conhost`, // Windows
		`foot`,    // wayland
	} {
		if tn == n {
			return false, nil
		}
	}
	switch inp.Name() {
	case `conhost`, // Windows
		`foot`,   // wayland
		`edexui`: // lots of frills around terminal area in window
		return false, nil
	}

	// systemd: XDG_SESSION_TYPE == x11
	sessionType, okST := inp.LookupEnv(`XDG_SESSION_TYPE`)
	if okST && sessionType != `x11` {
		// TODO `wayland`
		return false, nil
	}

	display, okD := inp.LookupEnv(`DISPLAY`)
	if !okD || len(display) == 0 {
		return false, nil
	}
	host := strings.Split(display, `:`)[0]
	if len(host) > 0 && host != `localhost` {
		return false, nil
	}

	// TODO close
	c, err := wm.NewConn(inp)
	if err != nil {
		return false, nil
	}
	defer c.Close()

	// w, err := c.Window(inp) // TODO -> inp.WindowFind()
	w := wm.NewWindow(nil, inp)
	err = w.WindowFind()
	if err != nil || w == nil {
		return false, nil
	}

	pr := environ.NewProperties()
	pr.SetProperty(propkeys.DrawerPrefix+drawerNameX11+propkeys.DrawerVolatileSuffix, `true`)

	return true, pr
}

func (d *drawerX11) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerX11) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
	if d == nil || tm == nil || img == nil || ctx == nil {
		return nil, errors.New(`nil parameter`)
	}
	start := time.Now()
	tmw := tm.Window()
	if tmw == nil {
		return nil, errors.New(`nil window`)
	}
	if tmw.WindowType() != `x11` {
		return nil, errors.New(`wrong window type`)
	}
	if err := tmw.WindowFind(); err != nil {
		return nil, err
	}
	connXU, okConn := tmw.WindowConn().Conn().(*xgbutil.XUtil) // TODO make generic Conn
	if !okConn {
		return nil, errors.New(`not a X11 connection`)
	}
	if connXU == nil {
		return nil, errors.New(`nil X11 connection`)
	}

	timg := term.NewImage(img)
	if timg == nil {
		return nil, errors.New(consts.ErrNilImage)
	}
	rsz := tm.Resizer()
	if rsz == nil {
		return nil, errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return nil, err
	}

	cw, ch, err := tm.CellSize()
	if err != nil {
		return nil, err
	}

	// X11

	tpw, tph, err := tm.SizeInPixels()
	hasTermSize := err == nil && tpw > 0 && tph > 0
	tpw97, tph97 := int(0.97*float64(tpw)), int(0.97*float64(tph))
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
		}
	}
	boundsPx := image.Rect(int(cw)*bounds.Min.X+xOffset, int(ch)*bounds.Min.Y+yOffset,
		int(cw)*bounds.Max.X+xOffset, int(ch)*bounds.Max.Y+yOffset)

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		err := draw(connXU, w, timg, boundsPx)
		return logx.Err(err, tm, slog.Level(slog.LevelInfo))
	}

	return drawFn, nil
}

type drawFunc func(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error

var draw drawFunc = drawCurrentImpl

func drawCurrentImpl(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error {
	ximg := xgraphics.NewConvert(connXU, timg.Cropped)

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
	// wCanvas.Unmap()
	// ximg.Destroy()
	// wCanvas.Destroy()
	return nil
}
