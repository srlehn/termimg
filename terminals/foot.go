package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Foot
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerFoot{term.NewTermCheckerCore(termNameFoot)})
}

const termNameFoot = `foot`

var _ term.TermChecker = (*termCheckerFoot)(nil)

type termCheckerFoot struct{ term.TermChecker }

func (t *termCheckerFoot) CheckIsQuery(qu term.Querier, tty term.TTY, ci environ.Proprietor) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameFoot, term.CheckTermFailed)
		return false, p
	}
	term.QueryDeviceAttributes(qu, tty, ci, ci)
	da3ID, _ := ci.Property(propkeys.DA3ID)
	var footDA3ID = `FOOT` // hex encoded: `464F4F54`
	if da3ID != footDA3ID {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameFoot, term.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameFoot, term.CheckTermPassed)
	return true, p
}

// https://codeberg.org/dnkl/foot#programmatically-checking-if-running-in-foot
/*
The secondary DA response is \E[>1;XXYYZZ;0c, where XXYYZZ is foot's major, minor and patch version numbers,
in decimal, using two digits for each number. For example, foot-1.4.2 would respond with \E[>1;010402;0c.

Starting with version 1.7.0, foot also implements XTVERSION, to which it will reply with \EP>|foot(version)\E\\.
Version is e.g. “1.8.2” for a regular release, or “1.8.2-36-g7db8e06f” for a git build.
*/
