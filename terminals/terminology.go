package terminals

import (
	"strconv"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/queries"
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

func (t *termCheckerTerminology) CheckExclude(pr environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTerminology, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`TERMINOLOGY`)
	if ok && v == "1" {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTerminology, consts.CheckTermPassed)
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameTerminology, consts.CheckTermFailed)
	return false, p
}

func (t *termCheckerTerminology) CheckIsQuery(qu term.Querier, tty term.TTY, pr environ.Proprietor) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameTerminology, consts.CheckTermFailed)
		return false, p
	}
	term.QueryDeviceAttributes(qu, tty, pr, pr)
	da3ID, _ := pr.Property(propkeys.DA3ID)
	var terminologyDA3ID = `~~TY` // hex encoded: `7E7E5459`
	if da3ID != terminologyDA3ID {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameTerminology, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameTerminology, consts.CheckTermPassed)
	return true, p
}

func (t *termCheckerTerminology) Surveyor(pr environ.Proprietor) term.PartialSurveyor {
	// return &term.SurveyorNoANSI{}
	return &surveyorTerminology{}
}

var _ term.PartialSurveyor = (*surveyorTerminology)(nil)

type surveyorTerminology struct {
	surveyorDefault term.SurveyorDefault
}

func (s *surveyorTerminology) IsPartialSurveyor() {}
func (s *surveyorTerminology) SizeInCellsQuery(qu term.Querier, tty term.TTY) (widthCells, heightCells uint, err error) {
	tpw, tph, _, _, err := queryTerminalAndCellSizeTerminology(qu, tty)
	if err != nil {
		return 0, 0, err
	}
	return tpw, tph, nil
}
func (s *surveyorTerminology) CellSizeQuery(qu term.Querier, tty term.TTY) (width, height float64, err error) {
	_, _, fontWidth, fontHeight, err := queryTerminalAndCellSizeTerminology(qu, tty)
	if err != nil {
		return 0, 0, err
	}
	return float64(fontWidth), float64(fontHeight), nil
}

func queryTerminalAndCellSizeTerminology(qu term.Querier, tty term.TTY) (tpw, tph, cpw, cph uint, _ error) {
	// TODO xterm doesn't reply to this on some systems. why?
	if qu == nil || tty == nil {
		return 0, 0, 0, 0, errors.New(consts.ErrNilParam)
	}
	qsTerminologySize := "\033}qs\000"
	qs := qsTerminologySize + queries.DA1
	var p term.ParserFunc = func(r rune) bool { return r == 'c' }
	repl, err := qu.Query(qs, tty, p)
	if err != nil {
		return 0, 0, 0, 0, errors.New(err)
	}
	// "213;58;9;17\n\x1b[?64;1;9;15;18;21;22c"
	errFormatStr := `terminology terminal size query (CSI }qs\x00): unable to recognize reply format`
	replParts := strings.SplitN(repl, "\n", 2)
	if len(replParts) != 2 {
		return 0, 0, 0, 0, errors.New(errFormatStr)
	}
	repl = replParts[0]
	replParts = strings.SplitN(repl, `;`, 4)
	if len(replParts) != 4 {
		return 0, 0, 0, 0, errors.New(errFormatStr)
	}
	termWidth, err := strconv.ParseUint(replParts[0], 10, 64)
	if err != nil || termWidth <= 0 {
		return 0, 0, 0, 0, errors.New(errFormatStr)
	}
	termHeight, err := strconv.ParseUint(replParts[1], 10, 64)
	if err != nil || termHeight <= 0 {
		return 0, 0, 0, 0, errors.New(errFormatStr)
	}
	fontWidth, err := strconv.ParseUint(replParts[2], 10, 64)
	if err != nil || fontWidth <= 1 {
		return 0, 0, 0, 0, errors.New(errFormatStr)
	}
	fontHeigth, err := strconv.ParseUint(replParts[3], 10, 64)
	if err != nil || fontHeigth <= 1 {
		return 0, 0, 0, 0, errors.New(errFormatStr)
	}
	return uint(termWidth), uint(termHeight), uint(fontWidth), uint(fontHeigth), err
}

// GetCursorQuery
func (s *surveyorTerminology) GetCursorQuery(qu term.Querier, tty term.TTY) (widthCells, heightCells uint, err error) {
	if s == nil {
		return 0, 0, errors.New(consts.ErrNilReceiver)
	}
	return s.surveyorDefault.GetCursorQuery(qu, tty)
}

// SetCursorQuery
func (s *surveyorTerminology) SetCursorQuery(xPosCells, yPosCells uint, qu term.Querier, tty term.TTY) (err error) {
	if s == nil {
		return errors.New(consts.ErrNilReceiver)
	}
	return s.surveyorDefault.SetCursorQuery(xPosCells, yPosCells, qu, tty)
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

query size
# "\033}qs\000"
https://github.com/borisfaure/terminology/blob/9f97aaa/src/bin/tyls.c#L926
snprintf(buf, sizeof(buf), "%c}qs", 0x1b)
scanf("%i;%i;%i;%i", &tw, &th, &cw, &ch)
*/
