package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// Alacritty
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerAlacritty{term.NewTermCheckerCore(termNameAlacritty)})
}

const termNameAlacritty = `alacritty`

var _ term.TermChecker = (*termCheckerAlacritty)(nil)

type termCheckerAlacritty struct{ term.TermChecker }

func (t *termCheckerAlacritty) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameAlacritty, term.CheckTermFailed)
		return false, p
	}
	v, ok := ci.LookupEnv(`ALACRITTY_LOG`)
	if ok && len(v) > 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameAlacritty, term.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameAlacritty, term.CheckTermFailed)
	return false, p
}
func (t *termCheckerAlacritty) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameAlacritty, term.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `Alacritty` &&
		w.WindowClass() == `Alacritty` &&
		w.WindowInstance() == `Alacritty`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameAlacritty, term.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameAlacritty, term.CheckTermFailed)
	}
	return isWindow, p
}

// https://github.com/alacritty/alacritty/blob/master/docs/escape_support.md
