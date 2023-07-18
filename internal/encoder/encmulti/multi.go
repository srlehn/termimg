package encmulti

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"

	"github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
)

var _ internal.ImageEncoder = (*MultiEncoder)(nil)

type MultiEncoder struct{}

func (e *MultiEncoder) Encode(w io.Writer, img image.Image, fileExt string) error {
	if img == nil {
		return errors.New(internal.ErrNilParam)
	}
	// allow passing whole filename
	fileExtParts := strings.Split(fileExt, `.`)
	fileExt = fileExtParts[len(fileExtParts)-1]
	fmtStr := strings.ToLower(strings.TrimPrefix(fileExt, `.`))

	if len(fmtStr) == 0 {
		return errors.New(`no file format specified`)
	}
	var err error
	switch fmtStr {
	case `bmp`:
		err = bmp.Encode(w, img)
	case `gif`:
		err = gif.Encode(w, img, nil)
	case `png`:
		err = png.Encode(w, img)
	case `tiff`:
		err = tiff.Encode(w, img, &tiff.Options{Compression: tiff.LZW, Predictor: true})
	case `jpg`, `jpeg`:
		err = jpeg.Encode(w, img, &jpeg.Options{Quality: 90})
	default:
		err = errors.New(`unsupported file format: "` + fmtStr + `"`)
	}
	if err != nil {
		return errors.New(err)
	}
	return nil
}
