package term

import (
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
)

func init() { RegisterTermChecker(&termCheckerGeneric{NewTermCheckerCore(internal.TermGenericName)}) }

var _ TermChecker = (*termCheckerGeneric)(nil)

type termCheckerGeneric struct{ TermChecker }

func (t *termCheckerGeneric) Name() string { return internal.TermGenericName }
func (t *termCheckerGeneric) CheckExclude(environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+internal.TermGenericName, CheckTermPassed)
	// inform that this is the final check (no CheckIsQuery)
	p.SetProperty(propkeys.CheckTermCompletePrefix+internal.TermGenericName, CheckTermPassed)
	// match any input
	return true, p
}
