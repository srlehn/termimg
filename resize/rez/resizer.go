package rez

import (
	"image"

	"github.com/bamiaux/rez"

	"github.com/srlehn/termimg/term"
)

// Resizer uses "github.com/bamiaux/rez"
type Resizer struct {
	// converter rez.Converter
}

var _ term.Resizer = (*Resizer)(nil)

// Resize ...
func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	m := image.NewNRGBA(image.Rectangle{Max: image.Point{X: size.X, Y: size.Y}})
	err := rez.Convert(m, img, rez.NewBilinearFilter()) // rez.NewLanczosFilter(???) // TODO
	if err != nil {
		return nil, err
	}
	return m, nil
}
