package term

import (
	"image"

	"github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
)

// TODO func for setting priority list

// AllDrawers returns all registered drawers
func AllDrawers() []Drawer {
	return drawersRegistered
}

// Drawer ...
type Drawer interface {
	Name() string
	New() Drawer
	IsApplicable(DrawerCheckerInput) bool
	Draw(img image.Image, bounds image.Rectangle, rsz Resizer, term *Terminal) error // TODO
	// Draw(img image.Image, bounds image.Rectangle, term Terminal) error
}

////////////////////////////////////////////////////////////////////////////////

// Draw ...
func Draw(img image.Image, bounds image.Rectangle, rsz Resizer, term *Terminal) error {
	if img == nil || term == nil {
		return errors.New(internal.ErrNilParam)
	}
	drawers := term.Drawers()
	if len(drawers) == 0 {
		return errors.New(`0 drawers`)
	}
	dr := drawers[0]
	if dr == nil {
		return errors.New(`nil drawer`)
	}
	return DrawWith(img, bounds, dr, rsz, term)
}

// DrawWith ...
func DrawWith(img image.Image, bounds image.Rectangle, dr Drawer, rsz Resizer, term *Terminal) error {
	return drawWith(img, bounds, dr, rsz, term)
}

func drawWith(img image.Image, bounds image.Rectangle, dr Drawer, rsz Resizer, term *Terminal) (err error) {
	if img == nil || dr == nil || term == nil {
		return errors.New(internal.ErrNilParam)
	}
	if rsz == nil {
		rsz = &resizerFallback{}
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(r)
		}
	}()
	// TODO check if redraw is necessary

	imgTerm := NewImage(img)
	// TODO leave the decoding to the drawers
	if err := imgTerm.Decode(); err != nil {
		return err
	}

	return dr.Draw(imgTerm, bounds, rsz, term)
}
