package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// mlterm
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerMlterm{term.NewTermCheckerCore(termNameMlterm)})
}

const termNameMlterm = `mlterm`

var _ term.TermChecker = (*termCheckerMlterm)(nil)

type termCheckerMlterm struct{ term.TermChecker }

func (t *termCheckerMlterm) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, term.CheckTermFailed)
		return false, p
	}
	envM, okM := ci.LookupEnv(`MLTERM`)
	if !okM || len(envM) == 0 {
		envT, _ := ci.LookupEnv(`TERM`)
		mayBeMlterm := envT == `mlterm`
		if mayBeMlterm {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, term.CheckTermPassed)
		} else {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, term.CheckTermFailed)
		}
		return mayBeMlterm, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, term.CheckTermPassed)
	p.SetProperty(propkeys.MltermVersion, envM)
	return true, p
}
func (t *termCheckerMlterm) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, term.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `mlterm` &&
		w.WindowClass() == `mlterm` &&
		w.WindowInstance() == `xterm`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, term.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, term.CheckTermFailed)
	}
	return isWindow, p
}
func (t *termCheckerMlterm) Args(ci environ.Proprietor) []string {
	args := []string{
		`--sbmod=none`, // disable scrollbar
	}
	return args
}
