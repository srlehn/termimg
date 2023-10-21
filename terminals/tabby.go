package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
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

func (t *termCheckerTabby) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, consts.CheckTermFailed)
		return false, p
	}
	envTP, okTP := pr.LookupEnv(`TERM_PROGRAM`)
	if !okTP || envTP != `Tabby` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, consts.CheckTermFailed)
		return false, p
	}
	_, okTVS := pr.LookupEnv(`TABBY_PLUGINS`)
	if !okTVS {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTabby, consts.CheckTermPassed)
	return true, p
}

// reports sixel capability but display no images
