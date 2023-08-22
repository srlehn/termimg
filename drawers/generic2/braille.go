package generic2

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/encoder/encpng"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/term"
)

// func init() { term.RegisterDrawer(&drawerBraille{}) }

var _ term.Drawer = (*drawerBraille)(nil)

type drawerBraille struct{}

func (d *drawerBraille) Name() string     { return `generic2` }
func (d *drawerBraille) New() term.Drawer { return &drawerBraille{} }

func (d *drawerBraille) IsApplicable(inp term.DrawerCheckerInput) bool { return true }

func (d *drawerBraille) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	if d == nil || tm == nil || img == nil {
		return errors.New(`nil parameter`)
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errors.New(consts.ErrNilImage)
	}

	boundsBraille := image.Rect(bounds.Min.X*2, bounds.Min.Y*4, bounds.Max.X*2, bounds.Max.Y*4)
	rsz := tm.Resizer()
	if rsz == nil {
		return errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return err
	}

	//

	var monochrome bool = true // TODO colored picture is not recognizable

	cimg := util.Must2(rsz.Resize(timg.Cropped, image.Pt(boundsBraille.Dx(), boundsBraille.Dy())))

	doAvgColors := true
	g := newGray2From(cimg, doAvgColors)

	rBase := '\u2800'
	b := &strings.Builder{}
	for y := 0; y < bounds.Dy(); y++ {
		b.WriteString(fmt.Sprintf("\033[%d;%dH", bounds.Min.Y+y, bounds.Min.X))
		for x := 0; x < bounds.Dx(); x++ {
			pxlRepr := 0 |
				g.GrayAt(x*2, y*4).Y/255 | g.GrayAt(x*2, y*4+1).Y/255<<1 | g.GrayAt(x*2+0, y*4+2).Y/255<<2 | g.GrayAt(x*2, y*4+3).Y/255<<3 |
				g.GrayAt(x*2+1, y*4).Y/255<<4 | g.GrayAt(x*2+1, y*4+1).Y/255<<5 | g.GrayAt(x*2+1, y*4+2).Y/255<<6 | g.GrayAt(x*2+1, y*4+3).Y/255<<7
			// https://github.com/iirelu/braillify/blob/master/src/braille.rs
			rBr := rBase + rune(0|
				(pxlRepr<<0&0b00000001)| // Moving _______X to _______X
				(pxlRepr<<0&0b00000010)| // Moving ______X_ to ______X_
				(pxlRepr<<0&0b00000100)| // Moving _____X__ to _____X__
				(pxlRepr<<3&0b01000000)| // Moving ____X___ to _X______
				(pxlRepr>>1&0b00001000)| // Moving ___X____ to ____X___
				(pxlRepr>>1&0b00010000)| // Moving __X_____ to ___X____
				(pxlRepr>>1&0b00100000)| // Moving _X______ to __X_____
				(pxlRepr<<0&0b10000000), // Moving X_______ to X_______
			)
			var fgr, fgg, fgb, bgr, bgg, bgb uint64
			var fgPxlCnt, bgPxlCnt int
			var fg, bg color.RGBA
			for cy := 0; cy < 4; cy++ {
				for cx := 0; cx < 2; cx++ {
					idx := (cx*4 + cy)
					r, g, b, _ := cimg.At(x*2+cx, y*4+cy).RGBA()
					switch pxlRepr >> idx & 1 {
					case 0:
						bgr += uint64(r * r)
						bgg += uint64(g * g)
						bgb += uint64(b * b)
						bgPxlCnt++
					case 1:
						fgr += uint64(r * r)
						fgg += uint64(g * g)
						fgb += uint64(b * b)
						fgPxlCnt++
					}
				}
			}
			if fgPxlCnt > 0 {
				fg = color.RGBA{
					R: uint8(math.Sqrt(float64(fgr) / float64(fgPxlCnt))),
					G: uint8(math.Sqrt(float64(fgg) / float64(fgPxlCnt))),
					B: uint8(math.Sqrt(float64(fgb) / float64(fgPxlCnt))),
				}
			}
			if bgPxlCnt > 0 {
				bg = color.RGBA{
					R: uint8(math.Sqrt(float64(bgr) / float64(bgPxlCnt))),
					G: uint8(math.Sqrt(float64(bgg) / float64(bgPxlCnt))),
					B: uint8(math.Sqrt(float64(bgb) / float64(bgPxlCnt))),
				}
			} else if fgPxlCnt > 0 {
				bg = fg
			}
			if fgPxlCnt == 0 && bgPxlCnt > 0 {
				fg = bg
			}
			if monochrome {
				b.WriteRune(rBr)
			} else {
				b.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm%c\033[0m", fg.R, fg.G, fg.B, bg.R, bg.G, bg.B, rBr))
			}
		}
	}
	str := b.String()
	_ = str
	fmt.Println(str)

	f2 := util.Must2(os.OpenFile(`test.png`, os.O_CREATE|os.O_RDWR, 0644))
	util.Must((&encpng.PngEncoder{}).Encode(f2, g, `.png`))
	f2.Close()

	return nil
}
