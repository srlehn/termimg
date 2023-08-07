//go:build !windows && !android && !darwin && !js

package all

import (
	_ "github.com/srlehn/termimg/drawers/w3mimgdisplay"
	_ "github.com/srlehn/termimg/drawers/x11"
)
