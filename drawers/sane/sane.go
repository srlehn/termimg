package sane

import (
	_ "github.com/srlehn/termimg/drawers/generic"

	_ "github.com/srlehn/termimg/drawers/domterm"
	_ "github.com/srlehn/termimg/drawers/iterm2"
	_ "github.com/srlehn/termimg/drawers/kitty"
	_ "github.com/srlehn/termimg/drawers/sixel"
	_ "github.com/srlehn/termimg/drawers/terminology"
)

// don't include Braille drawer by default!
