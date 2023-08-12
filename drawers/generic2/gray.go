package generic2

import (
	"image"
	"image/color"
	"image/draw"
)

type imageBlackWhite struct {
	gray     image.Gray
	avgColor color.Gray
}

var _ image.Image = (*imageBlackWhite)(nil)
var _ draw.Image = (*imageBlackWhite)(nil)
var _ color.Model = (*imageBlackWhite)(nil)

func newGray2From(img image.Image, doAvgColors bool) *imageBlackWhite {
	if img == nil {
		return nil
	}
	ret := &imageBlackWhite{}
	bounds := img.Bounds()
	if doAvgColors {
		_ = ret.averageColor(img)
	}
	g := image.NewGray(bounds)
	ret.gray = *g
	draw.Draw(ret, bounds, img, bounds.Min, draw.Src)

	return ret
}

func (m *imageBlackWhite) Set(x, y int, c color.Color) { m.gray.Set(x, y, m.Convert(c)) }
func (m *imageBlackWhite) At(x, y int) color.Color     { return m.gray.At(x, y) }
func (m *imageBlackWhite) GrayAt(x, y int) color.Gray  { return m.gray.GrayAt(x, y) }
func (m *imageBlackWhite) Bounds() image.Rectangle     { return m.gray.Bounds() }
func (m *imageBlackWhite) ColorModel() color.Model     { return m }

func (m *imageBlackWhite) averageColor(img image.Image) color.Gray {
	if img == nil {
		if m.avgColor.Y == 0 {
			m.avgColor.Y = 128
		}
		return m.avgColor
	}
	var sum uint64
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			cg, ok := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			if ok {
				sum += uint64(cg.Y)
				// } else { // TODO
			}
		}
	}
	m.avgColor.Y = uint8(sum / (uint64(width) * uint64(height)))
	return m.avgColor
}

func (m *imageBlackWhite) Convert(c color.Color) color.Color {
	avgColor := m.averageColor(nil).Y
	var y uint8
	cg, ok := color.GrayModel.Convert(c).(color.Gray)
	if ok {
		y = cg.Y
		// } else { // TODO
	}
	var ret color.Gray
	if y > avgColor {
		ret.Y = 255
	}
	return ret
}
