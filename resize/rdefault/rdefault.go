package rdefault

import (
	"image"
	"runtime"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/resize/imaging"
	"github.com/srlehn/termimg/resize/rez"
	"github.com/srlehn/termimg/term"
)

type Resizer struct {
	rszRez rez.Resizer
	rszImg imaging.Resizer
}

var _ term.Resizer = (*Resizer)(nil)

func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	if runtime.GOARCH != `amd64` {
		return r.rszImg.Resize(img, size)
	}
	im := img
	var lvls int
	maxLvl := 5
repeat:
	switch it := im.(type) {
	case *image.YCbCr, *image.RGBA, *image.NRGBA, *image.Gray:
		// use SIMD assembly if possible
		imgRet, err := r.rszRez.Resize(im, size)
		if err != nil {
			imgRet, err = r.rszImg.Resize(img, size)
		}
		if err != nil {
			return nil, err
		}
		return imgRet, nil
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
	return r.rszImg.Resize(img, size)
}
