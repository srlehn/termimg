package term

import (
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/util"
)

// TODO func for setting priority list

// AllDrawers returns all registered drawers
func AllDrawers() []Drawer {
	return drawersRegistered
}

// Drawer ...
type Drawer interface {
	Name() string
	New() Drawer
	IsApplicable(DrawerCheckerInput) (bool, environ.Properties)
	Draw(img image.Image, bounds image.Rectangle, term *Terminal) error
}

////////////////////////////////////////////////////////////////////////////////

// Draw draws an image on the terminal. bounds is the drawing area in cells.
// if the passed drawer is nil, the Terminals drawer is used.
func Draw(img image.Image, bounds image.Rectangle, term *Terminal, dr Drawer) error {
	if img == nil || term == nil {
		return errors.New(consts.ErrNilParam)
	}
	if dr == nil {
		drawers := term.Drawers()
		if len(drawers) == 0 {
			return errors.New(`terminal has no drawers`)
		}
		for _, drt := range drawers {
			if drt == nil {
				continue
			}
			dr = drt
			break
		}
	}
	if dr == nil {
		return errors.New(`nil drawer`)
	}
	return drawWith(img, bounds, term, dr)
}

func drawWith(img image.Image, bounds image.Rectangle, term *Terminal, dr Drawer) (err error) {
	if img == nil || dr == nil || term == nil {
		return errors.New(consts.ErrNilParam)
	}
	if term.resizer == nil {
		term.resizer = &resizerFallback{}
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(r)
		}
	}()
	// TODO check if redraw is necessary

	imgTerm := NewImage(img)

	return dr.Draw(imgTerm, bounds, term)
}

// Terminal Canvas

var _ draw.Image = (*Canvas)(nil)

var canvasScreenshotRefreshDuration = 100 * time.Millisecond // TODO add terminal Option for setting duration

type Canvas struct {
	terminal            *Terminal
	bounds              image.Rectangle
	boundsPixels        image.Rectangle
	screenshot          image.Image
	drawing             draw.Image
	lastScreenshotTaken time.Time
	lastSetX            int // draw.Draw: for y{for x{}}
	vid                 chan image.Image
}

func (c *Canvas) Set(x, y int, col color.Color) {
	if c == nil || c.terminal == nil {
		return
	}
	if !(&image.Point{X: x, Y: y}).In(c.boundsPixels.Sub(c.boundsPixels.Min)) {
		return
	}
	if c.drawing == nil ||
		c.drawing.Bounds().Dx() != c.boundsPixels.Dx() ||
		c.drawing.Bounds().Dy() != c.boundsPixels.Dy() {
		c.lastSetX = -2
		c.drawing = image.NewRGBA(image.Rect(0, 0, c.boundsPixels.Dx(), c.boundsPixels.Dy()))
	}
	c.drawing.Set(x, y, col)
	if ((x == 0 && y == 0) ||
		(x == c.boundsPixels.Dx()-1 && y == c.boundsPixels.Dy()-1)) &&
		(x-c.lastSetX == 1 || x-c.lastSetX == -1) {
		_ = c.terminal.Draw(c.drawing, c.bounds) // TODO log
	}
	c.lastSetX = x
}
func (c *Canvas) ColorModel() color.Model { return color.RGBAModel }
func (c *Canvas) Bounds() image.Rectangle { return c.boundsPixels.Sub(c.boundsPixels.Min) }
func (c *Canvas) At(x, y int) color.Color {
	if c == nil || c.terminal == nil {
		return color.RGBA{}
	}
	pos := &image.Point{X: x, Y: y}
	if !pos.In(c.boundsPixels.Sub(c.boundsPixels.Min)) {
		return color.RGBA{}
	}
	_ = c.storeScreenshot()
	return c.screenshot.At(x, y)
}
func (c *Canvas) storeScreenshot() error {
	if c != nil || c.terminal == nil {
		return errors.New(`nil receiver or nil struct field`)
	}
	if c.screenshot == nil || time.Since(c.lastScreenshotTaken) > canvasScreenshotRefreshDuration {
		w := c.terminal.Window()
		if w == nil {
			return errors.New(`nil window`)
		}
		img, err := w.Screenshot()
		if err != nil {
			return err
		}
		if img == nil {
			return errors.New(`nil image`)
		}
		cutout := image.NewRGBA(image.Rect(0, 0, c.boundsPixels.Dx(), c.boundsPixels.Dy()))
		draw.Draw(cutout, c.boundsPixels.Sub(c.boundsPixels.Min), img, c.boundsPixels.Min, draw.Src)
		c.screenshot = cutout
		c.lastScreenshotTaken = time.Now()
	}
	return nil
}

func (c *Canvas) CellArea() image.Rectangle  { return c.bounds }
func (c *Canvas) Offset() image.Point        { return c.boundsPixels.Min }
func (c *Canvas) Draw(img image.Image) error { return Draw(img, c.bounds, c.terminal, nil) }
func (c *Canvas) Flush() error {
	if c == nil || c.terminal == nil || c.drawing == nil || c.bounds.Eq(image.Rectangle{}) {
		return errors.New(`nil receiver or null value struct fields`)
	}
	return c.terminal.Draw(c.drawing, c.bounds)
}
func (c *Canvas) Video(dur time.Duration) chan<- image.Image {
	if c != nil {
		return nil
	}
	if c.vid != nil {
		return c.vid
	}
	c.vid = make(chan image.Image)
	_ = c.terminal.SetOptions(TUIMode)
	// TODO count miss/success ratio, avg draw time, etc
	go func() {
		var imgLast image.Image
		var tm time.Time
		for img := range c.vid {
			util.TryClose(imgLast)
			tm = time.Now()
			_ = Draw(img, c.bounds, c.terminal, nil) // TODO log
			drawTime := time.Since(tm)
			if drawTime < dur {
				time.Sleep(dur - drawTime)
			}
			imgLast = img
		}
	}()
	return c.vid
}
func (c *Canvas) Screenshot() (image.Image, error) {
	if c != nil {
		return nil, errors.New(consts.ErrNilReceiver)
	}
	if err := c.storeScreenshot(); err != nil {
		return nil, err
	}
	if c.screenshot == nil {
		return nil, errors.New(`nil image`)
	}
	return c.screenshot, nil
}
