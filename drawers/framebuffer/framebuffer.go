//go:build dev

package framebuffer

import (
	"errors"
	"image"
	"image/draw"

	"github.com/gen2brain/framebuffer"
	errorsGo "github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerFramebuffer{}) }

var _ term.Drawer = (*drawerFramebuffer)(nil)

type drawerFramebuffer struct{}

func (d *drawerFramebuffer) Name() string     { return `framebuffer` }
func (d *drawerFramebuffer) New() term.Drawer { return &drawerFramebuffer{} }

func (d *drawerFramebuffer) IsApplicable(inp term.DrawerCheckerInput) bool {
	if inp == nil {
		return false
	}
	// systemd: XDG_SESSION_TYPE == tty
	sessionType, okST := inp.LookupEnv(`XDG_SESSION_TYPE`)
	if okST && sessionType != `tty` {
		// might be `x11`, `wayland`, ...
		return false
	}

	// TODO check if user has permission for /dev/fb0 (video group)

	return false
}
func (d *drawerFramebuffer) Draw(img image.Image, bounds image.Rectangle, rsz term.Resizer, tm *term.Terminal) error {
	if d == nil || tm == nil || img == nil {
		return errors.New(`nil parameter`)
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errorsGo.New(internal.ErrNilImage)
	}

	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return err
	}

	//

	// TODO store canvas per Terminal
	canvas, err := framebuffer.Open(nil)
	if err != nil {
		return errorsGo.New(err)
	}
	defer canvas.Close()

	fbimg, err := canvas.Image()
	if err != nil {
		return errorsGo.New(err)
	}

	draw.Draw(fbimg, timg.Cropped.Bounds(), img, image.Point{}, draw.Src)

	return nil
}
