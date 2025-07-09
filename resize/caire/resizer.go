// Seam Carving for Content-Aware Image Resizing
//
// very slow resizer
//
// package requires cgo
package caire

import (
	"image"
	"image/draw"

	"github.com/esimov/caire"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type resizer struct {
	proc caire.Processor
}

var _ term.Resizer = (*resizer)(nil)

// shapeType can be "circle" or "line"
func NewResizer(blurRadius, sobelThreshold int, faceDetect bool, shapeType caire.ShapeType) term.Resizer {
	if blurRadius == 0 {
		// BlurRadius: 4 - github.com/esimov/caire/cmd/caire/main.go
		// BlurRadius: 1
		blurRadius = 4
	}
	if sobelThreshold == 0 {
		// SobelThreshold: 2 - github.com/esimov/caire/cmd/caire/main.go
		// SobelThreshold: 4
		sobelThreshold = 2
	}
	if len(shapeType) == 0 {
		// ShapeType: "circle" - github.com/esimov/caire/cmd/caire/main.go
		shapeType = `circle`
	}
	return &resizer{
		proc: caire.Processor{
			BlurRadius:     blurRadius,
			SobelThreshold: sobelThreshold,
			FaceDetect:     faceDetect,
			ShapeType:      shapeType,
		},
	}
}

func (r *resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	if r == nil {
		return nil, errors.NilReceiver()
	}
	r.proc.NewWidth = size.X
	r.proc.NewHeight = size.Y
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
				return nil, errors.New(consts.ErrNilImage)
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
	return r.proc.Resize(nimg)
}
