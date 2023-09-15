package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/prompt"
	"github.com/srlehn/termimg/term"
)

// func splitDimArg(dim string, surv term.Surveyor, imgFile string) (x, y, w, h int, e error) {
func splitDimArg(dim string, surv term.Surveyor, env []string, img image.Image) (x, y, w, h int, autoX, autoY bool, e error) {
	dimParts := strings.Split(dim, `,`)
	if len(dimParts) > 4 {
		return 0, 0, 0, 0, false, false, errors.New(`image position string not "<x>,<y>,<w>x<h>"`)
	}
	var err error
	var xu, yu, wu, hu uint64
	for i, dimPart := range dimParts {
		if strings.Contains(dimPart, `x`) {
			if i != len(dimParts)-1 {
				return 0, 0, 0, 0, false, false, errors.New(showUsageStr)
			}
			sizes := strings.SplitN(dimPart, `x`, 2)
			if len(sizes[0]) > 0 {
				wu, err = strconv.ParseUint(sizes[0], 10, 64)
				if err != nil {
					return 0, 0, 0, 0, false, false, errors.New(showUsageStr)
				}
			}
			if len(sizes[1]) > 0 {
				hu, err = strconv.ParseUint(sizes[1], 10, 64)
				if err != nil {
					return 0, 0, 0, 0, false, false, errors.New(showUsageStr)
				}
			}
			break
		}
		var val uint64
		// default to 0
		if len(dimPart) > 0 {
			val, err = strconv.ParseUint(dimPart, 10, 64)
			if err != nil {
				return 0, 0, 0, 0, false, false, errors.New(showUsageStr)
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
		if img == nil {
			return 0, 0, 0, 0, false, false, errors.New(`nil image`)
		}
		imgBounds := img.Bounds()
		// return 0, 0, 0, 0, errors.New(`rectangle side with length 0`)
		tcw, tch, err := surv.SizeInCells()
		if err != nil {
			return 0, 0, 0, 0, false, false, errors.New(err)
		}
		cpw, cph, err := surv.CellSize()
		if err != nil {
			return 0, 0, 0, 0, false, false, errors.New(err)
		}
		if cpw == 0 || cph == 0 {
			return 0, 0, 0, 0, false, false, errors.New(`unable to query terminal size in cells`)
		}
		_, ph, err := prompt.GetPromptSize(env)
		if err != nil {
			// TODO log error
			ph = 1
		}
		tch -= ph                                               // subtract shell prompt height
		ar := float64(imgBounds.Dx()) / float64(imgBounds.Dy()) // aspect ratio (pixels)
		arc := ar * float64(cph) / float64(cpw)                 // aspect ratio (cells)
		var wAvail, hAvail uint                                 // TODO subtract prompt height
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
			sizeScaled, err := surv.CellScale(img.Bounds().Size(), image.Point{})
			if err == nil &&
				(sizeScaled.X > 0 && sizeScaled.Y > 0) &&
				(sizeScaled.X < int(wScaled) || sizeScaled.Y < int(hScaled)) {
				// TODO log error
				wScaled = uint(sizeScaled.X)
				hScaled = uint(sizeScaled.Y)
			}
		}
		if wu == 0 {
			autoX = true
		}
		if hu == 0 {
			autoY = true
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
		return 0, 0, 0, 0, false, false, errors.New(`image position outside visible area`)
	}
	return x, y, w, h, autoX, autoY, nil
}
