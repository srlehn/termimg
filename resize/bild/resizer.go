package bild

import (
	"image"

	"github.com/anthonynsimon/bild/transform"

	"github.com/srlehn/termimg/term"
)

// Resizer uses "github.com/anthonynsimon/bild/transform"
type Resizer struct{}

var _ term.Resizer = (*Resizer)(nil)

// Resize ...
func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	m := transform.Resize(img, size.X, size.Y, transform.Lanczos)
	return m, nil
}
