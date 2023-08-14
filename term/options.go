package term

import (
	errorsGo "github.com/go-errors/errors"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/wm"
)

// TODO use functional options?

type Option interface {
	ModifyTerminal(t *Terminal) error
}

var _ Option = (Opt)(nil)

type Opt func(*Terminal) error

func (o Opt) ModifyTerminal(t *Terminal) error { return o(t) }

var _ Option = (Options)(nil)

type Options []Option

func (o Options) ModifyTerminal(t *Terminal) error { return t.SetOptions([]Option(o)...) }

func (t *Terminal) SetOptions(opts ...Option) error {
	if t == nil {
		t = &Terminal{}
	}
	for _, opt := range opts {
		if err := opt.ModifyTerminal(t); err != nil {
			return errorsGo.New(err)
		}
	}
	return nil
}

func SetPTYName(ptyName string) Option {
	return Opt(func(t *Terminal) error { t.ptyName = ptyName; return nil })
}
func SetTTY(tty TTY) Option {
	return Opt(func(t *Terminal) error { t.tty = tty; return nil })
}
func SetTTYFallback(ttyFallback TTY) Option {
	return Opt(func(t *Terminal) error { t.ttyDefault = ttyFallback; return nil })
}
func SetTTYProvFallback(ttyProvFallback TTYProvider) Option {
	return Opt(func(t *Terminal) error { t.ttyProvDefault = ttyProvFallback; return nil })
}
func SetQuerier(qu Querier) Option {
	return Opt(func(t *Terminal) error { t.querier = qu; return nil })
}
func SetQuerierFallback(quFallback Querier) Option {
	return Opt(func(t *Terminal) error { t.querierDefault = quFallback; return nil })
}
func SetPartialSurveyor(ps PartialSurveyor) Option {
	return Opt(func(t *Terminal) error { t.partialSurveyor = ps; return nil })
}
func SetPartialSurveyorFallback(psFallback PartialSurveyor) Option {
	return Opt(func(t *Terminal) error { t.partialSurveyorDefault = psFallback; return nil })
}
func SetWindowProvider(wProv wm.WindowProvider) Option {
	return Opt(func(t *Terminal) error { t.windowProvider = wProv; return nil })
}
func SetWindowProviderFallback(wProvFallback wm.WindowProvider) Option {
	return Opt(func(t *Terminal) error { t.windowProviderDefault = wProvFallback; return nil })
}
func SetResizer(rsz Resizer) Option {
	return Opt(func(t *Terminal) error { t.resizer = rsz; return nil })
}
func SetProprietor(pr environ.Proprietor) Option {
	return Opt(func(t *Terminal) error { t.proprietor = pr; return nil })
}
func SetTerminalName(termName string) Option {
	return Opt(func(t *Terminal) error { t.name = termName; return nil })
}
func SetExe(exe string) Option {
	return Opt(func(t *Terminal) error { t.exe = exe; return nil })
}
func SetArgs(args []string) Option {
	return Opt(func(t *Terminal) error { t.arger = newArger(args); return nil })
}
func SetDrawers(drs []Drawer) Option {
	return Opt(func(t *Terminal) error { t.drawers = drs; return nil })
}
func SetWindow(w wm.Window) Option {
	return Opt(func(t *Terminal) error { t.w = w; return nil })
}
func replaceTerminal(t *Terminal) Option {
	return Opt(func(tOld *Terminal) error { tOld = t; return nil })
}
