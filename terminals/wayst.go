package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// wayst
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerWayst{term.NewTermCheckerCore(termNameWayst)})
}

const termNameWayst = `wayst`

var _ term.TermChecker = (*termCheckerWayst)(nil)

type termCheckerWayst struct{ term.TermChecker }

func (t *termCheckerWayst) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameWayst, term.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `Wayst` &&
		w.WindowClass() == `Wayst` &&
		w.WindowInstance() == `Wayst`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameWayst, term.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameWayst, term.CheckTermFailed)
	}
	return isWindow, p
}
