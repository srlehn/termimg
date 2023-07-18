package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Tabby
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerTabby{term.NewTermCheckerCore(termNameTabby)})
}

const termNameTabby = `tabby`

var _ term.TermChecker = (*termCheckerTabby)(nil)

type termCheckerTabby struct{ term.TermChecker }

func (t *termCheckerTabby) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, term.CheckTermFailed)
		return false, p
	}
	envTP, okTP := ci.LookupEnv(`TERM_PROGRAM`)
	if !okTP || envTP != `Tabby` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, term.CheckTermFailed)
		return false, p
	}
	_, okTVS := ci.LookupEnv(`TABBY_PLUGINS`)
	if !okTVS {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, term.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, term.CheckTermPassed)
	return true, p
}

// reports sixel capability but display no images
