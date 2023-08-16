package sixel

import (
	"bytes"
	"fmt"
	"image"
	"strings"

	errorsGo "github.com/go-errors/errors"

	sixel "github.com/mattn/go-sixel"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerSixel{}) }

var _ term.Drawer = (*drawerSixel)(nil)

type drawerSixel struct{}

func (d *drawerSixel) Name() string     { return `sixel` }
func (d *drawerSixel) New() term.Drawer { return &drawerSixel{} }

func (d *drawerSixel) IsApplicable(inp term.DrawerCheckerInput) bool {
	// example query: "\033[0c"
	// possible answer from the terminal (here xterm): "\033[[?63;1;2;4;6;9;15;22c", vte(?): ...62,9;c
	// the "4" signals that the terminal is capable of sixel
	// conhost.exe knows this sequence.
	if inp == nil {
		return false
	}
	sixelCapable, isSixelCapable := inp.Property(propkeys.SixelCapable)
	if !isSixelCapable && sixelCapable == `true` {
		return false
	}

	tn := inp.Name()
	for _, n := range []string{
		`domterm`,
		`st`,
		`tabby`, // no images
		`wayst`, // spews character salad
	} {
		if tn == n {
			return false // skip buggy implementations
		}
	}

	repl, err := term.CachedQuery(inp, mux.Wrap("\033[0c", inp), inp, term.StopOnAlpha, inp, nil)
	// TODO fix mintty - querying in general on windows?
	if err != nil {
		return false
	}
	if len(repl) == 0 || repl[len(repl)-1] != 'c' {
		return false
	}
	repl = repl[:len(repl)-1]
	termCapabilities := strings.Split(repl, `;`)[1:]
	for _, cap := range termCapabilities {
		if cap == `4` {
			return true
		}
	}
	return false
}

func (d *drawerSixel) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	if d == nil || tm == nil || img == nil {
		return errorsGo.New(`nil parameter`)
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errorsGo.New(internal.ErrNilImage)
	}

	rsz := tm.Resizer()
	if rsz == nil {
		return errorsGo.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return err
	}

	sixelString, err := d.getInbandString(timg, bounds, tm)
	if err != nil {
		return err
	}

	tm.WriteString(sixelString)

	return nil
}

func (d *drawerSixel) getInbandString(timg *term.Image, bounds image.Rectangle, term *term.Terminal) (string, error) {
	if timg == nil {
		return ``, errorsGo.New(internal.ErrNilImage)
	}
	sixelString, err := timg.GetInband(bounds, d, term)
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
