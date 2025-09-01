package kitty

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"strings"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/mux"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerKitty{}) }

var _ term.Drawer = (*drawerKitty)(nil)

const kittyLimit = 4096

type drawerKitty struct {
	// imageCount int // TODO: for numbering of ids
}

func (d *drawerKitty) Name() string     { return `kitty` }
func (d *drawerKitty) New() term.Drawer { return &drawerKitty{} }

func (d *drawerKitty) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	if inp == nil {
		return false, nil
	}
	// TODO query if supported
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/#querying-support-and-available-transmission-mediums
	// example: <ESC>_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA<ESC>\<ESC>[c

	switch inp.Name() {
	case `kitty`:
		// `wayst`: // untested
		return true, nil
	case `apple`,
		`iterm2`, // changes window title
		`urxvt`,
		`terminology`,
		`darktile`:
		// TODO bugged parsing
		return false, nil
	}

	// TODO disable on Terminology
	repl, err := term.CachedQuery(inp, queries.KittyTest+queries.DA1, inp, parser.NewParser(false, true), inp, inp)
	ret := err == nil &&
		(len(strings.SplitN(repl, queries.ST, 2)) == 2 ||
			len(strings.SplitN(repl, "\a", 2)) == 2)
	return ret, nil
}

func (d *drawerKitty) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerKitty) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
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

	// TODO get inband

	tcw, tch, err := tm.SizeInCells()
	if err != nil {
		return nil, err
	}
	if tcw == 0 || tch == 0 {
		return nil, errors.New("could not query terminal dimensions")
	}

	var imgHeight uint
	if bounds.Max.Y < int(tch) {
		imgHeight = uint(bounds.Dy())
	} else {
		imgHeight = tch - 1
	}
	// imgHeight = uint(bounds.Dy())

	// https://sw.kovidgoyal.net/kitty/graphics-protocol.html#remote-client
	// https://sw.kovidgoyal.net/kitty/graphics-protocol.html#png-data
	// https://sw.kovidgoyal.net/kitty/graphics-protocol.html#controlling-displayed-image-layout
	bytBuf := new(bytes.Buffer)
	if err = png.Encode(bytBuf, img); err != nil {
		return nil, err
	}
	imgBase64 := base64.StdEncoding.EncodeToString(bytBuf.Bytes())
	lenImgB64 := len([]byte(imgBase64))
	// a=T           action
	// t=d           payload is (base64 encoded) data itself not a file location
	// f=100         format: 100 = PNG payload
	// o=z           data compression
	// X=...,Y=,,,   Upper left image corner in cell coordinates (starting with 1, 1)
	// c=...,r=...   image size in cell columns and rows
	// w=...,h=...   width & height (in pixels) of the image area to display   // TODO: Use this to let Kitty handle cropping!
	// z=0           z-index vertical stacking order of the image
	// m=[01]        0 last escape code chunk - 1 for all except the last
	var kittyString string
	var zIndex = 2 // draw over text
	settings := fmt.Sprintf("a=T,t=d,f=100,X=%d,Y=%d,c=%d,r=%d,z=%d,", bounds.Min.X+1, bounds.Min.Y+1, bounds.Dx(), imgHeight, zIndex)
	i := 0
	for ; i < (lenImgB64-1)/kittyLimit; i++ {
		// kittyString += mux.Wrap(fmt.Sprintf("\033_G%sm=1;%s\033\\", settings, imgBase64[i*kittyLimit:(i+1)*kittyLimit]), tm)
		kittyString += mux.Wrap(fmt.Sprintf("\033_G%sC=1,m=1;%s\033\\", settings, imgBase64[i*kittyLimit:(i+1)*kittyLimit]), tm)
		settings = ""
	}
	// kittyString += mux.Wrap(fmt.Sprintf("\033_G%sm=0;%s\033\\", settings, imgBase64[i*kittyLimit:lenImgB64]), tm)
	kittyString += mux.Wrap(fmt.Sprintf("\033_G%sC=1,m=0;%s\033\\", settings, imgBase64[i*kittyLimit:lenImgB64]), tm)
	kittyString = fmt.Sprintf("\033[%d;%dH%s", bounds.Min.Y+1, bounds.Min.X+1, kittyString)

	timg.SetInband(bounds, kittyString, d, tm)

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.Printf(`%s`, kittyString)
		return logx.Err(err, tm, slog.LevelInfo)
	}
	return drawFn, nil
}
