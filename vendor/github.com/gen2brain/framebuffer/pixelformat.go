// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

import "math"

// List of known image/pixel formats.
const (
	PF_UNKNOWN = iota
	PF_RGBA    // 32-bit color
	PF_BGRA    // 32-bit color
	PF_RGB_555 // 16-bit color
	PF_RGB_565 // 16-bit color
	PF_BGR_555 // 16-bit color
	PF_BGR_565 // 16-bit color
	PF_INDEXED // 8-bit color (grayscale or paletted).
)

// PixelFormat describes the color layout of a single pixel
// in a given pixel buffer. Specifically, how many and which bits
// are occupied by a given color channel.
//
// For example, a standard RGBA pixel would look like this:
//
//           | bit 31                 bit 0 |
//           |                              |
//    pixel: rrrrrrrrggggggggbbbbbbbbaaaaaaaa
//
// The PixelFormat for this looks as follows:
//
//    red bits:  8
//    red shift: 24
//
//    green bits:  8
//    green shift: 16
//
//    blue bits:  8
//    blue shift: 8
//
//    alpha bits:  8
//    alpha shift: 0
//
//
// We can extract the channel information as follows:
//
//    red_mask := (1 << red_bits) - 1
//    green_mask := (1 << green_bits) - 1
//    blue_mask := (1 << blue_bits) - 1
//    alpha_mask := (1 << alpha_bits) - 1
//
//    r := (pixel >> red_shift) & red_mask
//    g := (pixel >> green_shift) & green_mask
//    b := (pixel >> blue_shift) & blue_mask
//    a := (pixel >> alpha_shift) & alpha_mask
//
type PixelFormat struct {
	RedBits    uint8 // Bit count for the red channel.
	RedShift   uint8 // Shift offset for the red channel.
	GreenBits  uint8 // Bit count for the green channel.
	GreenShift uint8 // Shift offset for the green channel.
	BlueBits   uint8 // Bit count for the blue channel.
	BlueShift  uint8 // Shift offset for the blue channel.
	AlphaBits  uint8 // Bit count for the alpha channel.
	AlphaShift uint8 // Shift offset for the alpha channel.
}

// Stride returns the width, in bytes, for a single pixel.
func (p PixelFormat) Stride() int {
	return int(math.Ceil(float64(p.RedBits+p.GreenBits+p.BlueBits+p.AlphaBits) / 8))
}

// Type returns an integer constant from the PF_XXX list, which
// identifies the type of pixelformat.
func (p PixelFormat) Type() int {
	switch p.Stride() {
	case 4: // 32-bit color
		if p.RedShift > p.BlueShift {
			return PF_BGRA
		}

		return PF_RGBA

	case 2: // 16-bit color
		if p.RedShift > p.BlueShift {
			if p.RedBits == 5 && p.GreenBits == 6 && p.BlueBits == 5 {
				return PF_BGR_565
			}

			if p.RedBits == 5 && p.GreenBits == 5 && p.BlueBits == 5 {
				return PF_BGR_555
			}
		}

		if p.RedBits == 5 && p.GreenBits == 6 && p.BlueBits == 5 {
			return PF_RGB_565
		}

		if p.RedBits == 5 && p.GreenBits == 5 && p.BlueBits == 5 {
			return PF_RGB_555
		}

	case 1: // 8-bit color
		return PF_INDEXED
	}

	return PF_UNKNOWN
}
