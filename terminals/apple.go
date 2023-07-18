package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Apple
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerApple{term.NewTermCheckerCore(termNameApple)})
}

const termNameApple = `apple`

var _ term.TermChecker = (*termCheckerApple)(nil)

type termCheckerApple struct{ term.TermChecker }

func (t *termCheckerApple) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameApple, term.CheckTermFailed)
		return false, p
	}
	v, ok := ci.LookupEnv(`TERM_PROGRAM`)
	if ok && v == `Apple_Terminal` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameApple, term.CheckTermPassed)
		if ver, okV := ci.LookupEnv(`TERM_PROGRAM_VERSION`); okV && len(ver) > 0 {
			p.SetProperty(propkeys.AppleTermVersion, ver) // CFBundleVersion of Terminal.app
			return true, p
		}
		return true, nil
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameApple, term.CheckTermFailed)
	return false, p
}

// https://github.com/kmgrant/macterm/issues/3#issuecomment-458387953
