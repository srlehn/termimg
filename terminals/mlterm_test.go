package terminals

import (
	"testing"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

var envMlterm = `WINDOWID=90177552
TERM=mlterm
COLORFGBG=default;default
MLTERM=3.9.0`

func TestTermCheckerMltermCheckExclude(t *testing.T) {
	env := propFromText(envMlterm)
	chk := &termCheckerMlterm{term.NewTermCheckerCore(termNameMlterm)}

	mightBe, env := chk.CheckExclude(env)
	if !mightBe {
		t.Fatal(`(*termCheckerMlterm)CheckExclude() failed`)
	}
	if p, ok := env.Property(propkeys.CheckTermEnvExclPrefix + termNameMlterm); ok && p != consts.CheckTermPassed {
		t.Fatal(`(*termCheckerMlterm)CheckExclude() didn't set "passed" property`)
	}
}

func TestTermCheckerMltermCheckIsWindow(t *testing.T) {
	wm.SetImpl(wmimpl.Impl())
	w := wm.CreateWindow(`mlterm`, `mlterm`, `xterm`)

	chk := &termCheckerMlterm{term.NewTermCheckerCore(termNameMlterm)}

	is, pr := chk.CheckIsWindow(w)
	if !is {
		t.Fatal(`(*termCheckerXTerm)CheckIsWindow() failed`)
	}
	if p, ok := pr.Property(propkeys.CheckTermWindowIsPrefix + termNameMlterm); ok && p != consts.CheckTermPassed {
		t.Fatal(`(*termCheckerXTerm)CheckIsWindow() didn't set "passed" property`)
	}
}
