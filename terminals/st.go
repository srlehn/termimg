package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

// //////////////////////////////////////////////////////////////////////////////
// st
// //////////////////////////////////////////////////////////////////////////////
func init() {
	term.RegisterTermChecker(&termCheckerSt{term.NewTermCheckerCore(termNameSt)})
}

const termNameSt = `st`

var _ term.TermChecker = (*termCheckerSt)(nil)

type termCheckerSt struct{ term.TermChecker }

func (t *termCheckerSt) CheckExclude(pr term.Properties) (mightBe bool, p term.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameSt, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`TERM`)
	if ok && v == "st-256color" {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameSt, consts.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameSt, consts.CheckTermFailed)
	return false, p
}

/*
sixel capable forks:
https://gist.github.com/saitoha/70e0fdf22e3e8f63ce937c7f7da71809
https://github.com/charlesdaniels/st buggy sixel display
https://github.com/galatolofederico/st-sixel less buggy sixel display, buggy font, does not report sixel capability
*/
