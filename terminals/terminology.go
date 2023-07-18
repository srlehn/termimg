package terminals

import (
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Terminology
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerTerminology{term.NewTermCheckerCore(termNameTerminology)})
}

const termNameTerminology = `terminology`

var _ term.TermChecker = (*termCheckerTerminology)(nil)

type termCheckerTerminology struct{ term.TermChecker }

func (t *termCheckerTerminology) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTerminology, term.CheckTermFailed)
		return false, p
	}
	v, ok := ci.LookupEnv(`TERMINOLOGY`)
	if ok && v == "1" {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTerminology, term.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTerminology, term.CheckTermFailed)
	return false, p
}

func (t *termCheckerTerminology) CheckIsQuery(qu term.Querier, tty term.TTY, ci environ.Proprietor) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameTerminology, term.CheckTermFailed)
		return false, p
	}
	term.QueryDeviceAttributes(qu, tty, ci, ci)
	da3ID, _ := ci.Property(propkeys.DA3ID)
	var terminologyDA3ID = `~~TY` // hex encoded: `7E7E5459`
	if da3ID != terminologyDA3ID {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameTerminology, term.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameTerminology, term.CheckTermPassed)
	return true, p
}

// func (t *TermTerminology) X11WindowClass() string { return `terminology` }

/*
https://www.enlightenment.org/docs/apps/terminology.md#tycat
https://github.com/borisfaure/terminology/blob/master/src/bin/tycat.c

TERMINOLOGY=1 tycat image.png // not working: "not directly running in terminology"
#DEFINE ON_NOT_RUNNING_IN_TERMINOLOGY_EXIT_1()
https://github.com/borisfaure/terminology/blob/aca88e2/src/bin/tycommon.h#L7
expect_running_in_terminology() # compares DA3 reply
https://github.com/borisfaure/terminology/blob/aca88e2/src/bin/tycommon.c#L13

https://github.com/borisfaure/terminology#extended-escapes-for-terminology-only

image print
https://github.com/borisfaure/terminology/blob/master/src/bin/tycat.c#LL69C1-L69C1 # print()
snprintf(buf, sizeof(buf), "%c}is#%i;%i;%s", 0x1b, w, h, path)

sizes
https://github.com/borisfaure/terminology/blob/master/src/bin/tyls.c#L926
snprintf(buf, sizeof(buf), "%c}qs", 0x1b)
scanf("%i;%i;%i;%i", &tw, &th, &cw, &ch)
*/
