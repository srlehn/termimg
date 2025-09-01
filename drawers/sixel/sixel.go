package sixel

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	sixel "github.com/mattn/go-sixel"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerSixel{}) }

var _ term.Drawer = (*drawerSixel)(nil)

type drawerSixel struct{}

func (d *drawerSixel) Name() string     { return `sixel` }
func (d *drawerSixel) New() term.Drawer { return &drawerSixel{} }

func (d *drawerSixel) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	// example query: "\033[0c"
	// possible answer from the terminal (here xterm): "\033[[?63;1;2;4;6;9;15;22c", vte(?): ...62,9;c
	// the "4" signals that the terminal is capable of sixel
	// conhost.exe knows this sequence.
	if inp == nil {
		return false, nil
	}
	sixelCapable, isSixelCapable := inp.Property(propkeys.SixelCapable)
	if !isSixelCapable || sixelCapable != `true` {
		return false, nil
	}

	if slices.Contains([]string{
		`domterm`,
		`st`,
		`tabby`, // no images
		`wayst`, // spews character salad
	}, inp.Name()) {
		return false, nil // skip buggy implementations
	}

	repl, err := term.CachedQuery(inp, mux.Wrap(queries.DA1, inp), inp, parser.StopOnAlpha, inp, nil)
	// TODO fix mintty - querying in general on windows?
	if err != nil {
		return false, nil
	}
	if len(repl) == 0 || repl[len(repl)-1] != 'c' {
		return false, nil
	}
	repl = repl[:len(repl)-1]
	termCapabilities := strings.Split(repl, `;`)[1:]
	for _, cap := range termCapabilities {
		if cap == `4` {
			return true, nil
		}
	}
	return false, nil
}

func (d *drawerSixel) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerSixel) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
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
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return nil, err
	}

	sixelString, err := d.inbandString(timg, bounds, tm)
	if err != nil {
		return nil, err
	}

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.WriteString(sixelString)
		return logx.Err(err, tm, slog.LevelInfo)
	}

	return drawFn, nil
}

func (d *drawerSixel) inbandString(timg *term.Image, bounds image.Rectangle, term *term.Terminal) (string, error) {
	if timg == nil {
		return ``, errors.New(consts.ErrNilImage)
	}
	sixelString, err := timg.Inband(bounds, d, term)
	if err == nil {
		return sixelString, nil
	}
	if err := timg.Decode(); err != nil {
		return ``, err
	}
	if timg.Cropped == nil {
		timg.Cropped = timg.Original
	}

	// sixel
	// https://vt100.net/docs/vt3xx-gp/chapter14.html
	byteBuf := new(bytes.Buffer)
	enc := sixel.NewEncoder(byteBuf)
	enc.Dither = true
	if err := enc.Encode(timg.Cropped); err != nil {
		return ``, err
	}
	sixelString = mux.Wrap("\033[?8452h"+byteBuf.String(), term)
	// position where the image should appear (upper left corner) + sixel
	// https://github.com/mintty/mintty/wiki/CtrlSeqs#sixel-graphics-end-position
	// "\033[?8452h" sets the cursor next right to the bottom of the image instead of below
	// this prevents vertical scrolling when the image fills the last line.
	// horizontal scrolling because of this did not happen in my test cases.
	// "\033[?80l" disables sixel scrolling if it isn't already.
	//img.inband[term.Name()] = fmt.Sprintf("\033[%d;%dH%s", bounds.Min.Y+1, bounds.Min.X+1, sixelString)
	sixelString = fmt.Sprintf("\033[%d;%dH%s", bounds.Min.Y+1, bounds.Min.X+1, sixelString)
	// timg.SetInband(bounds, sixelString, d, term) // TODO uncomment
	// test string "HI"
	// wdgt.Block.ANSIString = fmt.Sprintf("\033[%d;%dH\033[?8452h%s", wdgt.Inner.Min.Y+1, wdgt.Inner.Min.X+1, "\033Pq#0;2;0;0;0#1;2;100;100;0#2;2;0;100;0#1~~@@vv@@~~@@~~$#2??}}GG}}??}}??-#1!14@\033\\")
	timg.SetInband(bounds, sixelString, d, term)

	return sixelString, nil
}
