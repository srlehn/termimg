// Seam Carving for Content-Aware Image Resizing
package caire

import (
	"image"
	"image/draw"

	"github.com/esimov/caire"
	"github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/term"
)

type Resizer struct{}

var _ term.Resizer = (*Resizer)(nil)

func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	p := &caire.Processor{
		BlurRadius:     1, // or ie. 4
		SobelThreshold: 4, // or ie. 2
		NewWidth:       size.X,
		NewHeight:      size.Y,
		FaceDetect:     true,
		ShapeType:      "circle",
	}
	var nimg *image.NRGBA
	im := img
	var lvls int
	maxLvl := 5
repeat:
	switch it := im.(type) {
	case *image.NRGBA:
		nimg = it
		goto end
	case *term.Image:
		if it.Original == nil {
			if lvls > maxLvl {
				break
			}
			lvls++
			if err := it.Decode(); err != nil {
				return nil, err
			}
			if it.Original == nil {
				return nil, errors.New(internal.ErrNilImage)
			}
			im = it
			goto repeat
		}
	}
	if nimg == nil {
		b := img.Bounds()
		nimg = image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(nimg, nimg.Bounds(), img, b.Min, draw.Src)
	}
end:
	return p.Resize(nimg)
}
