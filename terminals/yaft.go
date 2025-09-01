package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
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

func (t *termCheckerYaft) CheckExclude(pr term.Properties) (mightBe bool, p term.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameYaft, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`TERM`)
	if ok && v == "yaft-256color" {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameYaft, consts.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameYaft, consts.CheckTermFailed)
	return false, p
}
