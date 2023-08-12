// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

import (
	"image"
	"image/color"
)

type BGR565 struct {
	Pix    []byte
	Rect   image.Rectangle
	Stride int
}

func (i *BGR565) Bounds() image.Rectangle { return i.Rect }
func (i *BGR565) ColorModel() color.Model { return RGB565Model }

func (i *BGR565) At(x, y int) color.Color {
	if !(image.Point{x, y}.In(i.Rect)) {
		return RGBColor{}
	}

	pix := i.Pix[i.PixOffset(x, y):]
	clr := uint16(pix[0])<<8 | uint16(pix[1])

	return RGBColor{
		uint8(clr) & mask5,
		uint8(clr>>6) & mask6,
		uint8(clr>>11) & mask5,
	}
}

func (i *BGR565) Set(x, y int, c color.Color) {
	i.SetRGB(x, y, RGB565Model.Convert(c).(RGBColor))
}

func (i *BGR565) SetRGB(x, y int, c RGBColor) {
	if !(image.Point{x, y}.In(i.Rect)) {
		return
	}

	n := i.PixOffset(x, y)
	pix := i.Pix[n:]
	clr := uint16(c.G<<11) | uint16(c.G<<6) | uint16(c.R)

	pix[0] = uint8(clr >> 8)
	pix[1] = uint8(clr)
}

func (i *BGR565) PixOffset(x, y int) int {
	return (y-i.Rect.Min.Y)*i.Stride + (x-i.Rect.Min.X)*2
}
