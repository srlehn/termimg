package generic2

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
)

func init() {
	term.RegisterDrawer(&drawerGeneric2{
		monochrome:           false,
		useDistanceThreshold: true,
		distanceThreshold:    0.6875,
	})
}

var _ term.Drawer = (*drawerGeneric2)(nil)

type drawerGeneric2 struct {
	monochrome           bool
	useDistanceThreshold bool
	distanceThreshold    float64
}

func (d *drawerGeneric2) Name() string     { return `generic2` }
func (d *drawerGeneric2) New() term.Drawer { return &drawerGeneric2{} }

func (d *drawerGeneric2) IsApplicable(inp term.DrawerCheckerInput) (bool, environ.Properties) {
	// TODO disable sextants on xterm, terminology (font drawn)
	return true, nil
}

func (d *drawerGeneric2) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerGeneric2) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
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

	var (
		cellWidthPixels  uint = 2
		cellHeightPixels uint = 3
	)
	boundsPixelated := image.Rect(
		bounds.Min.X*int(cellWidthPixels), bounds.Min.Y*int(cellHeightPixels),
		bounds.Max.X*int(cellWidthPixels), bounds.Max.Y*int(cellHeightPixels),
	)
	rsz := tm.Resizer()
	if rsz == nil {
		return nil, errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return nil, err
	}

	//

	cimg, err := rsz.Resize(timg.Cropped, image.Pt(boundsPixelated.Dx(), boundsPixelated.Dy()))
	if err != nil {
		return nil, err
	}
	doAvgColors := true
	g := newGray2From(cimg, doAvgColors)

	b := &strings.Builder{}
	imgHeight := bounds.Dy()
	imgWidth := bounds.Dx()
	pix := make([][]coloredRune, imgHeight)
	var fgBgDistSum uint64
	var allPixelsAvgDistSum uint64
	for y := 0; y < bounds.Dy(); y++ {
		if d.monochrome || !d.useDistanceThreshold {
			b.WriteString(fmt.Sprintf("\033[%d;%dH", bounds.Min.Y+y+1, bounds.Min.X+1))
		}
		if !d.monochrome {
			pix[y] = make([]coloredRune, imgWidth)
		}
		for x := 0; x < bounds.Dx(); x++ {
			var pxlRepr uint8
			var rc coloredRune
			if d.monochrome {
				for cy := 0; cy < int(cellHeightPixels); cy++ {
					for cx := 0; cx < int(cellWidthPixels); cx++ {
						pxlRepr |= g.GrayAt(x*int(cellWidthPixels)+cx, y*int(cellHeightPixels)+cy).Y / 255 << (cx*int(cellHeightPixels) + cy)
					}
				}
			} else {
				var distMin uint64 = 1<<64 - 1
				for i := 0; i < 1<<(cellWidthPixels*cellHeightPixels); i++ {
					var fgPxlCnt, bgPxlCnt uint
					var fgTmp, bgTmp color.RGBA
					var colsFg, colsBg []color.Color
					for cy := 0; cy < int(cellHeightPixels); cy++ {
						for cx := 0; cx < int(cellWidthPixels); cx++ {
							idx := (cx*int(cellHeightPixels) + cy)
							col := cimg.At(x*int(cellWidthPixels)+cx, y*int(cellHeightPixels)+cy)
							switch i >> idx & 1 {
							case 0:
								colsBg = append(colsBg, col)
								bgPxlCnt++
							case 1:
								colsFg = append(colsFg, col)
								fgPxlCnt++
							}
						}
					}
					var avgDistSum, avgDistSumFg, avgDistSumBg uint64
					switch fgPxlCnt {
					case 0:
					case 1:
						fgTmp = colToRGB(colsFg[0])
					default:
						fgTmp, avgDistSumFg = getAvgDistSum(colsFg)
						avgDistSum += avgDistSumFg
					}
					switch bgPxlCnt {
					case 0:
						bgTmp = fgTmp
					case 1:
						bgTmp = colToRGB(colsBg[0])
					default:
						bgTmp, avgDistSumBg = getAvgDistSum(colsBg)
						avgDistSum += avgDistSumBg
					}
					if i == 0 {
						allPixelsAvgDistSum += avgDistSum
					}
					if avgDistSum >= distMin {
						continue
					}
					if fgPxlCnt == 0 {
						fgTmp = bgTmp
					}
					distMin = avgDistSum
					pxlRepr = uint8(i)
					// TODO calc sqrt here
					rc.fg = fgTmp
					rc.bg = bgTmp
					rc.fgPxlCnt = fgPxlCnt
					rc.bgPxlCnt = bgPxlCnt
					if rc.fgPxlCnt == 1 || rc.bgPxlCnt == 1 {
						rc.colsFg = colsFg
						rc.colsBg = colsBg
					}
				}
			}

			// TODO copy bit manipulation from
			// https://github.com/hapytex/unicode-tricks/blob/master/src/Data/Char/Block/Sextant.hs
			rPxl, okPxl := sextants[pxlRepr]
			if !okPxl {
				rPxl = ' '
			}
			if d.monochrome {
				b.WriteRune(rPxl)
			} else {
				rc.r = rPxl
				if d.useDistanceThreshold {
					pix[y][x] = rc
					fgBgDistSum += distQuad(rc.fg, rc.bg)
				} else {
					writeColoredChar(b, rc)
				}
			}
		}
	}
	if d.useDistanceThreshold {
		var allPixelsAvgDist float64
		if d.useDistanceThreshold {
			allPixelsAvgDist = float64(allPixelsAvgDistSum) / (float64(imgWidth * imgHeight))
		}
		for y, row := range pix {
			b.WriteString(fmt.Sprintf("\033[%d;%dH", bounds.Min.Y+y+1, bounds.Min.X+1))
			for _, cell := range row {
				if cell.fgPxlCnt == 1 || cell.bgPxlCnt == 1 {
					colAvg, avgDistSum := getAvgDistSum(append(cell.colsFg, cell.colsBg...))
					if avgDistSum < uint64(d.distanceThreshold*allPixelsAvgDist) {
						cell = coloredRune{
							fg: colAvg,
							// r:  ' ',
							r: sextants[1<<(cellWidthPixels*cellHeightPixels)-1], // full block
						}
					}
				}
				writeColoredChar(b, cell)
			}
		}
	}
	str := b.String()

	logx.Info(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.WriteString(str)
		return logx.Err(err, tm, slog.LevelInfo)
	}

	return drawFn, nil
}

