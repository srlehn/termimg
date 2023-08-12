//go:build linux && !android

package sane

import (
	_ "github.com/srlehn/termimg/drawers/framebuffer"
)
