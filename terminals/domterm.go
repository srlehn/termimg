package terminals

import (
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// DomTerm
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerDomTerm{term.NewTermCheckerCore(termNameDomTerm)})
}

const termNameDomTerm = `domterm`

var _ term.TermChecker = (*termCheckerDomTerm)(nil)

type termCheckerDomTerm struct{ term.TermChecker }

func (t *termCheckerDomTerm) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameDomTerm, consts.CheckTermFailed)
		return false, p
	}
	vDT, okEnvDT := pr.LookupEnv(`DOMTERM`)
	if !okEnvDT || len(vDT) == 0 {
		/*
			// TODO probably just from the chrome browser process
			vArg, errEnvArg := t.EnvVar(`ARGV0`)
			return errEnvArg == nil && vArg != `domterm`
		*/
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameDomTerm, consts.CheckTermFailed)
		return false, p
	}
	for _, propStr := range strings.Split(vDT, `;`) {
		// https://github.com/PerBothner/DomTerm/blob/master/native/pty/pty.c
		for _, propName := range []string{`version`, `libwebsockets`, `tty`, `session#`, `pid`} {
			if !strings.HasPrefix(propStr, propName+`=`) {
				continue
			}
			p.SetProperty(propkeys.DomTermPrefix+strings.TrimRight(propName, `#`), propStr[len(propName)+1:])
			break
		}
	}
	if ciw, okW := pr.(wm.Window); okW && ciw != nil {
		wName := ciw.WindowName()
		wInstance := ciw.WindowInstance()
		if !strings.HasPrefix(wName, `[DomTerm:`) || !strings.HasSuffix(wInstance, `_domterm_start.html`) {
			return false, nil
		}
		p.SetProperty(propkeys.DomTermWindowName, wName)
		p.SetProperty(propkeys.DomTermWindowInstance, wInstance)
	}

	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameDomTerm, consts.CheckTermPassed)
	return true, p
}

func (t *termCheckerDomTerm) CheckIsWindow(w wm.Window) (is bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameDomTerm, consts.CheckTermFailed)
		return false, p
	}
	if strings.HasPrefix(w.WindowName(), `[DomTerm:`) || strings.HasSuffix(w.WindowInstance(), `_domterm_start.html`) {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameDomTerm, consts.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameDomTerm, consts.CheckTermFailed)
	return false, p
}

func (t *termCheckerDomTerm) Args(pr environ.Properties) []string { return []string{`--no-daemonize`} }

/*
example env: DOMTERM=version=2.9.0;libwebsockets=4.1.99-v4.1.0-339-g124cbe02;tty=/dev/pts/3;session#=1;pid=9471
https://github.com/PerBothner/DomTerm/blob/master/bin/imgcat
https://github.com/PerBothner/DomTerm/blob/master/bin/svgcat
*/
