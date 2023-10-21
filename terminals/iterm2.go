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
)

////////////////////////////////////////////////////////////////////////////////
// iTerm2
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerITerm2{term.NewTermCheckerCore(termNameITerm2)})
}

const termNameITerm2 = `iterm2`

var _ term.TermChecker = (*termCheckerITerm2)(nil)

type termCheckerITerm2 struct{ term.TermChecker }

func (t *termCheckerITerm2) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	if _, isRemote := pr.Property(propkeys.IsRemote); !isRemote {
		if v, ok := pr.LookupEnv(`TERM_PROGRAM`); !ok || v != `iTerm.app` {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameITerm2, consts.CheckTermFailed)
			return false, p
		}
		if ver, okV := pr.LookupEnv(`TERM_PROGRAM_VERSION`); okV && len(ver) > 0 {
			p.SetProperty(propkeys.ITerm2VersionTPV, ver)
		}
	}
	if v, ok := pr.LookupEnv(`LC_TERMINAL`); !ok || v != `iTerm2` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameITerm2, consts.CheckTermPassed)
	return true, p
}

func (t *termCheckerITerm2) CheckIsQuery(qu term.Querier, tty term.TTY, pr environ.Properties) (is bool, p environ.Properties) {
	p = environ.NewProperties()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	// https://github.com/kmgrant/macterm/issues/3#issuecomment-458387953
	xtVer, okXTVer := pr.Property(propkeys.XTVERSION)
	if !okXTVer {
		// TODO check if XTVERSION was queried
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	iTerm2XTVersionPrefix := `iTerm2 `
	xtVer, hasITerm2Prefix := strings.CutPrefix(xtVer, iTerm2XTVersionPrefix)
	if !hasITerm2Prefix {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	// query: CSI '1337n'
	// https://iterm2.com/utilities/it2check
	// https://github.com/mintty/mintty/issues/881#issuecomment-614601911
	replITerm2Ver, err := term.CachedQuery(qu, queries.ITerm2PropVersion+queries.DA1, tty, &parser.ITerm2DA1Parser{}, pr, p)
	if err != nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	iTerm2PropVersionPrefix := `ITERM2 `
	propVer, hasITerm2Prefix := strings.CutPrefix(replITerm2Ver, iTerm2PropVersionPrefix)
	if !hasITerm2Prefix {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameITerm2, consts.CheckTermFailed)
		return false, p
	}
	propVer = strings.ToLower(propVer)
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameITerm2, consts.CheckTermPassed)
	p.SetProperty(propkeys.ITerm2VersionXTVersion, xtVer)
	p.SetProperty(propkeys.ITerm2VersionProprietary, propVer)
	return true, p
}

func (t *termCheckerITerm2) Surveyor(pr environ.Properties) term.PartialSurveyor {
	// return &term.SurveyorNoANSI{}
	return &surveyorITerm2{}
}

var _ term.PartialSurveyor = (*surveyorITerm2)(nil)

type surveyorITerm2 struct{ term.SurveyorDefault }

func (s *surveyorITerm2) CellSizeQuery(qu term.Querier, tty term.TTY) (width, height float64, err error) {
	// https://iterm2.com/documentation-escape-codes.html
	qs := queries.ITerm2CellSize + queries.DA1
	repl, err := qu.Query(qs, tty, parser.NewParser(false, true))
	if err != nil {
		return 0, 0, err
	}
	replParts := strings.SplitN(repl, queries.ST, 2)
	if len(replParts) != 2 {
		replParts := strings.SplitN(repl, "\a", 2)
		if len(replParts) != 2 {
			return 0, 0, errors.New(`no reply to iterm2 ReportCellSize query`)
		}
	}
	reportCellSizePrefix := queries.OSC + `1337;ReportCellSize=`
	repl, hasPrefix := strings.CutPrefix(replParts[0], reportCellSizePrefix)
	if !hasPrefix {
		return 0, 0, errors.New(`invalid reply to iterm2 ReportCellSize query`)
	}
	replParts = strings.Split(repl, `;`)
	if l := len(replParts); l < 2 || l > 3 {
		return 0, 0, errors.New(`invalid reply to iterm2 ReportCellSize query`)
	}
	// height:width(;scale)
	var fontRes [2]float64
	for i := 0; i < 2; i++ {
		f, err := strconv.ParseFloat(replParts[i], 64)
		if err != nil {
			return 0, 0, errors.New(err)
		}
		fontRes[i] = f
	}
	fontWidth, fontHeight := fontRes[1], fontRes[0]
	return fontWidth, fontHeight, nil
}
