//go:build cgo && caire

package main

import (
	"github.com/srlehn/termimg/resize/caire"
	"github.com/srlehn/termimg/term"
)

func init() {
	showCmd.PersistentFlags().BoolVarP(&showCaire, `caire`, `c`, false, `enable content aware image resizing`)
	showResizerCaire = func() term.Resizer {
		blurRadius := 4
		sobelThreshold := 2
		faceDetect := true
		shapeType := `circle`
		return caire.NewResizer(blurRadius, sobelThreshold, faceDetect, shapeType)
	}
}
