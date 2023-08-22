package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
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

func (t *termCheckerApple) CheckExclude(pr environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameApple, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`TERM_PROGRAM`)
	if !ok || v != `Apple_Terminal` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameApple, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameApple, consts.CheckTermPassed)
	if ver, okV := pr.LookupEnv(`TERM_PROGRAM_VERSION`); okV && len(ver) > 0 {
		p.SetProperty(propkeys.AppleTermVersion, ver) // CFBundleVersion of Terminal.app
	}
	return true, p
}

// https://github.com/kmgrant/macterm/issues/3#issuecomment-458387953
