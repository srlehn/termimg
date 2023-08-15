//go:build dev

package framebuffer

import "image/draw"

var _ draw.Image = (*Framebuffer)(nil)
