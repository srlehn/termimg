package gift

import (
	"image"

	"github.com/disintegration/gift"

	"github.com/srlehn/termimg/term"
)

// Resizer uses "github.com/disintegration/gift"
type Resizer struct{}

var _ term.Resizer = (*Resizer)(nil)

// Resize ...
func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	m := image.NewNRGBA(image.Rectangle{Max: image.Point{X: size.X, Y: size.Y}})
	gift.Resize(size.X, size.Y, gift.LanczosResampling).Draw(m, img, &gift.Options{Parallelization: true})
	return m, nil
}
