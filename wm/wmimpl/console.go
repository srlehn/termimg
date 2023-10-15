//go:build linux && !android

package wmimpl

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/framebuffer"
)

func createWindowConsole(env environ.Properties, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	var termTTY string
	if env != nil {
		termTTY, _ = env.Property(propkeys.TerminalTTY)
	}

	return &windowConsole{termTTY: termTTY}
}

var _ wm.Window = (*windowConsole)(nil)

type windowConsole struct {
	wminternal.WindowCore
	termTTY     string
	framebuffer *framebuffer.Framebuffer
	isInit      bool
	errFind     error
}

func (w *windowConsole) WindowFind() error {
	if w.isInit {
		return w.errFind
	}
	w.isInit = true
	devFB := `/dev/fb0` // TODO
	fb, err := framebuffer.Init(devFB)
	if err != nil {
		w.errFind = err
		return err
	}
	w.framebuffer = fb
	return nil
}
func (w *windowConsole) WindowType() string { return `tty` }
func (w *windowConsole) Close() error {
	if w == nil || w.framebuffer == nil {
		return nil
	}
	w.framebuffer.Close()
	return nil
}
func (w *windowConsole) Size() image.Point {
	if w == nil || w.framebuffer == nil || w.WindowFind() != nil {
		return image.Point{}
	}
	bounds := w.framebuffer.Bounds()
	return image.Point{X: bounds.Dx(), Y: bounds.Dy()}
}
func (w *windowConsole) Screenshot() (image.Image, error) {
	if w == nil || w.framebuffer == nil {
		return nil, errors.New(`nil receiver or framebuffer`)
	}
	if err := w.WindowFind(); err != nil {
		return nil, err
	}
	bounds := w.framebuffer.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), w.framebuffer, bounds.Min, draw.Src)
	return dst, nil
}

func (w *windowConsole) At(x, y int) color.Color {
	if w == nil || w.framebuffer == nil || w.WindowFind() != nil {
		return nil
	}
	return w.framebuffer.At(x, y)
}
func (w *windowConsole) Bounds() image.Rectangle {
	if w == nil || w.framebuffer == nil || w.WindowFind() != nil {
		return image.Rectangle{}
	}
	return w.framebuffer.Bounds()
}
func (w *windowConsole) ColorModel() color.Model {
	if w == nil || w.framebuffer == nil || w.WindowFind() != nil {
		return nil
	}
	return w.framebuffer.ColorModel()
}
func (w *windowConsole) Set(x, y int, c color.Color) {
	if w == nil || w.framebuffer == nil || w.WindowFind() != nil {
		return
	}
	w.framebuffer.Set(x, y, c)
}
