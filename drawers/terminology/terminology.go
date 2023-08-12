package terminology

import (
	"fmt"
	"image"
	"strings"

	"github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/encoder/encpng"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerTerminology{}) }

var _ term.Drawer = (*drawerTerminology)(nil)

type drawerTerminology struct{}

func (d *drawerTerminology) Name() string     { return `terminology` }
func (d *drawerTerminology) New() term.Drawer { return &drawerTerminology{} }

func (d *drawerTerminology) IsApplicable(inp term.DrawerCheckerInput) bool {
	return inp != nil && inp.Name() == `terminology`
}

func (d *drawerTerminology) Draw(img image.Image, bounds image.Rectangle, rsz term.Resizer, tm *term.Terminal) error {
	if d == nil {
		return errors.New(internal.ErrNilReceiver)
	}
	if tm == nil || img == nil {
		return errors.New(internal.ErrNilParam)
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errors.New(internal.ErrNilImage)
	}

	err := timg.Fit(bounds, rsz, tm)
	if err != nil {
		return err
	}

	terminologyString, err := d.getInbandString(timg, bounds, tm)
	if err != nil {
		return err
	}
	tm.WriteString(terminologyString)

	return nil
}

func (d *drawerTerminology) getInbandString(timg *term.Image, bounds image.Rectangle, term *term.Terminal) (string, error) {
	if timg == nil {
		return ``, errors.New(internal.ErrNilImage)
	}
	terminologyString, err := timg.GetInband(bounds, d, term)
	if err == nil {
		return terminologyString, nil
	}
	if timg.Cropped == nil {
		return ``, errors.New(internal.ErrNilImage)
	}
	w, h := uint(bounds.Dx()), uint(bounds.Dy())
	if w > 511 || h > 511 {
		// TODO tile image
		return ``, errors.New(internal.ErrNotImplemented)
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
