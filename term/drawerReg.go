package term

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
)

// DrawersFor ...
func DrawersFor(inp DrawerCheckerInput) ([]Drawer, error) {
	drs, _, err := drawersFor(inp)
	return drs, err
}

func drawersFor(inp DrawerCheckerInput) ([]Drawer, Properties, error) {
	if inp == nil {
		return nil, nil, errors.NilParam()
	}
	prs := environ.NewProperties()
	var applDrawers []Drawer
	for _, dr := range drawersRegistered {
		if dr == nil {
			continue
		}
		isApplicable, pr := dr.IsApplicable(inp)
		if !isApplicable {
			continue
		}
		applDrawers = append(applDrawers, dr)
		prs.MergeProperties(pr)
	}
	if len(applDrawers) == 0 {
		prs = nil
	}
	return applDrawers, prs, nil
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
