package encpng

import (
	"image"
	"image/png"
	"io"
	"strings"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
)

var _ internal.ImageEncoder = (*PngEncoder)(nil)

type PngEncoder struct{}

func (e *PngEncoder) Encode(w io.Writer, img image.Image, fileExt string) error {
	if w == nil || img == nil {
		return errors.New(consts.ErrNilParam)
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
