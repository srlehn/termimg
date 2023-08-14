package termimg

import (
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
	ttyDefault                    = internal.DefaultTTYDevice()
	ttyProvider                   = gotty.New
	windowProvider                = wm.NewWindow
	querier                       = qdefault.NewQuerier()
	wmImplementation              = wmimpl.Impl()
	resizer          term.Resizer = &rdefault.Resizer{}
)

var (
	DefaultConfig = term.Options{
		term.SetPTYName(ttyDefault),
		term.SetTTYProvider(ttyProvider, false),
		term.SetQuerier(querier, true),
		term.SetWindowProvider(windowProvider, true),
		term.SetResizer(resizer),
	}
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
	wm.SetImpl(wmImplementation)
	var err error
	termActive, err = term.NewTerminal(DefaultConfig)
	if err != nil {
		return err
	}
	return nil
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
	return tm.Draw(img, bounds)
}

// DrawBytes - for use with "embed", etc.
// requires the prior registration of a decoder. e.g.:
//
//	import _ "image/png"
func DrawBytes(imgBytes []byte, bounds image.Rectangle) error {
	tm, err := Terminal()
	if err != nil {
		return err
	}
	return tm.Draw(NewImageBytes(imgBytes), bounds)
}

// DrawFile ...
func DrawFile(imgFile string, bounds image.Rectangle) error {
	tm, err := Terminal()
	if err != nil {
		return err
	}
	return tm.Draw(term.NewImageFilename(imgFile), bounds)
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
func NewImageFileName(imgfile string) *term.Image { return term.NewImageFilename(imgfile) }

// NewImageBytes - for use with "embed", etc.
// requires the prior registration of a decoder. e.g.:
//
//	import _ "image/png"
func NewImageBytes(imgBytes []byte) *term.Image { return term.NewImageBytes(imgBytes) }
