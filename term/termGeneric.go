package term

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
)

func init() { RegisterTermChecker(&termCheckerGeneric{NewTermCheckerCore(consts.TermGenericName)}) }

var _ TermChecker = (*termCheckerGeneric)(nil)

type termCheckerGeneric struct{ TermChecker }

func (t *termCheckerGeneric) Name() string { return consts.TermGenericName }
func (t *termCheckerGeneric) CheckExclude(Properties) (mightBe bool, p Properties) {
	p = environ.NewProperties()
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+consts.TermGenericName, consts.CheckTermPassed)
	// inform that this is the final check (no CheckIsQuery)
	p.SetProperty(propkeys.CheckTermCompletePrefix+consts.TermGenericName, consts.CheckTermPassed)
	// match any input
	return true, p
}
