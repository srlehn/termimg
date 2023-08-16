//go:build !linux || android

package wmimpl

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/wminternal"
	"github.com/srlehn/termimg/wm"
)

func createWindowConsole(env environ.Proprietor, name, class, instance string, isWindow wm.IsWindowFunc) wm.Window {
	return &wminternal.WindowDummy{}
}
