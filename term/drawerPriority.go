package term

import (
	"slices"
	"strings"
)

func init() { ResetDrawerList() }

func DisableDrawer(name string) {
	name = strings.TrimSpace(name)
	idxLocal := slices.Index(drawersPriorityOrderedLocal, name)
	if idxLocal < 0 {
		return
	}
	drawersPriorityOrderedLocal = slices.Delete(drawersPriorityOrderedLocal, idxLocal, idxLocal+1)

	idxRemote := slices.Index(drawersPriorityOrderedRemote, name)
	if idxRemote < 0 {
		return
	}
	drawersPriorityOrderedRemote = slices.Delete(drawersPriorityOrderedRemote, idxRemote, idxRemote+1)
}

func ResetDrawerList() {
	drawersPriorityOrderedLocal = drawersPriorityOrderedLocalDefault
	drawersPriorityOrderedRemote = drawersPriorityOrderedRemoteDefault
}

var (
	drawersPriorityOrderedLocal, drawersPriorityOrderedRemote []string

	drawersPriorityOrderedLocalDefault = []string{
		`terminology`,
		`iterm2`,
		`kitty`,
		`sixel`,
		`domterm`,
		`framebuffer`,
		`urxvt`,
		`conhost_gdi`,
		`x11`,
		`w3mimgdisplay`,
		`generic`,
	}

	drawersPriorityOrderedRemoteDefault = []string{
		`terminology`,
		`iterm2`,
		`kitty`,
		`sixel`,
		`domterm`,
		`urxvt`,
		`generic`,
	}
)
