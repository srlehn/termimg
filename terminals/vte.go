package terminals

import (
	"strconv"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/xdg"
	"github.com/srlehn/termimg/term"
)

////////////////////////////////////////////////////////////////////////////////
// VTE
////////////////////////////////////////////////////////////////////////////////

func init() { term.RegisterTermChecker(&termCheckerVTE{term.NewTermCheckerCore(termNameVTE)}) }

const termNameVTE = `vte`

var _ term.TermChecker = (*termCheckerVTE)(nil)

type termCheckerVTE struct{ term.TermChecker }

func (t *termCheckerVTE) CheckExclude(pr environ.Properties) (mightBe bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVTE, consts.CheckTermFailed)
		return false, p
	}
	// TODO wayst sets this - imitates vte v0.56.2, v0.62.1
	// https://github.com/91861/wayst/commit/565b1f9
	envV, okV := pr.LookupEnv(`VTE_VERSION`)
	if !okV || len(envV) == 0 {
		envV, okV = pr.LookupEnv(`VTE`)
		if !okV || len(envV) == 0 {
			p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVTE, consts.CheckTermFailed)
			return false, p
		}
	}
	p.SetProperty(propkeys.CheckTermEnvExclPrefix+termNameVTE, consts.CheckTermPassed)
	p.SetProperty(propkeys.VTEVersion, envV)

	ver, err := strconv.ParseUint(envV, 10, 64)
	if err == nil {
		p.SetProperty(propkeys.VTEVersionMajor, strconv.Itoa(int(ver/10000)))
		p.SetProperty(propkeys.VTEVersionMinor, strconv.Itoa(int((ver%10000)/100)))
		p.SetProperty(propkeys.VTEVersionPatch, strconv.Itoa(int(ver%100)))
	}

	return true, p
}
func (t *termCheckerVTE) CheckIsQuery(qu term.Querier, tty term.TTY, pr environ.Properties) (is bool, p environ.Properties) {
	p = environ.NewProprietor()
	if t == nil || pr == nil {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameVTE, consts.CheckTermFailed)
		return false, p
	}
	term.QueryDeviceAttributes(qu, tty, pr, pr)
	da3ID, _ := pr.Property(propkeys.DA3ID)
	var vteDA3ID = `~VTE` // hex encoded: `7E565445`
	if da3ID != vteDA3ID {
		p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameVTE, consts.CheckTermFailed)
		return false, p
	}
	p.SetProperty(propkeys.CheckTermQueryIsPrefix+termNameVTE, consts.CheckTermPassed)

	return true, p
}

func (t *termCheckerVTE) Exe(pr environ.Properties) string {
	vteTerms := []string{
		"mate-terminal",
		"gnome-terminal",
		"sakura",
		"tilda",
		"tilix",
		"guake",
		"terminator",
		"terminus",
	}
	vteTerm := vteTerms[0]
	exes, err := xdg.InstalledTerminalsExe()
	if err != nil {
		return vteTerm
	}
	for _, tm := range vteTerms {
		if slices.Contains(exes, tm) {
			vteTerm = tm
			break
		}
	}
	return vteTerm
}

func (t *termCheckerVTE) Surveyor(pr environ.Properties) term.PartialSurveyor {
	var major, minor uint64
	envVMaj, okVMaj := pr.Property(propkeys.VTEVersionMajor)
	if okVMaj || len(envVMaj) > 0 {
		m, err := strconv.ParseUint(envVMaj, 10, 64)
		if err == nil {
			major = m
		}
	}
	if major >= 1 {
		return nil
	}
	envVMin, okVMin := pr.Property(propkeys.VTEVersionMinor)
	if okVMin || len(envVMin) > 0 {
		m, err := strconv.ParseUint(envVMin, 10, 64)
		if err != nil {
			return nil
		}
		minor = m
	}
	if minor >= 66 {
		return nil
	}

	return &surveyorVTEInterchanged14t{}
}

var _ term.PartialSurveyor = (*surveyorVTEInterchanged14t)(nil)

type surveyorVTEInterchanged14t struct{ term.SurveyorDefault }

// SizeInPixelsQuery - dtterm window manipulation CSI 14 t
func (s *surveyorVTEInterchanged14t) SizeInPixelsQuery(qu term.Querier, tty term.TTY) (widthPixels, heightPixels uint, e error) {
	// query terminal size in pixels
	// answer: <termHeightInPixels>;<termWidthInPixels>t (SHOULD)
	//
	// reported x and y are switched - BUG(VTE)
	// https://gitlab.gnome.org/GNOME/vte/-/issues/2509
	// answer: <termWidthInPixels>;<termHeightInPixels>t (BUG)
	// "Fixed on master, 0-66 and 0-64."
	//
	// 0.64: bug fixed after 0.64.2, but no more 0.64.x release
	// 0.65: 0.65.91 still with bug
	// 0.66: 0.66.0 fixed
	qs := "\033[14t"
	/*if needsWrap {
		qs = mux.Wrap(qs)
	}*/
	repl, err := qu.Query(qs, tty, parser.StopOnAlpha)
	if err != nil {
		return 0, 0, err
	}
	if len(repl) > 1 && repl[len(repl)-1] == 't' {
		repl = repl[:len(repl)-1]
	}
	q := strings.Split(repl, `;`)

	if len(q) != 3 {
		return 0, 0, errors.New(`unknown format`)
	}

	// reported x and y are switched - BUG(VTE)
	var x, y uint
	if xx, err := strconv.Atoi(string(q[1])); err == nil {
		if yy, err := strconv.Atoi(string(q[2])); err == nil {
			x = uint(xx)
			y = uint(yy)
		} else {
			return 0, 0, errors.New(err)
		}
	} else {
		return 0, 0, errors.New(err)
	}

	return x, y, nil
}
