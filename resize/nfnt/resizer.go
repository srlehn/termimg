package nfnt

import (
	"image"

	"github.com/nfnt/resize"

	"github.com/srlehn/termimg/term"
)

// Resizer uses "github.com/nfnt/resize"
type Resizer struct{}

var _ term.Resizer = (*Resizer)(nil)

// Resize ...
func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	m := resize.Resize(uint(size.X), uint(size.Y), img, resize.Lanczos3)
	return m, nil
}
