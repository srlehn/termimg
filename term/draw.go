package term

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/google/btree"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
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
	IsApplicable(DrawerCheckerInput) (bool, Properties)
	Draw(img image.Image, bounds image.Rectangle, term *Terminal) error
	Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, term *Terminal) (drawFn func() error, _ error)
}

////////////////////////////////////////////////////////////////////////////////

// Draw draws an image on the terminal. bounds is the drawing area in cells.
// if the passed drawer is nil, the Terminals drawer is used.
func Draw(img image.Image, bounds image.Rectangle, term *Terminal, dr Drawer) error {
	if err := errors.NilParam(img, term); err != nil {
		return err
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
		return errors.NilParam()
	}
	if term.resizer == nil {
		term.resizer = &resizerFallback{}
	}
	defer func() {
		if r := recover(); r != nil {
			logx.Err(r, term, slog.LevelError)
		}
	}()
	// TODO check if redraw is necessary

	imgTerm := NewImage(img)

	return dr.Draw(imgTerm, bounds, term)
}

// Terminal Canvas

var _ draw.Image = (*Canvas)(nil)

var canvasScreenshotRefreshDuration = 100 * time.Millisecond // TODO add terminal Option for setting duration

// Canvas has to be created with (*term.Terminal).NewCanvas()
type Canvas struct {
	terminal            *Terminal
	bounds              image.Rectangle
	boundsPixels        image.Rectangle
	screenshot          image.Image
	drawing             draw.Image
	image               image.Image
	lastScreenshotTaken time.Time
	lastSetX            int // draw.Draw: for y{for x{}}
	closed              chan struct{}
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
	if c.image != nil {
		if c.lastSetX < -1 &&
			c.drawing.Bounds().Dx() == c.image.Bounds().Dx() &&
			c.drawing.Bounds().Dy() == c.image.Bounds().Dy() {
			draw.Draw(c.drawing, c.drawing.Bounds(), c.image, c.image.Bounds().Min, draw.Src)
		}
		util.TryClose(c.image)
		c.image = nil
	}
	c.drawing.Set(x, y, col)
	if ((x == 0 && y == 0) ||
		(x == c.boundsPixels.Dx()-1 && y == c.boundsPixels.Dy()-1)) &&
		(x-c.lastSetX == 1 || x-c.lastSetX == -1) {
		err := c.terminal.Draw(c.drawing, c.bounds)
		logx.IsErr(err, c.terminal, slog.LevelError)
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
	if c == nil || c.terminal == nil {
		return errors.New(`nil receiver or nil struct field`)
	}
	if c.screenshot == nil || time.Since(c.lastScreenshotTaken) > canvasScreenshotRefreshDuration {
		w := c.terminal.Window()
		if w == nil {
			return errors.New(`nil window`)
		}
		img, err := w.Screenshot()
		if logx.IsErr(err, c.terminal, slog.LevelInfo) {
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

func (c *Canvas) SetCellArea(bounds image.Rectangle) error {
	err := errors.NilReceiver(c, c.terminal)
	if logx.IsErr(err, c.terminal, slog.LevelInfo) {
		return err
	}
	if err := errors.NilReceiver(c, c.terminal); err != nil {
		return err
	}
	cpw, cph, err := c.terminal.CellSize()
	if logx.IsErr(err, c.terminal, slog.LevelError) {
		return err
	}
	boundsPixels := image.Rectangle{
		Min: image.Point{
			X: int(float64(bounds.Min.X) * cpw),
			Y: int(float64(bounds.Min.Y) * cph),
		},
		Max: image.Point{
			X: int(float64(bounds.Max.X) * cpw),
			Y: int(float64(bounds.Max.Y) * cph),
		},
	}
	c.bounds = bounds
	c.boundsPixels = boundsPixels
	c.drawing = image.NewRGBA(image.Rect(0, 0, boundsPixels.Dx(), boundsPixels.Dy()))
	c.lastSetX = -2
	return nil
}
func (c *Canvas) CellArea() image.Rectangle { return c.bounds }
func (c *Canvas) Offset() image.Point       { return c.boundsPixels.Min }
func (c *Canvas) SetImage(img image.Image) error {
	if c == nil || c.bounds.Eq(image.Rectangle{}) {
		return logx.Err(`nil receiver or null value struct fields`, c.terminal, slog.LevelError)
	}
	c.image = img
	c.drawing = nil
	return nil
}

// Draw flushes stored image when img is nil
func (c *Canvas) Draw(img image.Image) error {
	if c == nil {
		return errors.NilReceiver()
	}
	if img != nil {
		if err := c.SetImage(img); err != nil {
			return err
		}
	}
	if c.terminal == nil {
		return logx.Err(`no terminal`, c.terminal, slog.LevelError)
	}
	drawers := c.terminal.Drawers()
	if len(drawers) == 0 {
		return logx.Err(`no drawer`, c.terminal, slog.LevelError)
	}
	if c.image == nil {
		if c.drawing != nil {
			c.image = c.drawing
		} else {
			return logx.Err(`nothing to draw`, c.terminal, slog.LevelError)
		}
	}
	var errs []error
	for _, dr := range drawers {
		err := Draw(c.image, c.bounds, c.terminal, dr)
		if !logx.IsErr(err, c.terminal, slog.LevelInfo) {
			goto successfulDraw
		}
		errs = append(errs, err)
	}
successfulDraw:
	return logx.Err(errors.Join(errs...), c.terminal, slog.LevelError)
}
func (c *Canvas) Video(ctx context.Context, vid <-chan image.Image, frameDur time.Duration) error {
	if c == nil || c.terminal == nil {
		return logx.Err(errors.NilReceiver(), c.terminal, slog.LevelError)
	}
	if ctx == nil || vid == nil {
		return logx.Err(errors.NilParam(), c.terminal, slog.LevelError)
	}
	_ = c.terminal.SetOptions(TUIMode)
	// TODO count miss/success ratio, avg draw time, etc

	drawFnChan, err := c.prepImagesParallelOrdered(ctx, vid, frameDur, runtime.NumCPU())
	if err != nil {
		return err
	}

	tm := time.Now()
	var tmLast time.Time
outer:
	for {
		select {
		case drawFn, ok := <-drawFnChan:
			if !ok {
				break outer
			}
			tm, tmLast = time.Now(), tm
			frameTime := tm.Sub(tmLast)
			err := logx.TimeIt(drawFn.fn, `image drawing`, c.terminal, `drawer`, c.terminal.Drawers()[0].Name())
			logx.IsErr(err, c.terminal, slog.LevelError)
			if c.drawing != nil {
				util.TryClose(c.drawing)
				c.drawing = nil
			}
			drawTime := time.Since(tm)
			logx.Debug("durations", c.terminal, "draw-duration", drawTime, "frame-time", frameTime)
			if drawTime < frameDur {
				time.Sleep(frameDur - drawTime)
			}
		case <-c.closed:
			break outer
		case <-ctx.Done():
			break outer
		}
	}
	return nil
}

type drawFn struct {
	id int
	fn func() error
}

type imgWithID struct {
	id  int
	img image.Image
}

func (c *Canvas) prepImagesParallelOrdered(ctx context.Context, vid <-chan image.Image, frameDur time.Duration, n int) (<-chan drawFn, error) {
	drawFnUnorderedChan, err := c.prepImagesParallelUnordered(ctx, vid, runtime.NumCPU())
	if err != nil {
		return nil, err
	}
	drawFnChan := make(chan drawFn)
	go func() {
		defer close(drawFnChan)
		tr := btree.NewG(2, func(a, b drawFn) bool { return a.id < b.id })
		defer func() {
			tr.Ascend(func(item drawFn) bool {
				logx.Debug("remaining frame", c.terminal, "remaining-frame", item.id) // TODO rm
				return true
			})
		}()
		var drFnLast drawFn
		for drFn := range drawFnUnorderedChan {
			var isNext bool
			if drFn.id == drFnLast.id+1 {
				isNext = true
			} else {
				tr.DescendRange(drFn, drFnLast, func(item drawFn) bool {
					if item.id == drFnLast.id+1 {
						isNext = true
						return false
					}
					return true
				})
			}
			if isNext {
				drawFnChan <- drFn
				tr.Delete(drFnLast)
				drFnLast = drFn
			} else {
				tr.ReplaceOrInsert(drFn)
			}
			logx.Debug("b-tree", c.terminal, "node-count", tr.Len())
		}
	}()
	return drawFnChan, nil
}
func (c *Canvas) prepImagesParallelUnordered(ctx context.Context, vid <-chan image.Image, n int) (<-chan drawFn, error) {
	vidwid := make(chan imgWithID)
	go func() {
		defer close(vidwid)
		frame := 0
		for img := range vid {
			logx.Debug("frame", c.terminal, "frame", frame)
			vidwid <- imgWithID{id: frame, img: img}
			frame++
		}
	}()
	var drawFnChans []<-chan drawFn
	for i := 0; i < n; i++ {
		// TODO order images
		drawFnChan, err := c.prepImages(ctx, vidwid)
		if !logx.IsErr(err, c.terminal, slog.LevelInfo) && drawFnChan != nil {
			drawFnChans = append(drawFnChans, drawFnChan)
		}
	}
	drawFnCombChan := make(chan drawFn)
	var wg sync.WaitGroup
	for i, drawFnChan := range drawFnChans {
		wg.Go(func() {
			drawFnChan := drawFnChan
			workerID := i
			for drawFn := range drawFnChan {
				logx.Debug("drawer func id", c.terminal, "worker-id", workerID, "frame", drawFn.id)
				drawFnCombChan <- drawFn
			}
		})
	}
	go func() {
		wg.Wait()
		close(drawFnCombChan)
	}()
	return drawFnCombChan, nil
}
func (c *Canvas) prepImages(ctx context.Context, imgChan <-chan imgWithID) (<-chan drawFn, error) {
	var dr Drawer
	if drawers := c.terminal.Drawers(); len(drawers) == 0 {
		return nil, logx.Err(`no drawer`, c.terminal, slog.LevelError)
	} else {
		dr = drawers[0]
	}
	drawFnChan := make(chan drawFn)
	go func() {
		defer close(drawFnChan)
		for {
			select {
			case imgwid, ok := <-imgChan:
				if !ok {
					return
				}
				drFn, err := dr.Prepare(ctx, imgwid.img, c.bounds, c.terminal)
				if !logx.IsErr(err, c.terminal, slog.LevelInfo) && drFn != nil {
					drawFnChan <- drawFn{id: imgwid.id, fn: drFn}
				}
				go func(imgLast image.Image) { util.TryClose(imgLast) }(c.image)
				c.image = imgwid.img
			case <-ctx.Done():
				return
			}
		}
	}()

	return drawFnChan, nil
}
func (c *Canvas) Screenshot() (image.Image, error) {
	if c == nil || c.terminal == nil {
		return nil, errors.NilReceiver()
	}
	if err := c.storeScreenshot(); logx.IsErr(err, c.terminal, slog.LevelInfo) {
		return nil, err
	}
	if c.screenshot == nil {
		return nil, errors.New(`nil image`)
	}
	return c.screenshot, nil
}
func (c *Canvas) Close() error {
	if c == nil || c.closed == nil {
		return nil
	}
	select {
	case c.closed <- struct{}{}:
	default:
	}
	return nil
}
