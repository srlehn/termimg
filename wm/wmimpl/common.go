// actual implementation (for X11, Windows)
package wmimpl

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/wm"
)

var _ wm.Implementation = (*implementation)(nil)

type implementation struct{}

func Impl() wm.Implementation { return &implementation{} }

func (i *implementation) Name() string { return `generic` }

func (i *implementation) Conn(env environ.Properties) (wm.Connection, error) {
	return newConn(env)
}

func (i *implementation) CreateWindow(env environ.Properties, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	return createWindow(env, name, class, instance, isWindow)
}
