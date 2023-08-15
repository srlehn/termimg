//go:build unix && !android && !darwin && !js

package wmimpl

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/wm"
)

func createWindow(env environ.Proprietor, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	var xdgSessionType string
	if env != nil {
		xdgSessionType, _ = env.LookupEnv(`XDG_SESSION_TYPE`)
	}
	switch xdgSessionType {
	case `tty`:
		return createWindowConsole(env, name, class, instance, isWindow)
	case `x11`:
		return createWindowX11(env, name, class, instance, isWindow)
	case `wayland`:
		return createWindowWayland(env, name, class, instance, isWindow)
	case ``:
		return &wminternal.WindowDummy{}
	default:
		return &wminternal.WindowDummy{}
	}
}
