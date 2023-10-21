package generic

import (
	"context"
	"fmt"
	"image"
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

////////////////////////////////////////////////////////////////////////////////

func init() { term.RegisterDrawer(&DrawerGeneric{}) }

type DrawerGeneric struct{}

func (d *DrawerGeneric) Name() string     { return consts.DrawerGenericName }
func (d *DrawerGeneric) New() term.Drawer { return &DrawerGeneric{} }
func (d *DrawerGeneric) IsApplicable(term.DrawerCheckerInput) (bool, environ.Properties) {
	return true, nil
}

/*
TODO fails on urxvt
TODO stretch
*/

func (d *DrawerGeneric) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *DrawerGeneric) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
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

	blochCharString, err := d.inbandString(timg, bounds, tm)
	if err != nil {
		return nil, err
	}

	logx.Info(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		_, err := tm.WriteString(blochCharString)
		return logx.Err(err, tm, slog.LevelInfo)
	}

	return drawFn, nil
}

func (d *DrawerGeneric) inbandString(timg *term.Image, bounds image.Rectangle, tm *term.Terminal) (string, error) {
	if timg == nil {
		return ``, errors.New(consts.ErrNilImage)
	}
	blochCharString, err := timg.Inband(bounds, d, tm)
	if err == nil {
		return blochCharString, nil
	}
	if err := timg.Decode(); err != nil {
		return ``, err
	}
	cpw, cph, err := tm.CellSize()
	if err != nil {
		return ``, err
	}
	var aspectRatio float64
	if cph > 0 && cpw > 0 {
		aspectRatio = cpw / cph
	} else {
		aspectRatio = 1.5 // TODO guess
	}
	im := &imgBlock{
		image:       timg.Original,
		aspectRatio: aspectRatio,
		colors:      TrueColor,
	}

	blochCharString = im.Draw(bounds)
	timg.SetInband(bounds, blochCharString, d, tm)

	return blochCharString, nil
}

////////////////////////////////////////////////////////////////////////////////

// code ripped out from tview´s image widget and its dependency tcell
// https://github.com/rivo/tview commit 4a1f85b, MIT License
// https://github.com/rivo/tview/blob/4a1f85b/image.go#L439 , etc
// https://github.com/gdamore/tcell commit c951371, Apache-2.0 License

type imgBlock struct {
	// The image to be displayed. If nil, the widget will be empty.
	image image.Image

	// The size of the image. If a value is 0, the corresponding size is chosen
	// automatically based on the other size while preserving the image's aspect
	// ratio. If both are 0, the image uses as much space as possible. A
	// negative value represents a percentage, e.g. -50 means 50% of the
	// available space.
	width, height int

	// The number of colors to use. If 0, the number of colors is chosen based
	// on the terminal's capabilities.
	colors int

	// The dithering algorithm to use, one of the constants starting with
	// "ImageDithering".
	dithering int

	// The width of a terminal's cell divided by its height.
	aspectRatio float64

	// Horizontal and vertical alignment, one of the "Align" constants.
	// alignHorizontal, alignVertical int

	// The actual image size (in cells) when it was drawn the last time.
	lastWidth, lastHeight int

	// The actual image (in cells) when it was drawn the last time. The size of
	// this slice is lastWidth * lastHeight, indexed by y*lastWidth + x.
	pixels []pixel
}

// Colors returns the number of colors that will be used while drawing the
// image. This is one of the values listed in [Image.SetColors], except 0 which
// will be replaced by the actual number of colors used.
func (i *imgBlock) Colors() int {
	switch {
	case i.colors == 0:
		return availableColors
	case i.colors <= 2:
		return 2
	case i.colors <= 8:
		return 8
	case i.colors <= 256:
		return 256
	}
	return TrueColor
}

