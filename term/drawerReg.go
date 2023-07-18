package term

import (
	"github.com/go-errors/errors"
	"github.com/srlehn/termimg/internal"
)

// DrawersFor ...
func DrawersFor(inp DrawerCheckerInput) ([]Drawer, error) {
	if inp == nil {
		return nil, errors.New(internal.ErrNilParam)
	}
	var applDrawers []Drawer
	for _, dr := range drawersRegistered {
		if dr == nil || !dr.IsApplicable(inp) {
			continue
		}
		applDrawers = append(applDrawers, dr)
	}
	return applDrawers, nil
}

var drawersRegistered []Drawer

// RegisterDrawer ...
func RegisterDrawer(d Drawer) {
	drawersRegistered = append(drawersRegistered, d)
}

// EnabledDrawers returns all enabled registered drawers
func EnabledDrawers() []Drawer {
	mapDrawers := make(map[string]struct{})
	for _, name := range append(drawersPriorityOrderedLocal, drawersPriorityOrderedRemote...) {
		mapDrawers[name] = struct{}{}
	}
	drawersEnabled := make([]Drawer, 0, len(drawersPriorityOrderedLocal)+len(drawersPriorityOrderedRemote))
	for _, dr := range drawersRegistered {
		if dr == nil {
			continue
		}
		if _, ok := mapDrawers[dr.Name()]; ok {
			drawersEnabled = append(drawersEnabled, dr)
		}
	}
	return drawersEnabled
}

var mapDrawer map[string]Drawer

// GetRegDrawerByName returns registered drawers
func GetRegDrawerByName(name string) Drawer {
	if mapDrawer == nil {
		mapDrawer = make(map[string]Drawer)
		for _, dr := range drawersRegistered {
			if dr == nil {
				continue
			}
			mapDrawer[dr.Name()] = dr
		}
	}
	dr, ok := mapDrawer[name]
	if ok && dr != nil {
		return dr
	}
	return nil
}
