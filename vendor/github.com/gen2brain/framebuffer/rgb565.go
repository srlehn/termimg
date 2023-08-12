// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

import (
	"image"
	"image/color"
)

var RGB565Model = color.ModelFunc(
	func(c color.Color) color.Color {
		if _, ok := c.(RGBColor); ok {
			return c
		}

		r, g, b, _ := c.RGBA()
		return RGBColor{
			uint8(r>>8) & mask5,
			uint8(g>>8) & mask6,
			uint8(b>>8) & mask5,
		}
	})

type RGB565 struct {
	Pix    []byte
	Rect   image.Rectangle
	Stride int
}

func (i *RGB565) Bounds() image.Rectangle { return i.Rect }
func (i *RGB565) ColorModel() color.Model { return RGB565Model }

func (i *RGB565) At(x, y int) color.Color {
	if !(image.Point{x, y}.In(i.Rect)) {
		return RGBColor{}
	}

	pix := i.Pix[i.PixOffset(x, y):]
	clr := uint16(pix[0])<<8 | uint16(pix[1])

	return RGBColor{
		uint8(clr>>11) & mask5,
		uint8(clr>>6) & mask6,
		uint8(clr) & mask5,
	}
}

func (i *RGB565) Set(x, y int, c color.Color) {
	i.SetRGB(x, y, RGB565Model.Convert(c).(RGBColor))
}

func (i *RGB565) SetRGB(x, y int, c RGBColor) {
	if !(image.Point{x, y}.In(i.Rect)) {
		return
	}

	n := i.PixOffset(x, y)
	pix := i.Pix[n:]
	clr := uint16(c.R<<11) | uint16(c.G<<6) | uint16(c.B)

	pix[0] = uint8(clr >> 8)
	pix[1] = uint8(clr)
}

func (i *RGB565) PixOffset(x, y int) int {
	return (y-i.Rect.Min.Y)*i.Stride + (x-i.Rect.Min.X)*2
}
