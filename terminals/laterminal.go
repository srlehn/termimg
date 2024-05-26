package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// LaTerminal (iOS)
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerLaTerminal{term.NewTermCheckerCore(termNameLaTerminal)})
}

const termNameLaTerminal = `laterminal`

var _ term.TermChecker = (*termCheckerLaTerminal)(nil)

type termCheckerLaTerminal struct{ term.TermChecker }

func (t *termCheckerLaTerminal) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameLaTerminal, consts.CheckTermFailed)
		return false, p
	}
	if v, ok := pr.LookupEnv(`LC_TERMINAL`); !ok || v != `LA_TERMINAL` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameLaTerminal, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameLaTerminal, consts.CheckTermPassed)
	return true, p
}

// https://docs.la-terminal.net/documentation/la_terminal/image-support
// https://blog.la-terminal.net/images-on-the-command-line/

// TODO query timeouts might be too short (remote)