// render re-populates the [Image.pixels] slice based on the current settings,
// if [Image.lastWidth] and [Image.lastHeight] don't match the current image's
// size. It also sets the new image size in these two variables.
func (i *imgBlock) render(bounds image.Rectangle) {
	// If there is no image, there are no pixels.
	if i.image == nil {
		i.pixels = nil
		return
	}

	// Calculate the new (terminal-space) image size.
	imgBounds := i.image.Bounds()
	imageWidth, imageHeight := imgBounds.Dx(), imgBounds.Dy()
	if i.aspectRatio != 1.0 {
		imageWidth = int(float64(imageWidth) / i.aspectRatio)
	}
	width, height := i.width, i.height
	innerWidth, innerHeight := bounds.Dx(), bounds.Dy()
	if innerWidth <= 0 {
		i.pixels = nil
		return
	}
	if width == 0 && height == 0 {
		// Use all available space.
		width, height = innerWidth, innerHeight
		if adjustedWidth := imageWidth * height / imageHeight; adjustedWidth < width {
			width = adjustedWidth
		} else {
			height = imageHeight * width / imageWidth
		}
	} else {
		// Turn percentages into absolute values.
		if width < 0 {
			width = innerWidth * -width / 100
		}
		if height < 0 {
			height = innerHeight * -height / 100
		}
		if width == 0 {
			// Adjust the width.
			width = imageWidth * height / imageHeight
		} else if height == 0 {
			// Adjust the height.
			height = imageHeight * width / imageWidth
		}
	}
	if width <= 0 || height <= 0 {
		i.pixels = nil
		return
	}

	// If nothing has changed, we're done.
	if i.lastWidth == width && i.lastHeight == height {
		return
	}
	i.lastWidth, i.lastHeight = width, height // This could still be larger than the available space but that's ok for now.

	// Generate the initial pixels by resizing the image (8x8 per cell).
	pixels := i.resize()

	// Turn them into block elements with background/foreground colors.
	i.stamp(pixels)
}

// resize resizes the image to the current size and returns the result as a
// slice of pixels. It is assumed that [Image.lastWidth] (w) and
// [Image.lastHeight] (h) are positive, non-zero values, and the slice has a
// size of 64*w*h, with each pixel being represented by 3 float64 values in the
// range of 0-1. The factor of 64 is due to the fact that we calculate 8x8
// pixels per cell.
func (i *imgBlock) resize() [][3]float64 {
	// Because most of the time, we will be downsizing the image, we don't even
	// attempt to do any fancy interpolation. For each target pixel, we
	// calculate a weighted average of the source pixels using their coverage
	// area.

	bounds := i.image.Bounds()
	srcWidth, srcHeight := bounds.Dx(), bounds.Dy()
	tgtWidth, tgtHeight := i.lastWidth*8, i.lastHeight*8
	coverageWidth, coverageHeight := float64(tgtWidth)/float64(srcWidth), float64(tgtHeight)/float64(srcHeight)
	pixels := make([][3]float64, tgtWidth*tgtHeight)
	weights := make([]float64, tgtWidth*tgtHeight)
	for srcY := bounds.Min.Y; srcY < bounds.Max.Y; srcY++ {
		for srcX := bounds.Min.X; srcX < bounds.Max.X; srcX++ {
			r32, g32, b32, _ := i.image.At(srcX, srcY).RGBA()
			r, g, b := float64(r32)/0xffff, float64(g32)/0xffff, float64(b32)/0xffff

			// Iterate over all target pixels. Outer loop is Y.
			startY := float64(srcY-bounds.Min.Y) * coverageHeight
			endY := startY + coverageHeight
			fromY, toY := int(startY), int(endY)
			for tgtY := fromY; tgtY <= toY && tgtY < tgtHeight; tgtY++ {
				coverageY := 1.0
				if tgtY == fromY {
					coverageY -= math.Mod(startY, 1.0)
				}
				if tgtY == toY {
					coverageY -= 1.0 - math.Mod(endY, 1.0)
				}

				// Inner loop is X.
				startX := float64(srcX-bounds.Min.X) * coverageWidth
				endX := startX + coverageWidth
				fromX, toX := int(startX), int(endX)
				for tgtX := fromX; tgtX <= toX && tgtX < tgtWidth; tgtX++ {
					coverageX := 1.0
					if tgtX == fromX {
						coverageX -= math.Mod(startX, 1.0)
					}
					if tgtX == toX {
						coverageX -= 1.0 - math.Mod(endX, 1.0)
					}

					// Add a weighted contribution to the target pixel.
					index := tgtY*tgtWidth + tgtX
					coverage := coverageX * coverageY
					pixels[index][0] += r * coverage
					pixels[index][1] += g * coverage
					pixels[index][2] += b * coverage
					weights[index] += coverage
				}
			}
		}
	}

	// Normalize the pixels.
	for index, weight := range weights {
		if weight > 0 {
			pixels[index][0] /= weight
			pixels[index][1] /= weight
			pixels[index][2] /= weight
		}
	}

	return pixels
}

