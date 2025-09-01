package terminology

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/encoder/encpng"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerTerminology{}) }

var _ term.Drawer = (*drawerTerminology)(nil)

type drawerTerminology struct{}

func (d *drawerTerminology) Name() string     { return `terminology` }
func (d *drawerTerminology) New() term.Drawer { return &drawerTerminology{} }

func (d *drawerTerminology) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	return inp != nil && inp.Name() == `terminology`, nil
}

func (d *drawerTerminology) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerTerminology) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
	if d == nil || tm == nil || img == nil || ctx == nil {
		return nil, errors.New(`nil parameter`)
	}
	start := time.Now()
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
	err := timg.Fit(bounds, rsz, tm)
	if err != nil {
		return nil, err
	}

	terminologyString, err := d.inbandString(timg, bounds, tm)
	if err != nil {
		return nil, err
	}

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.WriteString(terminologyString)
		return logx.Err(err, tm, slog.LevelInfo)
	}

	return drawFn, nil
}

func (d *drawerTerminology) inbandString(timg *term.Image, bounds image.Rectangle, term *term.Terminal) (string, error) {
	if timg == nil {
		return ``, errors.New(consts.ErrNilImage)
	}
	terminologyString, err := timg.Inband(bounds, d, term)
	if err == nil {
		return terminologyString, nil
	}
	if timg.Cropped == nil {
		return ``, errors.New(consts.ErrNilImage)
	}
	w, h := uint(bounds.Dx()), uint(bounds.Dy())
	if w > 511 || h > 511 {
		// TODO tile image
		return ``, errors.New("image too large - requires tiling (not implemented)")
	}
	_, err = timg.SaveAsFile(term, `png`, &encpng.PngEncoder{})
	if err != nil {
		return ``, err
	}

	replaceChar := " "
	var hyperlink string // unused
	// hyperlink = timg.FileName
	if len(hyperlink) > 0 {
		hyperlink += "\n"
	}

	var b strings.Builder
	b.Grow(
		3*2 + // width, height
			len(timg.FileName)*2 +
			9 + // length of fixed parts of initial string
			11 + // mux.Wrap() - for 1x tmux
			int(h)*( // area string
			int(w)+
				11+ // length of fixed parts
				11) + // mux.Wrap() - for 1x tmux
			20, // some buffer
	)
	_, err = b.WriteString(mux.Wrap(fmt.Sprintf("\033}is"+replaceChar+"%d;%d;%s%s\000", w, h, hyperlink, timg.FileName), term))
	if err != nil {
		return ``, errors.New(err)
	}
	lineArea := mux.Wrap("\033}ib\000"+strings.Repeat(replaceChar, int(w))+"\033}ie\000\n", term)
	for y := 0; y < int(h); y++ {
		_, err = b.WriteString(fmt.Sprintf("\033[%d;%dH%s", bounds.Min.Y+1+y, bounds.Min.X+1, lineArea))
		if err != nil {
			return ``, errors.New(err)
		}
	}
	terminologyString = b.String()

	timg.SetInband(bounds, terminologyString, d, term)

	return terminologyString, nil
}

// https://github.com/borisfaure/terminology/tree/master#available-commands
