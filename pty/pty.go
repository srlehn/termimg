package pty

import (
	"image"

	"github.com/go-errors/errors"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/term"
)

type PTYRunFunc = func(pty string, pidTerm uint) error

// PTYRun starts a pseudo terminal and runs a function on its pty.
func PTYRun(f PTYRunFunc, termCmd ...string) error {
	if f == nil {
		return errors.New(internal.ErrNilParam)
	}
	// TODO use default terminal: xterm, conhost, Terminal.app
	if len(termCmd) == 0 {
		return errors.New(`no command`)
	}
	return ptyRun(termCmd, f)
}

type DrawFuncProvider = func(img image.Image, cellBounds image.Rectangle) DrawFunc

// DrawFunc returns the area that will be screenshot by TakeScreenshotFunc and the scale for unstretching a possibly contained image
type DrawFunc = func(tm *term.Terminal, dr term.Drawer, rsz term.Resizer, cpw, cph uint) (areaOfInterest image.Rectangle, scaleX, scaleY float64, e error)

type TermProviderFunc = func(ttyFile string) (*term.Terminal, error)
