package rdefault

import (
	"image"
	"runtime"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/resize/xdraw"
	"github.com/srlehn/termimg/resize/rez"
	"github.com/srlehn/termimg/term"
)

type Resizer struct{}

var _ term.Resizer = (*Resizer)(nil)

func (r *Resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	if runtime.GOARCH != `amd64` {
		return xdraw.ApproxBiLinear().Resize(img, size)
	}
	im := img
	var lvls int
	maxLvl := 5
repeat:
	switch it := im.(type) {
	case *image.YCbCr, *image.RGBA, *image.NRGBA, *image.Gray:
		// use SIMD assembly if possible
		imgRet, err := rez.Resizer{}.Resize(im, size)
		if err != nil {
			imgRet, err = xdraw.ApproxBiLinear().Resize(img, size)
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
	return xdraw.ApproxBiLinear().Resize(img, size)
}
