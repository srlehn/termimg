package terminals

import (
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterTermChecker(&termCheckerVSCode{term.NewTermCheckerCore(termNameVSCode)}) }

const termNameVSCode = `vscode`

var _ term.TermChecker = (*termCheckerVSCode)(nil)

type termCheckerVSCode struct{ term.TermChecker }

func (t *termCheckerVSCode) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, consts.CheckTermFailed)
		return false, p
	}

	envTP, okTP := pr.LookupEnv(`TERM_PROGRAM`)
	if !okTP || envTP != `vscode` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, consts.CheckTermFailed)
		return false, p
	}
	envTPV, okTPV := pr.LookupEnv(`TERM_PROGRAM_VERSION`)
	if !okTPV || len(envTPV) == 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, consts.CheckTermFailed)
		return false, p
	}

	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVSCode, consts.CheckTermPassed)
	p.SetProperty(propkeys.VSCodeVersion, envTPV)
	ver := strings.Split(envTPV, `.`)
	if len(ver) == 3 {
		p.SetProperty(propkeys.VSCodeVersionMajor, ver[0])
		p.SetProperty(propkeys.VSCodeVersionMinor, ver[1])
		p.SetProperty(propkeys.VSCodeVersionPatch, ver[2])
	}

	return true, p
}
