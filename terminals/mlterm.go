package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
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

func (t *termCheckerMlterm) CheckExclude(pr environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, consts.CheckTermFailed)
		return false, p
	}
	envM, okM := pr.LookupEnv(`MLTERM`)
	if !okM || len(envM) == 0 {
		envT, _ := pr.LookupEnv(`TERM`)
		mayBeMlterm := envT == `mlterm`
		if mayBeMlterm {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, consts.CheckTermPassed)
		} else {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, consts.CheckTermFailed)
		}
		return mayBeMlterm, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMlterm, consts.CheckTermPassed)
	p.SetProperty(propkeys.MltermVersion, envM)
	return true, p
}
func (t *termCheckerMlterm) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, consts.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `mlterm` &&
		w.WindowClass() == `mlterm` &&
		w.WindowInstance() == `xterm`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, consts.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, consts.CheckTermFailed)
	}
	return isWindow, p
}
func (t *termCheckerMlterm) Args(pr environ.Proprietor) []string {
	args := []string{
		`--sbmod=none`, // disable scrollbar
	}
	return args
}
