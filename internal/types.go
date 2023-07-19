package internal

import (
	"image"
	"io"
)

// Arger is for passing extra arguments to the default terminal execution call
// e.g. disabling daemonizing (domterm)
type Arger interface{ Args() []string }

type ImageEncoder interface {
	Encode(w io.Writer, img image.Image, fileExt string) error
}
