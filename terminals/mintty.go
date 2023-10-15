package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// mintty
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerMintty{term.NewTermCheckerCore(termNameMintty)})
}

const termNameMintty = `mintty`

var _ term.TermChecker = (*termCheckerMintty)(nil)

type termCheckerMintty struct{ term.TermChecker }

func (t *termCheckerMintty) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMintty, consts.CheckTermFailed)
		return false, p
	}
	var r bool

	/*
		MINTTY_SHORTCUT=/cygdrive/c/Users/user/Desktop/mintty.lnk
		TERM_PROGRAM=mintty
		TERM_PROGRAM_VERSION=3.6.4
	*/

	// only set if mintty was started via .lnk shortcut
	vMS, okMS := pr.LookupEnv(`MINTTY_SHORTCUT`)
	r = r || okMS
	if okMS && len(vMS) > 0 {
		p.SetProperty(propkeys.MinttyShortcut, vMS)
	}
	vTP, okTP := pr.LookupEnv(`TERM_PROGRAM`)
	r = r || (okTP && vTP == `mintty`)
	if r {
		vTPV, okTPV := pr.LookupEnv(`TERM_PROGRAM_VERSION`)
		if okTPV && len(vTPV) > 0 {
			p.SetProperty(propkeys.MinttyVersion, vTPV)
		}
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMintty, consts.CheckTermPassed)
		return true, p
	}

	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameMintty, consts.CheckTermFailed)
	return false, p
}

func (t *termCheckerMintty) CheckIsQuery(qu term.Querier, tty term.TTY, pr environ.Properties) (is bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameMintty, consts.CheckTermFailed)
		return false, p
	}
	// https://github.com/mintty/mintty/issues/881#issuecomment-499687377
	term.QueryDeviceAttributes(qu, tty, pr, pr)
	da2Model, _ := pr.Property(propkeys.DA2ModelLetter)
	var minttyDA2Model = `M`
	if da2Model != minttyDA2Model {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameMintty, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameMintty, consts.CheckTermPassed)
	return true, p
}