// stamp takes the pixels generated by [Image.resize] and populates the
// [Image.pixels] slice accordingly.
func (i *imgBlock) stamp(resized [][3]float64) {
	// For each 8x8 pixel block, we find the best block element to represent it,
	// given the available colors.
	i.pixels = make([]pixel, i.lastWidth*i.lastHeight)
	colors := i.Colors()
	for row := 0; row < i.lastHeight; row++ {
		for col := 0; col < i.lastWidth; col++ {
			// Calculate an error for each potential block element + color. Keep
			// the one with the lowest error.

			// Note that the values in "resize" may lie outside [0, 1] due to
			// the error distribution during dithering.

			minMSE := math.MaxFloat64 // Mean squared error.
			var final [64][3]float64  // The final pixel values.
			for element, bits := range blockElements {
				// Calculate the average color for the pixels covered by the set
				// bits and unset bits.
				var (
					bg, fg  [3]float64
					setBits float64
					bit     uint64 = 1
				)
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
						if bits&bit != 0 {
							fg[0] += resized[index][0]
							fg[1] += resized[index][1]
							fg[2] += resized[index][2]
							setBits++
						} else {
							bg[0] += resized[index][0]
							bg[1] += resized[index][1]
							bg[2] += resized[index][2]
						}
						bit <<= 1
					}
				}
				for ch := 0; ch < 3; ch++ {
					fg[ch] /= setBits
					if fg[ch] < 0 {
						fg[ch] = 0
					} else if fg[ch] > 1 {
						fg[ch] = 1
					}
					bg[ch] /= 64 - setBits
					if bg[ch] < 0 {
						bg[ch] = 0
					}
					if bg[ch] > 1 {
						bg[ch] = 1
					}
				}

				// Quantize to the nearest acceptable color.
				for _, color := range []*[3]float64{&fg, &bg} {
					if colors == 2 {
						// Monochrome. The following weights correspond better
						// to human perception than the arithmetic mean.
						gray := 0.299*color[0] + 0.587*color[1] + 0.114*color[2]
						if gray < 0.5 {
							*color = [3]float64{0, 0, 0}
						} else {
							*color = [3]float64{1, 1, 1}
						}
					} else {
						for index, ch := range color {
							switch {
							case colors == 8:
								// Colors vary wildly for each terminal. Expect
								// suboptimal results.
								if ch < 0.5 {
									color[index] = 0
								} else {
									color[index] = 1
								}
							case colors == 256:
								color[index] = math.Round(ch*6) / 6
							}
						}
					}
				}

				// Calculate the error (and the final pixel values).
				var (
					mse         float64
					values      [64][3]float64
					valuesIndex int
				)
				bit = 1
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						if bits&bit != 0 {
							values[valuesIndex] = fg
						} else {
							values[valuesIndex] = bg
						}
						index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
						for ch := 0; ch < 3; ch++ {
							err := resized[index][ch] - values[valuesIndex][ch]
							mse += err * err
						}
						bit <<= 1
						valuesIndex++
					}
				}

				// Do we have a better match?
				if mse < minMSE {
					// Yes. Save it.
					minMSE = mse
					final = values
					index := row*i.lastWidth + col
					i.pixels[index].element = element
					i.pixels[index].style = styleDefault.
						SetForeground(newRGBColor(int32(math.Min(255, fg[0]*255)), int32(math.Min(255, fg[1]*255)), int32(math.Min(255, fg[2]*255)))).
						SetBackground(newRGBColor(int32(math.Min(255, bg[0]*255)), int32(math.Min(255, bg[1]*255)), int32(math.Min(255, bg[2]*255))))
				}
			}

			// Check if there is a shade block which results in a smaller error.

			// What's the overall average color?
			var avg [3]float64
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
					for ch := 0; ch < 3; ch++ {
						avg[ch] += resized[index][ch] / 64
					}
				}
			}
			for ch := 0; ch < 3; ch++ {
				if avg[ch] < 0 {
					avg[ch] = 0
				} else if avg[ch] > 1 {
					avg[ch] = 1
				}
			}

			// Quantize and choose shade element.
			element := BlockFullBlock
			var fg, bg colr
			shades := []rune{' ', BlockLightShade, BlockMediumShade, BlockDarkShade, BlockFullBlock}
			if colors == 2 {
				// Monochrome.
				gray := 0.299*avg[0] + 0.587*avg[1] + 0.114*avg[2] // See above for details.
				shade := int(math.Round(gray * 4))
				element = shades[shade]
				for ch := 0; ch < 3; ch++ {
					avg[ch] = float64(shade) / 4
				}
				bg = ColorBlack
				fg = ColorWhite
			} else if colors == TrueColor {
				// True color.
				fg = newRGBColor(int32(math.Min(255, avg[0]*255)), int32(math.Min(255, avg[1]*255)), int32(math.Min(255, avg[2]*255)))
				bg = fg
			} else {
				// 8 or 256 colors.
				steps := 1.0
				if colors == 256 {
					steps = 6.0
				}
				var (
					lo, hi, pos [3]float64
					shade       float64
				)
				for ch := 0; ch < 3; ch++ {
					lo[ch] = math.Floor(avg[ch]*steps) / steps
					hi[ch] = math.Ceil(avg[ch]*steps) / steps
					if r := hi[ch] - lo[ch]; r > 0 {
						pos[ch] = (avg[ch] - lo[ch]) / r
						if math.Abs(pos[ch]-0.5) < math.Abs(shade-0.5) {
							shade = pos[ch]
						}
					}
				}
				shade = math.Round(shade * 4)
				element = shades[int(shade)]
				shade /= 4
				for ch := 0; ch < 3; ch++ { // Find the closest channel value.
					best := math.Abs(avg[ch] - (lo[ch] + (hi[ch]-lo[ch])*shade)) // Start shade from lo to hi.
					if value := math.Abs(avg[ch] - (hi[ch] - (hi[ch]-lo[ch])*shade)); value < best {
						best = value // Swap lo and hi.
						lo[ch], hi[ch] = hi[ch], lo[ch]
					}
					if value := math.Abs(avg[ch] - lo[ch]); value < best {
						best = value // Use lo.
						hi[ch] = lo[ch]
					}
					if value := math.Abs(avg[ch] - hi[ch]); value < best {
						lo[ch] = hi[ch] // Use hi.
					}
					avg[ch] = lo[ch] + (hi[ch]-lo[ch])*shade // Quantize.
				}
				bg = newRGBColor(int32(math.Min(255, lo[0]*255)), int32(math.Min(255, lo[1]*255)), int32(math.Min(255, lo[2]*255)))
				fg = newRGBColor(int32(math.Min(255, hi[0]*255)), int32(math.Min(255, hi[1]*255)), int32(math.Min(255, hi[2]*255)))
			}

			// Calculate the error (and the final pixel values).
			var (
				mse         float64
				values      [64][3]float64
				valuesIndex int
			)
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					index := (row*8+y)*i.lastWidth*8 + (col*8 + x)
					for ch := 0; ch < 3; ch++ {
						err := resized[index][ch] - avg[ch]
						mse += err * err
					}
					values[valuesIndex] = avg
					valuesIndex++
				}
			}

			// Is this shade element better than the block element?
			if mse < minMSE {
				// Yes. Save it.
				final = values
				index := row*i.lastWidth + col
				i.pixels[index].element = element
				i.pixels[index].style = styleDefault.SetForeground(fg).SetBackground(bg)
			}

			// Apply dithering.
			if colors < TrueColor && i.dithering == DitheringFloydSteinberg {
				// The dithering mask determines how the error is distributed.
				// Each element has three values: dx, dy, and weight (in 16th).
				var mask = [4][3]int{
					{1, 0, 7},
					{-1, 1, 3},
					{0, 1, 5},
					{1, 1, 1},
				}

				// We dither the 8x8 block as a 2x2 block, transferring errors
				// to its 2x2 neighbors.
				for ch := 0; ch < 3; ch++ {
					for y := 0; y < 2; y++ {
						for x := 0; x < 2; x++ {
							// What's the error for this 4x4 block?
							var err float64
							for dy := 0; dy < 4; dy++ {
								for dx := 0; dx < 4; dx++ {
									err += (final[(y*4+dy)*8+(x*4+dx)][ch] - resized[(row*8+(y*4+dy))*i.lastWidth*8+(col*8+(x*4+dx))][ch]) / 16
								}
							}

							// Distribute it to the 2x2 neighbors.
							for _, dist := range mask {
								for dy := 0; dy < 4; dy++ {
									for dx := 0; dx < 4; dx++ {
										targetX, targetY := (x+dist[0])*4+dx, (y+dist[1])*4+dy
										if targetX < 0 || col*8+targetX >= i.lastWidth*8 || targetY < 0 || row*8+targetY >= i.lastHeight*8 {
											continue
										}
										resized[(row*8+targetY)*i.lastWidth*8+(col*8+targetX)][ch] -= err * float64(dist[2]) / 16
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// Draw draws this primitive onto the screen.
func (i *imgBlock) Draw(bounds image.Rectangle) string {
	// Regenerate image if necessary.
	i.render(bounds)

	viewX, viewY := bounds.Min.X, bounds.Min.Y
	// Determine image placement.
	width, height := i.lastWidth, i.lastHeight // set in render()

	// Draw the image.
	b := &strings.Builder{}
	for row := 0; row < height; row++ {
		b.WriteString(fmt.Sprintf("\033[%d;%dH", viewY+row+1, viewX+1))
		for _, pxl := range i.pixels[row*width : (row+1)*width] {
			// TODO handle cases other than True Color
			fgr, fgg, fgb := pxl.style.fg.RGB()
			bgr, bgg, bgb := pxl.style.bg.RGB()
			b.WriteString(fmt.Sprintf(
				"\033[0;38;2;%d;%d;%d;48;2;%d;%d;%dm%c",
				fgr, fgg, fgb,
				bgr, bgg, bgb,
				pxl.element,
			))
		}
	}
	return b.String()
}

// Types of dithering applied to images.
const (
	DitheringNone           = iota // No dithering.
	DitheringFloydSteinberg        // Floyd-Steinberg dithering (the default).
)

// The number of colors available in the terminal.
var availableColors = 256

// The number of colors supported by true color terminals (R*G*B = 256*256*256).
const TrueColor = 256 * 256 * 256 // 16777216

// Text alignment within a box. Also used to align images.
const (
	AlignLeft = iota
	AlignCenter
	AlignRight
	AlignTop    = 0
	AlignBottom = 2
)

// pixel represents a character on screen used to draw part of an image.
type pixel struct {
	style   style
	element rune // The block element.
}

type style struct {
	fg colr
	bg colr
}

var styleDefault style

// SetForeground returns a new style based on s, with the foreground color set
// as requested.  ColorDefault can be used to select the global default.
func (s style) SetForeground(c colr) style {
	return style{
		fg: c,
		bg: s.bg,
	}
}

// SetBackground returns a new style based on s, with the background color set
// as requested.  ColorDefault can be used to select the global default.
func (s style) SetBackground(c colr) style {
	return style{
		fg: s.fg,
		bg: c,
	}
}

////////////////////////////////////////////////////////////////////////////////

// This map describes what each block element looks like. A 1 bit represents a
// pixel that is drawn, a 0 bit represents a pixel that is not drawn. The least
// significant bit is the top left pixel, the most significant bit is the bottom
// right pixel, moving row by row from left to right, top to bottom.
var blockElements = map[rune]uint64{
	BlockLowerOneEighthBlock:            0b1111111100000000000000000000000000000000000000000000000000000000,
	BlockLowerOneQuarterBlock:           0b1111111111111111000000000000000000000000000000000000000000000000,
	BlockLowerThreeEighthsBlock:         0b1111111111111111111111110000000000000000000000000000000000000000,
	BlockLowerHalfBlock:                 0b1111111111111111111111111111111100000000000000000000000000000000,
	BlockLowerFiveEighthsBlock:          0b1111111111111111111111111111111111111111000000000000000000000000,
	BlockLowerThreeQuartersBlock:        0b1111111111111111111111111111111111111111111111110000000000000000,
	BlockLowerSevenEighthsBlock:         0b1111111111111111111111111111111111111111111111111111111100000000,
	BlockLeftSevenEighthsBlock:          0b0111111101111111011111110111111101111111011111110111111101111111,
	BlockLeftThreeQuartersBlock:         0b0011111100111111001111110011111100111111001111110011111100111111,
	BlockLeftFiveEighthsBlock:           0b0001111100011111000111110001111100011111000111110001111100011111,
	BlockLeftHalfBlock:                  0b0000111100001111000011110000111100001111000011110000111100001111,
	BlockLeftThreeEighthsBlock:          0b0000011100000111000001110000011100000111000001110000011100000111,
	BlockLeftOneQuarterBlock:            0b0000001100000011000000110000001100000011000000110000001100000011,
	BlockLeftOneEighthBlock:             0b0000000100000001000000010000000100000001000000010000000100000001,
	BlockQuadrantLowerLeft:              0b0000111100001111000011110000111100000000000000000000000000000000,
	BlockQuadrantLowerRight:             0b1111000011110000111100001111000000000000000000000000000000000000,
	BlockQuadrantUpperLeft:              0b0000000000000000000000000000000000001111000011110000111100001111,
	BlockQuadrantUpperRight:             0b0000000000000000000000000000000011110000111100001111000011110000,
	BlockQuadrantUpperLeftAndLowerRight: 0b1111000011110000111100001111000000001111000011110000111100001111,
}

// https://github.com/rivo/tview/blob/4a1f85b/semigraphics.go#L143
// Semigraphics provides an easy way to access unicode characters for drawing.
// Named like the unicode characters, 'Semigraphics'-prefix used if unicode block
// isn't prefixed itself.
const (
	// Block Elements.
	BlockUpperHalfBlock                              rune = '\u2580' // ▀
	BlockLowerOneEighthBlock                         rune = '\u2581' // ▁
	BlockLowerOneQuarterBlock                        rune = '\u2582' // ▂
	BlockLowerThreeEighthsBlock                      rune = '\u2583' // ▃
	BlockLowerHalfBlock                              rune = '\u2584' // ▄
	BlockLowerFiveEighthsBlock                       rune = '\u2585' // ▅
	BlockLowerThreeQuartersBlock                     rune = '\u2586' // ▆
	BlockLowerSevenEighthsBlock                      rune = '\u2587' // ▇
	BlockFullBlock                                   rune = '\u2588' // █
	BlockLeftSevenEighthsBlock                       rune = '\u2589' // ▉
	BlockLeftThreeQuartersBlock                      rune = '\u258A' // ▊
	BlockLeftFiveEighthsBlock                        rune = '\u258B' // ▋
	BlockLeftHalfBlock                               rune = '\u258C' // ▌
	BlockLeftThreeEighthsBlock                       rune = '\u258D' // ▍
	BlockLeftOneQuarterBlock                         rune = '\u258E' // ▎
	BlockLeftOneEighthBlock                          rune = '\u258F' // ▏
	BlockRightHalfBlock                              rune = '\u2590' // ▐
	BlockLightShade                                  rune = '\u2591' // ░
	BlockMediumShade                                 rune = '\u2592' // ▒
	BlockDarkShade                                   rune = '\u2593' // ▓
	BlockUpperOneEighthBlock                         rune = '\u2594' // ▔
	BlockRightOneEighthBlock                         rune = '\u2595' // ▕
	BlockQuadrantLowerLeft                           rune = '\u2596' // ▖
	BlockQuadrantLowerRight                          rune = '\u2597' // ▗
	BlockQuadrantUpperLeft                           rune = '\u2598' // ▘
	BlockQuadrantUpperLeftAndLowerLeftAndLowerRight  rune = '\u2599' // ▙
	BlockQuadrantUpperLeftAndLowerRight              rune = '\u259A' // ▚
	BlockQuadrantUpperLeftAndUpperRightAndLowerLeft  rune = '\u259B' // ▛
	BlockQuadrantUpperLeftAndUpperRightAndLowerRight rune = '\u259C' // ▜
	BlockQuadrantUpperRight                          rune = '\u259D' // ▝
	BlockQuadrantUpperRightAndLowerLeft              rune = '\u259E' // ▞
	BlockQuadrantUpperRightAndLowerLeftAndLowerRight rune = '\u259F' // ▟
)
