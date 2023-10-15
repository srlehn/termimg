package term

import (
	"log/slog"

	"github.com/srlehn/termimg/env/advanced"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
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
			return errors.New(err)
		}
	}
	return nil
}

func SetPTYName(ptyName string) Option {
	return OptFunc(func(t *Terminal) error {
		if t.proprietor == nil {
			t.proprietor = environ.NewProprietor()
		}
		if len(ptyName) == 0 {
			ptyName = internal.DefaultTTYDevice()
		}
		t.SetProperty(propkeys.PTYName, ptyName)
		return nil
	})
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
func SetProprietor(pr environ.Properties, merge bool) Option {
	return OptFunc(func(t *Terminal) error {
		if merge && t.proprietor != nil {
			t.proprietor.Merge(pr)
		} else {
			t.proprietor = pr
		}
		return nil
	})
}
func SetTerminalName(termName string) Option {
	return OptFunc(func(t *Terminal) error {
		if t.proprietor == nil {
			t.proprietor = environ.NewProprietor()
		}
		t.SetProperty(propkeys.TerminalName, `true`)
		return nil
	})
}
func SetExe(exe string) Option {
	return OptFunc(func(t *Terminal) error {
		if t.proprietor == nil {
			t.proprietor = environ.NewProprietor()
		}
		if len(exe) > 0 {
			return nil
		}
		t.SetProperty(propkeys.Executable, exe)
		return nil
	})
}
func SetArgs(args []string) Option {
	return OptFunc(func(t *Terminal) error { t.arger = newArger(args); return nil })
}
func SetDrawers(drs []Drawer) Option {
	return OptFunc(func(t *Terminal) error { t.drawers = drs; return nil })
}
func SetWindow(w wm.Window) Option {
	return OptFunc(func(t *Terminal) error { t.window = w; return nil })
}
func SetSLogger(h slog.Handler, enable bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enable {
			if h == nil {
				t.logger = slog.Default()
			} else {
				t.logger = slog.New(h)
			}
		} else {
			t.logger = nil
		}
		return nil
	})
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
			return errors.New(`cannot swap nil terminals`)
		}
		*tOld = *t
		return nil
	})
}

var setInternalDefaults Option = OptFunc(func(t *Terminal) error {
	ptyName := t.ptyName()
	if len(ptyName) == 0 {
		ptyName = internal.DefaultTTYDevice()
	}
	t.SetProperty(propkeys.PTYName, ptyName)
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
		ptyName := t.ptyName()
		if len(ptyName) == 0 {
			if t.proprietor != nil {
				ptyNameDefault := internal.DefaultTTYDevice()
				t.SetProperty(propkeys.PTYName, ptyNameDefault)
				ptyName = ptyNameDefault
			}
		}
		pr, passages, err := advanced.GetEnv(ptyName)
		if err != nil {
			// TODO log error
		}
		var skipSettingEnvIsLoaded bool
		if t.proprietor != nil {
			envIsLoadedStr, _ := t.Property(propkeys.EnvIsLoaded)
			switch envIsLoadedStr {
			case `true`:
				skipSettingEnvIsLoaded = true
			case `merge`:
				// use new Proprietor as receiver to allow implementation choice
				pr.Merge(t.proprietor)
			}
		}
		if !skipSettingEnvIsLoaded {
			t.proprietor = pr
		}
		if t.proprietor != nil {
			t.SetProperty(propkeys.EnvIsLoaded, `true`)
		}
		t.passages = passages

		// X11 Resources

		xdgSessionType, hasXDGSessionType := t.proprietor.LookupEnv(`XDG_SESSION_TYPE`)
		if !hasXDGSessionType || xdgSessionType != `x11` {
			return nil
		}

		conn, err := wm.NewConn(t.proprietor)
		if err != nil {
			return err
		}
		res, err := conn.Resources()
		if err != nil {
			return err
		}
		if t.proprietor != nil {
			t.proprietor.Merge(res)
		} else {
			t.proprietor = res
		}

		return nil
	})
}
