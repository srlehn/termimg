package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// MacTerm
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerMacTerm{term.NewTermCheckerCore(termNameMacTerm)})
}

const termNameMacTerm = `macterm`

var _ term.TermChecker = (*termCheckerMacTerm)(nil)

type termCheckerMacTerm struct{ term.TermChecker }

func (t *termCheckerMacTerm) CheckExclude(pr term.Properties) (mightBe bool, p term.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMacTerm, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`TERM_PROGRAM`)
	if ok && v == `MacTerm` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMacTerm, consts.CheckTermPassed)
		if ver, okV := pr.LookupEnv(`TERM_PROGRAM_VERSION`); okV && len(ver) > 0 {
			p.SetProperty(propkeys.MacTermBuildNr, ver) // YYYYMMDD
			return true, p
		}
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMacTerm, consts.CheckTermFailed)
	return false, p
}
