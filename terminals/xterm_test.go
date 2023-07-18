package terminals

import (
	"strings"
	"testing"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

const envXTerm = `WINDOWID=90177548
XTERM_VERSION=XTerm(366)
XTERM_LOCALE=en_US.UTF-8
TERM=xterm`

func TestTermCheckerXTermCheckExclude(t *testing.T) {
	wm.SetImpl(wmimpl.Impl())
	env := propFromText(envXTerm)
	chk := &termCheckerXTerm{term.NewTermCheckerCore(termNameXTerm)}

	mightBe, env := chk.CheckExclude(env)
	if !mightBe {
		t.Fatal(`(*termCheckerXTerm)CheckExclude() failed`)
	}
	if p, ok := env.Property(propkeys.CheckTermEnvExclPrefix + termNameXTerm); ok && p != term.CheckTermPassed {
		t.Fatal(`(*termCheckerXTerm)CheckExclude() didn't set "passed" property`)
	}
}

func TestTermCheckerXTermCheckIsWindow(t *testing.T) {
	w := wm.CreateWindow(`xterm`, `XTerm`, `xterm`)

	chk := &termCheckerXTerm{term.NewTermCheckerCore(termNameXTerm)}

	is, pr := chk.CheckIsWindow(w)
	if !is {
		t.Fatal(`(*termCheckerXTerm)CheckIsWindow() failed`)
	}
	if p, ok := pr.Property(propkeys.CheckTermWindowIsPrefix + termNameXTerm); ok && p != term.CheckTermPassed {
		t.Fatal(`(*termCheckerXTerm)CheckIsWindow() didn't set "passed" property`)
	}
}

func propFromText(e string) environ.Proprietor {
	env := environ.NewProprietor()
	for _, line := range strings.Split(e, "\n") {
		parts := strings.SplitN(line, `=`, 2)
		if len(parts) != 2 {
			continue
		}
		env.SetProperty(propkeys.EnvPrefix+parts[0], parts[1])
	}
	return env
}
