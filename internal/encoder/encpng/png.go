package encpng

import (
	"image"
	"image/png"
	"io"
	"strings"

	"github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
)

var _ internal.ImageEncoder = (*PngEncoder)(nil)

type PngEncoder struct{}

func (e *PngEncoder) Encode(w io.Writer, img image.Image, fileExt string) error {
	if w == nil || img == nil {
		return errors.New(internal.ErrNilParam)
	}
	// allow passing whole filename
	fileExtParts := strings.Split(fileExt, `.`)
	fileExt = fileExtParts[len(fileExtParts)-1]
	fmtStr := strings.ToLower(strings.TrimPrefix(fileExt, `.`))
	if fmtStr != `png` {
		return errors.New(`unsupported file format`)
	}
	if err := png.Encode(w, img); err != nil {
		return errors.New(err)
	}
	return nil
}
