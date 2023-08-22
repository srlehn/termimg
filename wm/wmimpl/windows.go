//go:build windows

package wmimpl

import (
	"image"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/wm"
)

var _ wm.Connection = (*connWindows)(nil)

// connWindows ...
type connWindows struct{}

func newConn(_ environ.Proprietor) (*connWindows, error) { return &connWindows{}, nil }

func (c *connWindows) Close() error { return nil }

func (c *connWindows) Conn() any { return nil }

func (c *connWindows) Windows() ([]wm.Window, error) {
	return nil, errors.New(consts.ErrNotImplemented)
}

// DisplayImage ...
func (c *connWindows) DisplayImage(img image.Image, windowName string) {}
func (c *connWindows) Resources() (environ.Proprietor, error) {
	return nil, errors.New(consts.ErrNotImplemented)
}

// windowWindows ...
type windowWindows struct {
	wminternal.WindowDummy
	is      func(w wm.Window) (is bool, p environ.Proprietor)
	isInit  bool
	errFind error
}

var _ wm.Window = (*windowWindows)(nil)

func createWindow(env environ.Proprietor, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	return &windowWindows{}
}

func (w *windowWindows) WindowType() string        { return `windows` }
func (w *windowWindows) WindowName() string        { return `` }
func (w *windowWindows) WindowClass() string       { return `` }
func (w *windowWindows) WindowInstance() string    { return `` }
func (w *windowWindows) WindowID() uint64          { return 0 }
func (w *windowWindows) WindowPID() uint64         { return 0 }
func (w *windowWindows) WindowConn() wm.Connection { return nil }
