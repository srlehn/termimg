package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Yaft
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerYaft{term.NewTermCheckerCore(termNameYaft)})
}

const termNameYaft = `yaft`

var _ term.TermChecker = (*termCheckerYaft)(nil)

type termCheckerYaft struct{ term.TermChecker }

func (t *termCheckerYaft) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameYaft, term.CheckTermFailed)
		return false, p
	}
	v, ok := ci.LookupEnv(`TERM`)
	if ok && v == "yaft-256color" {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameYaft, term.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameYaft, term.CheckTermFailed)
	return false, p
}
