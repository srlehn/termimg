//go:build linux && !android

package all

import (
	_ "github.com/srlehn/termimg/drawers/framebuffer"
)
