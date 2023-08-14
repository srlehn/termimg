package term

import (
	errorsGo "github.com/go-errors/errors"
	"github.com/srlehn/termimg/env/advanced"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/wm"
)

// TODO use functional options?

type Option interface {
	ApplyOption(t *Terminal) error
}

var _ Option = (OptFunc)(nil)

type OptFunc func(*Terminal) error

func (o OptFunc) ApplyOption(t *Terminal) error { return o(t) }

var _ Option = (Options)(nil)

type Options []Option

func (o Options) ApplyOption(t *Terminal) error { return t.SetOptions([]Option(o)...) }

func (t *Terminal) SetOptions(opts ...Option) error {
	if t == nil {
		t = &Terminal{}
	}
	for _, opt := range opts {
		if err := opt.ApplyOption(t); err != nil {
			return errorsGo.New(err)
		}
	}
	return nil
}

func SetPTYName(ptyName string) Option {
	return OptFunc(func(t *Terminal) error { t.ptyName = ptyName; return nil })
}
func SetTTY(tty TTY, enforce bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enforce {
			t.tty = tty
		} else {
			t.ttyDefault = tty
		}
		return nil
	})
}
func SetTTYProvider(ttyProv TTYProvider, enforce bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enforce {
			t.ttyProv = ttyProv // TODO
		} else {
			t.ttyProvDefault = ttyProv
		}
		return nil
	})
}
func SetQuerier(qu Querier, enforce bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enforce {
			t.querier = qu
		} else {
			t.querierDefault = qu
		}
		return nil
	})
}
func SetSurveyor(ps PartialSurveyor, enforce bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enforce {
			t.partialSurveyor = ps
		} else {
			t.partialSurveyorDefault = ps
		}
		return nil
	})
}
func SetWindowProvider(wProv wm.WindowProvider, enforce bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enforce {
			t.windowProvider = wProv
		} else {
			t.windowProviderDefault = wProv
		}
		return nil
	})
}
func SetResizer(rsz Resizer) Option {
	return OptFunc(func(t *Terminal) error { t.resizer = rsz; return nil })
}
func SetProprietor(pr environ.Proprietor) Option {
	return OptFunc(func(t *Terminal) error { t.proprietor = pr; return nil })
}
func SetTerminalName(termName string) Option {
	return OptFunc(func(t *Terminal) error { t.name = termName; return nil })
}
func SetExe(exe string) Option {
	return OptFunc(func(t *Terminal) error { t.exe = exe; return nil })
}
func SetArgs(args []string) Option {
	return OptFunc(func(t *Terminal) error { t.arger = newArger(args); return nil })
}
func SetDrawers(drs []Drawer) Option {
	return OptFunc(func(t *Terminal) error { t.drawers = drs; return nil })
}
func SetWindow(w wm.Window) Option {
	return OptFunc(func(t *Terminal) error { t.w = w; return nil })
}

var (
	TUIMode           Option = tuiMode
	CLIMode           Option = cliMode
	ManualComposition Option = manualComposition // disable terminal detection
)

var tuiMode Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.Mode, `tui`); return nil })
var cliMode Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.Mode, `cli`); return nil })
var manualComposition Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.ManualComposition, `true`); return nil })

func replaceTerminal(t *Terminal) Option {
	return OptFunc(func(tOld *Terminal) error {
		if t == nil || tOld == nil {
			return errorsGo.New(`cannot swap nil terminals`)
		}
		*tOld = *t
		return nil
	})
}

var setInternalDefaults Option = OptFunc(func(t *Terminal) error {
	if len(t.ptyName) == 0 {
		t.ptyName = internal.DefaultTTYDevice()
	}
	if t.partialSurveyorDefault == nil {
		t.partialSurveyorDefault = &SurveyorDefault{}
	}
	if t.windowProviderDefault == nil {
		t.windowProviderDefault = wm.NewWindow
	}
	return nil
})

func setTTYAndQuerier(tc *termCheckerCore) Option {
	return OptFunc(func(t *Terminal) error {
		tty, qu, err := getTTYAndQuerier(t, tc)
		if err != nil {
			return err
		}
		t.tty = tty
		t.querier = qu
		return nil
	})
}

func setEnvAndMuxers(overwrite bool) Option {
	return OptFunc(func(t *Terminal) error {
		if overwrite {
			if t.proprietor != nil {
				composeManuallyStr, composeManually := t.Property(propkeys.ManualComposition)
				if composeManually && composeManuallyStr == `true` {
					envIsLoadedStr, envIsLoaded := t.Property(propkeys.EnvIsLoaded)
					if envIsLoaded && envIsLoadedStr == `true` {
						return nil
					}
				}
			}
		} else {
			if t.proprietor != nil || t.passages != nil {
				return nil
			}
		}
		if len(t.ptyName) == 0 {
			t.ptyName = internal.DefaultTTYDevice()
		}
		pr, passages, err := advanced.GetEnv(t.ptyName)
		if err != nil {
			return err
		}
		var skipSettingEnvIsLoaded bool
		if t.proprietor != nil {
			envIsLoadedStr, envIsLoaded := t.Property(propkeys.EnvIsLoaded)
			skipSettingEnvIsLoaded = envIsLoaded && envIsLoadedStr != `true`
		}
		t.proprietor = pr
		if !skipSettingEnvIsLoaded && t.proprietor != nil {
			t.SetProperty(propkeys.EnvIsLoaded, `true`)
		}
		t.passages = passages
		return nil
	})
}
