package domterm

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"image"
	"image/jpeg"
	"log/slog"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerDomTerm{}) }

var _ term.Drawer = (*drawerDomTerm)(nil)

type drawerDomTerm struct{}

func (d *drawerDomTerm) Name() string     { return `domterm` }
func (d *drawerDomTerm) New() term.Drawer { return &drawerDomTerm{} }

func (d *drawerDomTerm) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	return inp != nil && inp.Name() == `domterm`, nil
}

func (d *drawerDomTerm) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerDomTerm) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
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

	domTermString, err := d.inbandString(timg, bounds, tm)
	if err != nil {
		return nil, err
	}

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.WriteString(domTermString)
		return logx.Err(err, tm, slog.LevelInfo)
	}
	return drawFn, nil
}

func (d *drawerDomTerm) inbandString(timg *term.Image, bounds image.Rectangle, tm *term.Terminal) (string, error) {
	if timg == nil {
		return ``, errors.New(consts.ErrNilImage)
	}
	domTermString, err := timg.Inband(bounds, d, tm)
	if err == nil {
		return domTermString, nil
	}
	rsz := tm.Resizer()
	if rsz == nil {
		return ``, errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return ``, err
	}

	// encode
	// png seems to cause freeze
	buf := new(bytes.Buffer)
	// if err := jpeg.Encode(buf, timg.Prepared, &jpeg.Options{Quality: 100}); err != nil {
	if err := jpeg.Encode(buf, timg.Cropped, &jpeg.Options{Quality: 50}); err != nil {
		return ``, err
	}
	mimeType := `image/jpeg`

	imgBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// https://domterm.org/Wire-byte-protocol.html#Miscellaneous-sequences
	// valid attribute names: "alt", "longdesc", "height", "width", "border", "hspace", "vspace", "class"
	var attrs string
	if len(timg.FileName) > 0 {
		attrs += `alt='` + html.EscapeString(timg.FileName) + `" `
	}
	attrs += `class='` + consts.LibraryName + `' `
	attrs += fmt.Sprintf(`width='%d' height='%d'`, timg.Cropped.Bounds().Dx(), timg.Cropped.Bounds().Dy())
	domTermString = mux.Wrap(fmt.Sprintf("\033]72;<img %s src='data:%s;base64,%s\n'/>\a", attrs, mimeType, imgBase64), tm)
	domTermString = fmt.Sprintf("\033[%d;%dH%s%s", bounds.Min.Y, bounds.Min.X, ` `, domTermString) // TODO
	timg.SetInband(bounds, domTermString, d, tm)

	return domTermString, nil
}
