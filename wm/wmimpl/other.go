//go:build noX11 || android || darwin || js

// not supported platforms

package wmimpl

import (
	"image"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/wm"
)

var _ wm.Connection = (*connOthers)(nil)

// connOthers ...
type connOthers struct{}

// NewConn ...
func newConn(env environ.Properties) (*connOthers, error) {
	return nil, errors.New(consts.ErrPlatformNotSupported)
}

func (c *connOthers) Close() error { return nil }

func (c *connOthers) Conn() any { return nil }

func (c *connOthers) Windows() ([]wm.Window, error) {
	return nil, errors.New(consts.ErrPlatformNotSupported)
}

// DisplayImage ...
func (c *connOthers) DisplayImage(img image.Image, windowName string) {}
func (c *connOthers) Resources() (environ.Properties, error) {
	return nil, errors.New(consts.ErrNotImplemented)
}

// windowOther ...
type windowOther struct {
	wminternal.WindowDummy
	is      func(w wm.Window) (is bool, p environ.Properties)
	isInit  bool
	errFind error
}

var _ wm.Window = (*windowOther)(nil)

func createWindow(env environ.Properties, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	return &windowOther{}
}

// func (w *Window) WindowType() string { return `windows` }

func (w *windowOther) WindowName() string        { return `` }
func (w *windowOther) WindowClass() string       { return `` }
func (w *windowOther) WindowInstance() string    { return `` }
func (w *windowOther) WindowID() uint64          { return 0 }
func (w *windowOther) WindowPID() uint64         { return 0 }
func (w *windowOther) WindowConn() wm.Connection { return nil }
