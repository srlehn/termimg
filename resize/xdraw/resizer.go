// Package xdraw provides a resizer implementation using golang.org/x/image/draw.
// ApproxBiLinear is recommended for balanced speed/quality scaling.
package xdraw

import (
	"image"

	"golang.org/x/image/draw"

	"github.com/srlehn/termimg/term"
)

// resizer uses "golang.org/x/image/draw"
type resizer struct {
	scaler draw.Scaler
}

var _ term.Resizer = (*resizer)(nil)

// ApproxBiLinear creates a new resizer with ApproxBiLinear scaling (balanced speed/quality).
func ApproxBiLinear() term.Resizer {
	return &resizer{scaler: draw.ApproxBiLinear}
}

// BiLinear creates a new resizer with BiLinear scaling (higher quality, slower).
func BiLinear() term.Resizer {
	return &resizer{scaler: draw.BiLinear}
}

// CatmullRom creates a new resizer with CatmullRom scaling (highest quality, slowest).
func CatmullRom() term.Resizer {
	return &resizer{scaler: draw.CatmullRom}
}

// Resize scales an image to the target size using the configured scaler.
func (r *resizer) Resize(img image.Image, size image.Point) (image.Image, error) {
	dst := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	r.scaler.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst, nil
}