func distQuad(c1, c2 color.Color) uint64 {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	r := int(r1) - int(r2)
	g := int(g1) - int(g2)
	b := int(b1) - int(b2)
	a := int(a1) - int(a2)
	// return uint64(math.Sqrt(float64(r*r + g*g + b*b + a*a)))
	return uint64(r*r + g*g + b*b + a*a)
}

func getAvgDistSum(cols []color.Color) (avgCol color.RGBA, avgDistSum uint64) {
	switch len(cols) {
	case 0:
		return
	case 1:
		return colToRGB(cols[0]), 0
	}
	var avgr, avgg, avgb, avga uint64
	for _, col := range cols {
		r, g, b, a := col.RGBA()
		avgr += uint64(r)
		avgg += uint64(g)
		avgb += uint64(b)
		avga += uint64(a)
	}
	n := uint64(len(cols))
	// TODO calc sqrt later
	avgCol = color.RGBA{
		R: capUInt8(math.Sqrt(float64(avgr / n))),
		G: capUInt8(math.Sqrt(float64(avgg / n))),
		B: capUInt8(math.Sqrt(float64(avgb / n))),
		A: capUInt8(math.Sqrt(float64(avga / n))),
	}
	for _, col := range cols {
		avgDistSum += distQuad(avgCol, col)
	}
	return
}

func colToRGB(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	rgba := color.RGBA{capUInt8(r >> 8), capUInt8(g >> 8), capUInt8(b >> 8), capUInt8(a >> 8)}
	return rgba
}

func capUInt8[N ~int | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64](n N) uint8 {
	var capped uint8
	switch {
	case n < 0:
	case n > 255:
		capped = 255
	default:
		capped = uint8(n)
	}
	return capped
}

type coloredRune struct {
	fg       color.RGBA
	bg       color.RGBA
	r        rune
	fgPxlCnt uint
	bgPxlCnt uint
	colsFg   []color.Color
	colsBg   []color.Color
}

func writeColoredChar(b *strings.Builder, rc coloredRune) {
	// true color
	b.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm%c\033[0m", rc.fg.R, rc.fg.G, rc.fg.B, rc.bg.R, rc.bg.G, rc.bg.B, rc.r))
}
