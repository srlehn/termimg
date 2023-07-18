package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Konsole
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerKonsole{term.NewTermCheckerCore(termNameKonsole)})
}

const termNameKonsole = `konsole`

var _ term.TermChecker = (*termCheckerKonsole)(nil)

type termCheckerKonsole struct{ term.TermChecker }

func (t *termCheckerKonsole) CheckIsQuery(qu term.Querier, tty term.TTY, ci environ.Proprietor) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameKonsole, term.CheckTermFailed)
		return false, p
	}
	term.QueryDeviceAttributes(qu, tty, ci, ci)
	da3ID, _ := ci.Property(propkeys.DA3ID)
	var konsoleDA3ID = `~KDE` // hex encoded: `7E4B4445`
	if da3ID != konsoleDA3ID {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameKonsole, term.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameKonsole, term.CheckTermPassed)
	return true, p
}
