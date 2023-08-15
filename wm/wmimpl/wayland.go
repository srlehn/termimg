//go:build unix && !noWayland && !android && !darwin && !js

// based on xgbutil examples

package wmimpl

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/wm"
)

func createWindowWayland(env environ.Proprietor, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	return nil // TODO implement wayland window
}
