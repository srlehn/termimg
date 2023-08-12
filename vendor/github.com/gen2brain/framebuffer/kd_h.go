// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

// <linux/kd.h>

const (
	_KD_GRAPHICS = 0x01
	_KDSETMODE   = 0x4B3A // set text/graphics mode
	_KDGETMODE   = 0x4B3B // get current mode
)
