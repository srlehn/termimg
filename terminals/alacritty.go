package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
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

func (t *termCheckerAlacritty) CheckExclude(pr term.Properties) (mightBe bool, p term.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameAlacritty, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`ALACRITTY_LOG`)
	if ok && len(v) > 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameAlacritty, consts.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameAlacritty, consts.CheckTermFailed)
	return false, p
}
func (t *termCheckerAlacritty) CheckIsWindow(w wm.Window) (is bool, p term.Properties) {
	p = environ.NewProperties()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameAlacritty, consts.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `Alacritty` &&
		w.WindowClass() == `Alacritty` &&
		w.WindowInstance() == `Alacritty`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameAlacritty, consts.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameAlacritty, consts.CheckTermFailed)
	}
	return isWindow, p
}

// https://github.com/alacritty/alacritty/blob/master/docs/escape_support.md
