package terminals

import (
	"strconv"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// Hyper
////////////////////////////////////////////////////////////////////////////////

func init() {
	term.RegisterTermChecker(&termCheckerHyper{term.NewTermCheckerCore(termNameHyper)})
}

const termNameHyper = `hyper`

var _ term.TermChecker = (*termCheckerHyper)(nil)

type termCheckerHyper struct{ term.TermChecker }

func (t *termCheckerHyper) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameHyper, consts.CheckTermFailed)
		return false, p
	}
	v, ok := pr.LookupEnv(`TERM_PROGRAM`)
	if !ok || v != `Hyper` {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameHyper, consts.CheckTermFailed)
		return false, p
	}

	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameHyper, consts.CheckTermPassed)
	if ver, okV := pr.LookupEnv(`TERM_PROGRAM_VERSION`); okV && len(ver) > 0 {
		p.SetProperty(propkeys.HyperVersion, ver)
		verParts := strings.SplitN(ver, `.`, 3)
		if len(verParts) != 3 {
			goto end
		}
		if _, err := strconv.ParseUint(verParts[0], 10, 64); err != nil {
			goto end
		}
		if _, err := strconv.ParseUint(verParts[1], 10, 64); err != nil {
			goto end
		}
		patchParts := strings.SplitN(verParts[2], `-canary.`, 2)
		if _, err := strconv.ParseUint(patchParts[0], 10, 64); err != nil {
			goto end
		}
		p.SetProperty(propkeys.HyperVersionMajor, verParts[0])
		p.SetProperty(propkeys.HyperVersionMinor, verParts[1])
		p.SetProperty(propkeys.HyperVersionPatch, patchParts[0])
		if len(patchParts) < 2 || len(patchParts[1]) == 0 {
			goto end
		}
		if _, err := strconv.ParseUint(patchParts[1], 10, 64); err != nil {
			goto end
		}
		p.SetProperty(propkeys.HyperVersionCanary, patchParts[1])
	}
end:
	return true, p
}

func (t *termCheckerHyper) Surveyor(pr environ.Properties) term.PartialSurveyor {
	// BUG in v4.0.0-canary.4: CSI 14t reports wrong pixel size
	return &surveyorHyperWrong14t{}
}

var _ term.PartialSurveyor = (*surveyorHyperWrong14t)(nil)

type surveyorHyperWrong14t struct {
	term.SurveyorDefault
	survXTerm surveyorXTerm
}

func (s *surveyorHyperWrong14t) SizeInPixelsQuery(qu term.Querier, tty term.TTY) (widthPixels, heightPixels uint, e error) {
	// disable dtterm window manipulation CSI 14 t
	// BUG in v4.0.0-canary.4: CSI 14t reports wrong pixel size
	return 0, 0, errors.New(consts.ErrNotImplemented)
}

func (s *surveyorHyperWrong14t) CellSizeQuery(qu term.Querier, tty term.TTY) (width, height float64, err error) {
	if s == nil {
		return 0, 0, errors.New(consts.ErrNilReceiver)
	}
	// likely also wrong
	return 0, 0, errors.New(consts.ErrNotImplemented)

	// TODO
	// time.Sleep(125 * time.Millisecond) // slow terminal
	// return s.survXTerm.CellSizeQuery(qu, tty)
}
