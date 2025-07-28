package imaging

import (
	"image"

	"github.com/kovidgoyal/imaging"

	"github.com/srlehn/termimg/term"
)

// Resizer uses "github.com/kovidgoyal/imaging"
type Resizer struct{}

var _ term.Resizer = (*Resizer)(nil)

// Resize ...
func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	return imaging.Resize(img, size.X, size.Y, imaging.Lanczos), nil
}
