//go:build dev

package termuiimg

import (
	"image"

	"github.com/gizak/termui/v3"
)

// debug

var prIdx int

func pr(s string, img *Image, buf *termui.Buffer) {
	if img == nil || buf == nil || len(s) == 0 {
		return
	}
	blockH := img.Inner.Dy()
	if blockH != 0 {
		prIdx = prIdx%blockH + 1
	} else {
		prIdx++
	}
	buf.SetString(s, termui.StyleClear, image.Pt(img.Min.X+1, img.Min.Y+prIdx))
}
