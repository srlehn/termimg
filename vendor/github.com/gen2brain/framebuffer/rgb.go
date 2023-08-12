// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

const (
	mask5 = 1<<5 - 1
	mask6 = 1<<6 - 1
)

type RGBColor struct {
	R, G, B uint8
}

func (c RGBColor) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8
	a = 0xff
	return
}
