package terminals

import (
	"strconv"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

////////////////////////////////////////////////////////////////////////////////
// URXVT
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerURXVT{TermChecker: term.NewTermCheckerCore(termNameURXVT)})
}

const termNameURXVT = `urxvt`

var _ term.TermChecker = (*termCheckerURXVT)(nil)

type termCheckerURXVT struct {
	term.TermChecker
	canQueryCellSize776 bool
}

func (t *termCheckerURXVT) CheckExclude(pr environ.Proprietor) (mightBe bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameURXVT, consts.CheckTermFailed)
		return false, p
	}
	s, ok := pr.LookupEnv(`TERM`) // TODO tmux overwrites this
	urxvtPrefix := `rxvt-unicode`
	mayBeURXVT := ok && len(s) >= len(urxvtPrefix) && s[:len(urxvtPrefix)] == urxvtPrefix
	if !mayBeURXVT {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameURXVT, consts.CheckTermFailed)
	} else {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameURXVT, consts.CheckTermPassed)
		// TODO move code
		border, hasBorder := pr.Property(propkeys.XResourcesPrefix + `URxvt.internalBorder`)
		if hasBorder && len(border) > 0 && border != `0` {
			if _, err := strconv.ParseUint(border, 10, 64); err == nil {
				p.SetProperty(propkeys.WindowBorderEstimated+`_`+termNameURXVT, border)
			}
		}
	}
	return mayBeURXVT, p
}
func (t *termCheckerURXVT) CheckIsQuery(qu term.Querier, tty term.TTY, pr environ.Proprietor) (is bool, p environ.Proprietor) {
	// https://terminalguide.namepad.de/seq/osc-702/
	// $ printf '\033]702;?\033\'
	// # "\033]702;rxvt-unicode;urxvt;9;2\033"

	// TODO urxvt isn't recognized anymore

	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameURXVT, consts.CheckTermFailed)
		return false, p
	}

	var b bool
	var urxvtParser term.ParserFunc = func(r rune) bool {
		if r == '\033' {
			if b {
				return true
			}
			b = true
		}
		return false
	}
	// example reply: 702;rxvt-unicode;urxvt;9;2\x1b
	replVersion, err := term.CachedQuery(qu, "\033]702;?\033\\", tty, urxvtParser, pr, p)
	if err != nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameURXVT, consts.CheckTermFailed)
		return false, p // TODO ?
	}
	replVersion = strings.TrimSuffix(
		strings.TrimPrefix(
			strings.TrimPrefix(
				replVersion,
				"\033]",
			),
			"]",
		),
		"\033",
	)
	urxvtReplPrefix := "702;rxvt-unicode;"
	replVersion, found := strings.CutPrefix(replVersion, urxvtReplPrefix)
	if !found {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameURXVT, consts.CheckTermFailed)
		return false, p
	}
	r := strings.Split(strings.TrimRight(replVersion, "\033"), `;`)
	if len(r) != 3 {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameURXVT, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.URXVTExeName, r[0])
	p.SetProperty(propkeys.URXVTVerFirstChar, r[1])
	p.SetProperty(propkeys.URXVTVerThirdChar, r[2])

	if _, _, err := queryCellSize776(qu, tty); err == nil {
		t.canQueryCellSize776 = true
	}

	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameURXVT, consts.CheckTermPassed)
	return true, p
}
func (t *termCheckerURXVT) CheckIsWindow(w wm.Window) (is bool, p environ.Proprietor) {
	p = environ.NewProprietor()
	if t == nil || w == nil {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameURXVT, consts.CheckTermFailed)
		return false, p
	}
	isWindow := w.WindowType() == `x11` &&
		w.WindowName() == `urxvt` /* TODO remove? (can be modified) */ &&
		w.WindowClass() == `URxvt` &&
		w.WindowInstance() == `urxvt`
	if isWindow {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameURXVT, consts.CheckTermPassed)
	} else {
		p.SetProperty(propkeys.CheckTermWindowIsPrefix+termNameURXVT, consts.CheckTermFailed)
	}

	return isWindow, p
}

// func (t *TermURXVT) X11WindowName() string  { return `urxvt` }
// func (t *TermURXVT) X11WindowClass() string { return `URxvt` }

func (t *termCheckerURXVT) Surveyor(pr environ.Proprietor) term.PartialSurveyor {
	if t == nil || !t.canQueryCellSize776 {
		return nil
	}
	return &surveyorURXVT{}
}

var _ term.PartialSurveyor = (*surveyorURXVT)(nil)

type surveyorURXVT struct{ term.SurveyorDefault }

func (s *surveyorURXVT) CellSizeQuery(qu term.Querier, tty term.TTY) (width, height float64, err error) {
	fontWidth, fontHeight, err := queryCellSize776(qu, tty)
	if err != nil {
		return 0, 0, err
	}
	return float64(fontWidth), float64(fontHeight), nil
}

func queryCellSize776(qu term.Querier, tty term.TTY) (width, heigth uint, _ error) {
	// cell size query
	// urxvt: "\033]776;?\033\\" -> "\033]776;%d;%d;%d%s" (font (width, height, ascent (baseline)))
	// urxvt: "\033]776;?\033\\"+DA1 -> assumed response: \033]776;%d;%d;%d(\033(\\|)|\a)(\033|)[?%d;%d;...c" - use parser.StopOnAlpha
	if qu == nil || tty == nil {
		return 0, 0, errors.New(consts.ErrNilParam)
	}
	qsURXVTCellSize := "\033]776;?\033\\"
	qs := qsURXVTCellSize + queries.DA1
	replCellSize, err := qu.Query(qs, tty, parser.StopOnAlpha)
	if err != nil {
		return 0, 0, errors.New(err)
	}
	errFormatStr := `urxvt cell info query (776): unable to recognize reply format`
	replCellSizeParts := strings.SplitN(replCellSize, `[`, 2)
	if len(replCellSizeParts) != 2 {
		return 0, 0, errors.New(errFormatStr)
	}
	replCellSize = replCellSizeParts[0]
	replCellSizeParts = strings.SplitN(replCellSize, `]`, 2)
	if len(replCellSizeParts) != 2 {
		return 0, 0, errors.New(errFormatStr)
	}
	replCellSize = replCellSizeParts[1]
	replCellSizeParts = strings.Split(replCellSize, `;`)
	if len(replCellSizeParts) != 4 || replCellSizeParts[0] != `776` {
		return 0, 0, errors.New(errFormatStr)
	}
	fontWidth, err := strconv.ParseUint(replCellSizeParts[1], 10, 64)
	if err != nil {
		return 0, 0, errors.New(err)
	}
	if fontWidth > 0 {
		return 0, 0, errors.New(errFormatStr)
	}
	fontHeigth, err := strconv.ParseUint(replCellSizeParts[2], 10, 64)
	if err != nil {
		return 0, 0, errors.New(err)
	}
	if fontHeigth > 0 {
		return 0, 0, errors.New(errFormatStr)
	}
	return uint(fontWidth), uint(fontHeigth), nil
}
