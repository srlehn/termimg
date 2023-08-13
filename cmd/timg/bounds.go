package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"

	errorsGo "github.com/go-errors/errors"

	"github.com/srlehn/termimg/term"
)

func splitDimArg(dim string, surv term.Surveyor, imgFile string) (x, y, w, h int, e error) {
	dimParts := strings.Split(dim, `,`)
	if len(dimParts) > 4 {
		return 0, 0, 0, 0, errorsGo.New(`image position string not "<x>,<y>,<w>x<h>"`)
	}
	var err error
	var xu, yu, wu, hu uint64
	for i, dimPart := range dimParts {
		if strings.Contains(dimPart, `x`) {
			if i != len(dimParts)-1 {
				return 0, 0, 0, 0, errorsGo.New(errShowUsage)
			}
			sizes := strings.SplitN(dimPart, `x`, 2)
			if len(sizes[0]) > 0 {
				wu, err = strconv.ParseUint(sizes[0], 10, 64)
				if err != nil {
					return 0, 0, 0, 0, errorsGo.New(errShowUsage)
				}
			}
			if len(sizes[1]) > 0 {
				hu, err = strconv.ParseUint(sizes[1], 10, 64)
				if err != nil {
					return 0, 0, 0, 0, errorsGo.New(errShowUsage)
				}
			}
			break
		}
		var val uint64
		// default to 0
		if len(dimPart) > 0 {
			val, err = strconv.ParseUint(dimPart, 10, 64)
			if err != nil {
				return 0, 0, 0, 0, errorsGo.New(errShowUsage)
			}
		}
		switch i {
		case 0:
			xu = val
		case 1:
			yu = val
		case 2:
			wu = val
		case 3:
			hu = val
		}
	}
	var wScaled, hScaled uint
	if wu == 0 || hu == 0 {
		// return 0, 0, 0, 0, errorsGo.New(`rectangle side with length 0`)
		tcw, tch, err := surv.SizeInCells()
		if err != nil {
			return 0, 0, 0, 0, errorsGo.New(err)
		}
		cpw, cph, err := surv.CellSize()
		if err != nil {
			return 0, 0, 0, 0, errorsGo.New(err)
		}
		f, err := os.Open(imgFile) // TODO don't open image multiple times
		if err != nil {
			return 0, 0, 0, 0, errorsGo.New(errShowUsage)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			return 0, 0, 0, 0, errorsGo.New(err)
		}
		_ = f.Close()
		imgBounds := img.Bounds()
		ar := float64(imgBounds.Dx()) / float64(imgBounds.Dy())
		arc := ar * float64(cph) / float64(cpw)
		var wAvail, hAvail uint // TODO subtract prompt height
		if uint(xu) < tcw {
			wAvail = tcw - uint(xu)
		}
		if uint(yu) < tch {
			hAvail = tch - uint(yu)
		}
		if wu > 0 {
			hScaled = uint(float64(wu) / arc)
		}
		if hu > 0 {
			wScaled = uint(float64(hu) * arc)
		}
		if wu == 0 && hu == 0 && wAvail > 0 && hAvail > 0 {
			areaRatio := float64(wAvail) / float64(hAvail)
			if areaRatio > arc {
				wScaled = uint((float64(wAvail) * arc) / areaRatio)
				hScaled = hAvail
			} else {
				wScaled = wAvail
				hScaled = uint((float64(hAvail) * areaRatio) / arc)
			}
		}
	}
	x, y = int(xu), int(yu)
	if wScaled > 0 {
		w = int(wScaled)
	} else {
		w = int(wu)
	}
	if hScaled > 0 {
		h = int(hScaled)
	} else {
		h = int(hu)
	}
	if w < 1 && h < 1 {
		return 0, 0, 0, 0, errorsGo.New(`image position outside visible area`)
	}
	return x, y, w, h, nil
}
