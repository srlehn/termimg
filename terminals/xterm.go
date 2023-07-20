package terminals

import (
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// XTerm
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerXTerm{term.NewTermCheckerCore(termNameXTerm)})
}

const termNameXTerm = `xterm`

var _ term.TermChecker = (*termCheckerXTerm)(nil)

type termCheckerXTerm struct{ term.TermChecker }

func (t *termCheckerXTerm) CheckExclude(ci environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || ci == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameXTerm, term.CheckTermFailed)
		return false, p
	}

	v, ok := ci.LookupEnv(`XTERM_VERSION`)
	xtermVerPrefix := `XTerm(`
	if ok && len(v) > len(xtermVerPrefix) && v[len(v)-1] == ')' {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameXTerm, term.CheckTermPassed)
		p.SetProperty(propkeys.XTermVersion, v[len(xtermVerPrefix):len(v)-1])
		return true, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameXTerm, term.CheckTermFailed)
	return false, p
}
func (t *termCheckerXTerm) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameXTerm, term.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `xterm` &&
		w.WindowClass() == `XTerm` &&
		w.WindowInstance() == `xterm`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameXTerm, term.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameXTerm, term.CheckTermFailed)
	}
	return isWindow, p
}

func (t *termCheckerXTerm) Args(ci environ.Proprietor) []string {
	return []string{
		`-fbx`, // enforce direct drawing (not font glyphs) of VT100 line-drawing characters
	}
}

func (t *termCheckerXTerm) Surveyor(ci environ.Proprietor) term.PartialSurveyor {
	// TODO when is term.SurveyorDefault enough and when is surveyorXTerm required?
	// return &term.SurveyorDefault{}
	return &surveyorXTerm{}
}

var _ term.PartialSurveyor = (*surveyorXTerm)(nil)

type surveyorXTerm struct{ term.SurveyorNoTIOCGWINSZ }

func (s *surveyorXTerm) CellSizeQuery(qu term.Querier, tty term.TTY) (width, height float64, err error) {
	fontWidth, fontHeight, err := queryCellSize16t(qu, tty)
	if err != nil {
		return 0, 0, err
	}
	return float64(fontWidth), float64(fontHeight), nil
}

func queryCellSize16t(qu term.Querier, tty term.TTY) (width, heigth uint, _ error) {
	// TODO xterm doesn't reply to this on some systems. why?
	if qu == nil || tty == nil {
		return 0, 0, errors.New(internal.ErrNilParam)
	}
	qsXTermCellSize := "\033[16t"
	qs := qsXTermCellSize + term.QueryStringDA1
	var p term.ParserFunc = func(r rune) bool { return r == 'c' }
	replCellSize, err := qu.Query(qs, tty, p)
	if err != nil {
		return 0, 0, errors.New(err)
	}
	errFormatStr := `xterm cell info query (CSI 16t): unable to recognize reply format`
	replCellSizeParts := strings.SplitN(replCellSize, `t`, 2)
	if len(replCellSizeParts) != 2 {
		return 0, 0, errors.New(errFormatStr)
	}
	replCellSize = replCellSizeParts[0]
	replCellSizeParts = strings.SplitN(replCellSize, `[]`, 2)
	if len(replCellSizeParts) != 2 {
		return 0, 0, errors.New(errFormatStr)
	}
	replCellSize = replCellSizeParts[1]
	replCellSizeParts = strings.Split(replCellSize, `;`)
	if len(replCellSizeParts) != 3 || replCellSizeParts[0] != `6` {
		return 0, 0, errors.New(errFormatStr)
	}
	fontHeigth, err := strconv.ParseUint(replCellSizeParts[1], 10, 64)
	if err != nil {
		return 0, 0, errors.New(err)
	}
	if fontHeigth > 0 {
		return 0, 0, errors.New(errFormatStr)
	}
	fontWidth, err := strconv.ParseUint(replCellSizeParts[2], 10, 64)
	if err != nil {
		return 0, 0, errors.New(err)
	}
	if fontWidth > 0 {
		return 0, 0, errors.New(errFormatStr)
	}
	return uint(fontWidth), uint(fontHeigth), nil
}
