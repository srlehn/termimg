//go:build linux && !android

package framebuffer

import (
	"context"
	"image"
	"image/draw"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerFramebuffer{}) }

var _ term.Drawer = (*drawerFramebuffer)(nil)

type drawerFramebuffer struct{}

func (d *drawerFramebuffer) Name() string     { return `framebuffer` }
func (d *drawerFramebuffer) New() term.Drawer { return &drawerFramebuffer{} }

func (d *drawerFramebuffer) IsApplicable(inp term.DrawerCheckerInput) (bool, environ.Properties) {
	if inp == nil {
		return false, nil
	}
	// systemd: XDG_SESSION_TYPE == tty
	sessionType, okST := inp.LookupEnv(`XDG_SESSION_TYPE`)
	if okST && sessionType != `tty` {
		// might be `x11`, `wayland`, ...
		return false, nil
	}

	// TODO check if user has permission for /dev/fb0 (video group)

	return true, nil
}
func (d *drawerFramebuffer) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerFramebuffer) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
	if d == nil || tm == nil || img == nil || ctx == nil {
		return nil, errors.New(`nil parameter`)
	}
	start := time.Now()
	w := tm.Window()
	if err := w.WindowFind(); err != nil {
		return nil, err
	}
	if w.WindowType() != `tty` {
		return nil, errors.New(`window of wrong type`)
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
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

	//

	dimg, ok := w.(draw.Image)
	if !ok {
		return nil, errors.New(`draw.Image not implemented by window`)
	}
	cpw, cph, err := tm.CellSize()
	if err != nil {
		return nil, err
	}
	boundsPixels := image.Rectangle{
		Min: image.Point{X: int(float64(bounds.Min.X) * cpw), Y: int(float64(bounds.Min.Y) * cph)},
		Max: image.Point{X: int(float64(bounds.Max.X) * cpw), Y: int(float64(bounds.Max.Y) * cph)},
	}

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		draw.Draw(dimg, boundsPixels, timg.Cropped, image.Point{}, draw.Src)
		return nil
	}

	return drawFn, nil
}
