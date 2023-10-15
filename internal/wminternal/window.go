package wminternal

import (
	"image"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/wm"
)

var _ wm.Window = (*WindowDummy)(nil)

type WindowCore struct{}

func (w *WindowCore) WindowConn() wm.Connection { return nil }
func (w *WindowCore) WindowFind() error         { return errors.New(`dummy implementation`) }
func (w *WindowCore) WindowType() string        { return `` }
func (w *WindowCore) WindowName() string        { return `` }
func (w *WindowCore) WindowClass() string       { return `` }
func (w *WindowCore) WindowInstance() string    { return `` }
func (w *WindowCore) WindowID() uint64          { return 0 }
func (w *WindowCore) WindowPID() uint64         { return 0 }
func (w *WindowCore) DeviceContext() uintptr    { return 0 }
func (w *WindowCore) Screenshot() (image.Image, error) {
	return nil, errors.New(`dummy implementation`)
}

type WindowDummy struct{ WindowCore }

func (w *WindowDummy) Close() error { return nil }

func NewWindowCore() *WindowCore { return &WindowCore{} }
func NewWindowDummy() wm.Window  { return &WindowDummy{} }

var _ wm.Implementation = (*dummyImplementation)(nil)

type dummyImplementation struct{}

func DummyImpl() wm.Implementation { return &dummyImplementation{} }

func (i *dummyImplementation) Name() string { return `dummy` }

func (i *dummyImplementation) Conn(env environ.Properties) (wm.Connection, error) {
	return nil, errors.New(consts.ErrNotImplemented)
}

func (i *dummyImplementation) CreateWindow(env environ.Properties, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	return NewWindowDummy()
}
