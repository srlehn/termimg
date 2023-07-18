//go:build windows || android || darwin || js

package pty

import (
	"image"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"

	"github.com/go-errors/errors"
)

func ptyRun(termCmd []string, f PTYRunFunc) (errRet error) {
	return errors.New(internal.ErrNotImplemented)
}

// DrawFuncOnlyPicture ...
func DrawFuncOnlyPicture(img image.Image, cellBounds image.Rectangle) DrawFunc {
	return func(tm *term.Terminal, dr term.Drawer, rsz term.Resizer, cpw, cph uint) (areaOfInterest image.Rectangle, scaleX, scaleY float64, e error) {
		return image.Rectangle{}, 0, 0, errors.New(internal.ErrNotImplemented)
	}
}

// TakeScreenshot ...
func TakeScreenshot(termName string, termProvider TermProviderFunc, drawerName string, drawFuncProvider DrawFuncProvider, imgBytes []byte, cellBounds image.Rectangle, rsz term.Resizer) (image.Image, error) {
	return nil, errors.New(internal.ErrNotImplemented)
}

// TakeScreenshotFunc displays an image in a pseudo-terminal and captures the displayed version for comparison.
func TakeScreenshotFunc(termProvider TermProviderFunc, termKind wm.Window, drawerName string, drawFunc DrawFunc, rsz term.Resizer, imgRetChan chan<- image.Image) PTYRunFunc {
	return func(pty string, pid uint) error {
		return errors.New(internal.ErrNotImplemented)
	}
}
