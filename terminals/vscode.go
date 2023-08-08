package terminals

import (
	"strings"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterTermChecker(&termCheckerVSCode{term.NewTermCheckerCore(termNameVSCode)}) }

const termNameVSCode = `vscode`

var _ term.TermChecker = (*termCheckerVSCode)(nil)

type termCheckerVSCode struct{ term.TermChecker }

func (t *termCheckerVSCode) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, term.CheckTermFailed)
		return false, p
	}

	envTP, okTP := ci.LookupEnv(`TERM_PROGRAM`)
	if !okTP || envTP != `vscode` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, term.CheckTermFailed)
		return false, p
	}
	envTPV, okTPV := ci.LookupEnv(`TERM_PROGRAM_VERSION`)
	if !okTPV || len(envTPV) == 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, term.CheckTermFailed)
		return false, p
	}

	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, term.CheckTermPassed)
	p.SetProperty(propkeys.VSCodeVersion, envTPV)
	ver := strings.Split(envTPV, `.`)
	if len(ver) == 3 {
		p.SetProperty(propkeys.VSCodeVersionMajor, ver[0])
		p.SetProperty(propkeys.VSCodeVersionMinor, ver[1])
		p.SetProperty(propkeys.VSCodeVersionPatch, ver[2])
	}

	return true, p
}
