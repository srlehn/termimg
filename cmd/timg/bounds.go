package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"

	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/prompt"
	"github.com/srlehn/termimg/term"
)

type bounder interface {
	Bounds() image.Rectangle
}

func splitDimArg(dim string, surv term.Surveyor, env []string, img bounder) (x, y, w, h int, autoX, autoY bool, e error) {
	dimParts := strings.Split(dim, `,`)
	// var logger *slog.Logger
	var loggerProv logx.LoggerProvider
	if surv != nil {
		if lp, ok := surv.(logx.LoggerProvider); ok {
			loggerProv = lp
		}
	}
	if len(dimParts) > 4 {
		return 0, 0, 0, 0, false, false, logx.Err(`image position string not "<x>,<y>,<w>x<h>"`, loggerProv, slog.LevelError)
	}
	var err error
	var xu, yu, wu, hu uint64
	for i, dimPart := range dimParts {
		if strings.Contains(dimPart, `x`) {
			if i != len(dimParts)-1 {
				return 0, 0, 0, 0, false, false, logx.Err(showUsageStr, loggerProv, slog.LevelError)
			}
			sizes := strings.SplitN(dimPart, `x`, 2)
			if len(sizes[0]) > 0 {
				wu, err = strconv.ParseUint(sizes[0], 10, 64)
				if err != nil {
					return 0, 0, 0, 0, false, false, logx.Err(showUsageStr, loggerProv, slog.LevelError)
				}
			}
			if len(sizes[1]) > 0 {
				hu, err = strconv.ParseUint(sizes[1], 10, 64)
				if err != nil {
					return 0, 0, 0, 0, false, false, logx.Err(showUsageStr, loggerProv, slog.LevelError)
				}
			}
			break
		}
		var val uint64
		// default to 0
		if len(dimPart) > 0 {
			val, err = strconv.ParseUint(dimPart, 10, 64)
			if err != nil {
				return 0, 0, 0, 0, false, false, logx.Err(showUsageStr, loggerProv, slog.LevelError)
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
		tcw, tch, err := surv.SizeInCells()
		if err != nil {
			return 0, 0, 0, 0, false, false, logx.Err(err, loggerProv, slog.LevelError)
		}
		if img == nil {
			// scale unknown, stretch
			return x, y, int(tcw) - x, int(tch) - y, true, true, nil
		}
		imgBounds := img.Bounds()
		// return 0, 0, 0, 0, errors.New(`rectangle side with length 0`)
		cpw, cph, err := surv.CellSize()
		if err != nil {
			return 0, 0, 0, 0, false, false, logx.Err(err, loggerProv, slog.LevelError)
		}
		if cpw == 0 || cph == 0 {
			return 0, 0, 0, 0, false, false, logx.Err(`unable to query terminal size in cells`, loggerProv, slog.LevelError)
		}
		_, ph, err := prompt.GetPromptSize(env, tcw)
		if logx.IsErr(err, loggerProv, slog.LevelInfo) {
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
			if !logx.IsErr(err, loggerProv, slog.LevelInfo) &&
				(sizeScaled.X > 0 && sizeScaled.Y > 0) &&
				(sizeScaled.X < int(wScaled) || sizeScaled.Y < int(hScaled)) {
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
		return 0, 0, 0, 0, false, false, logx.Err(`image position outside visible area`, loggerProv, slog.LevelError)
	}
	return x, y, w, h, autoX, autoY, nil
}
