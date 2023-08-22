package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// Kitty
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerKitty{term.NewTermCheckerCore(termNameKitty)})
}

const termNameKitty = `kitty`

var _ term.TermChecker = (*termCheckerKitty)(nil)

type termCheckerKitty struct{ term.TermChecker }

func (t *termCheckerKitty) CheckExclude(pr environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameKitty, consts.CheckTermFailed)
		return false, p
	}
	envI, okI := pr.LookupEnv(`KITTY_WINDOW_ID`)
	if !okI || len(envI) == 0 {
		envT, _ := pr.LookupEnv(`TERM`)
		isKitty := envT == `xterm-kitty`
		if isKitty {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameKitty, consts.CheckTermPassed)
		} else {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameKitty, consts.CheckTermFailed)
		}
		return isKitty, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameKitty, consts.CheckTermPassed)
	p.SetProperty(propkeys.KittyWindowID, envI) // kitty tab id
	return true, p
}
func (t *termCheckerKitty) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, consts.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowClass() == `kitty` &&
		w.WindowInstance() == `kitty`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, consts.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameMlterm, consts.CheckTermFailed)
	}
	return isWindow, p
}

// https://sw.kovidgoyal.net/kitty/graphics-protocol.html
