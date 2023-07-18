package termimg

import (
	"bytes"
	"image"

	// _ "image/jpeg" // ...
	// _ "image/png"  // ...

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

var (
	// chosen defaults
	ttyProvider                   = gotty.TTYProv
	windowProvider                = wm.NewWindow
	querier                       = qdefault.NewQuerier()
	wmImplementation              = wmimpl.Impl()
	resizer          term.Resizer = &rdefault.Resizer{}
)

var (
	termActive *term.Terminal
)

// Terminal ...
func Terminal() (*term.Terminal, error) {
	return termActive, initTerm()
}

func initTerm() error {
	if termActive != nil {
		return nil
	}
	ptyName := internal.DefaultTTYDevice()
	wm.SetImpl(wmImplementation)
	var err error
	cr := &term.Creator{
		PTYName:         ptyName,
		TTYProvFallback: ttyProvider,
		Querier:         querier,
		WindowProvider:  windowProvider,
		Resizer:         resizer,
	}
	termActive, err = term.NewTerminal(cr)
	if err != nil {
		return err
	}
	return nil
}

// NewTerminal ...
func NewTerminal(ptyName string) (*term.Terminal, error) {
	wm.SetImpl(wmImplementation)
	cr := &term.Creator{
		PTYName:         ptyName,
		TTYProvFallback: ttyProvider,
		Querier:         querier,
		WindowProvider:  windowProvider,
		Resizer:         resizer,
	}
	tm, err := term.NewTerminal(cr)
	if err != nil {
		return nil, err
	}
	return tm, nil
}

// Query ...
func Query(qs string, p term.Parser) (string, error) {
	tm, err := Terminal()
	if err != nil {
		return ``, err
	}
	return tm.Query(qs, p)
}

// Draw ...
func Draw(img image.Image, bounds image.Rectangle) error {
	tm, err := Terminal()
	if err != nil {
		return err
	}
	return term.Draw(img, bounds, resizer, tm)
}

// DrawBytes - for use with "embed", etc.
// requires the prior registration of a decoder. e.g.:
//
//	import _ "image/png"
func DrawBytes(imgBytes []byte, bounds image.Rectangle) error {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return err
	}
	tm, err := Terminal()
	if err != nil {
		return err
	}
	return term.Draw(img, bounds, resizer, tm)
}

// DrawFile ...
func DrawFile(imgFile string, bounds image.Rectangle) error {
	if err := initTerm(); err != nil {
		return err
	}
	return termActive.Draw(term.NewImageFileName(imgFile), bounds)
}

// CleanUp ...
func CleanUp() error {
	if termActive == nil {
		return nil
	}
	return termActive.Close()
}

// NewImage ...
func NewImage(img image.Image) *term.Image { return term.NewImage(img) }

// NewImageFileName ...
func NewImageFileName(imgfile string) *term.Image { return term.NewImageFileName(imgfile) }

// NewImageBytes ...
func NewImageBytes(imgBytes []byte) *term.Image { return term.NewImageBytes(imgBytes) }
