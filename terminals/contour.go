package terminals

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// contour
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerContour{term.NewTermCheckerCore(termNameContour)})
}

const termNameContour = `contour`

var _ term.TermChecker = (*termCheckerContour)(nil)

type termCheckerContour struct{ term.TermChecker }

func (t *termCheckerContour) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameContour, consts.CheckTermFailed)
		return false, p
	}
	envTN, okTN := pr.LookupEnv(`TERMINAL_NAME`)
	if !okTN || envTN != `contour` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameContour, consts.CheckTermFailed)
		return false, p
	}
	// X.X.X.X
	envTVS, okTVS := pr.LookupEnv(`TERMINAL_VERSION_STRING`)
	if !okTVS || len(envTVS) == 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameContour, consts.CheckTermFailed)
		return false, p
	}
	// X.X.X
	envTVT, okTVT := pr.LookupEnv(`TERMINAL_VERSION_TRIPLE`)
	if !okTVT || len(envTVT) == 0 {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameContour, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameContour, consts.CheckTermPassed)
	p.SetProperty(propkeys.ContourVersion, envTVS)
	return true, p
}
func (t *termCheckerContour) CheckIsWindow(w wm.Window) (is bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameContour, consts.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `contour` &&
		w.WindowClass() == `contour` &&
		w.WindowInstance() == `contour`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameContour, consts.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameContour, consts.CheckTermFailed)
	}
	return isWindow, p
}
