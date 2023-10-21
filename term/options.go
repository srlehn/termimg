package term

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/srlehn/termimg/env/advanced"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
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
		t = newDummyTerminal()
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.ApplyOption(t); logx.IsErr(err, t, slog.LevelError) {
			return errors.New(err)
		}
	}
	return nil
}

func SetPTYName(ptyName string) Option {
	return OptFunc(func(t *Terminal) error {
		if t.properties == nil {
			t.properties = environ.NewProperties()
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
		if merge && t.properties != nil {
			t.properties.MergeProperties(pr)
		} else {
			t.properties = pr
		}
		return nil
	})
}
func SetTerminalName(termName string) Option {
	return OptFunc(func(t *Terminal) error {
		if t.properties == nil {
			t.properties = environ.NewProperties()
		}
		t.SetProperty(propkeys.TerminalName, `true`)
		return nil
	})
}
func SetExe(exe string) Option {
	return OptFunc(func(t *Terminal) error {
		if t.properties == nil {
			t.properties = environ.NewProperties()
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
func SetSLogHandler(h slog.Handler, enable bool) Option {
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
func SetLogFile(filename string, enable bool) Option {
	return OptFunc(func(t *Terminal) error {
		if enable {
			logFile, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}
			hOpts := &slog.HandlerOptions{
				AddSource: true,
			}
			t.logger = slog.New(slog.NewTextHandler(logFile, hOpts))
			t.logInfo(`opening log file`)
			logBuildInfo(t)
			t.OnClose(func() error {
				if logFile == nil {
					return nil
				}
				t.logInfo(`closing log file`)
				return logFile.Close()
			})
		} else {
			t.logger = nil
		}
		return nil
	})
}
func logBuildInfo(loggerProv logx.LoggerProvider) {
	if loggerProv == nil {
		return
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok || bi == nil {
		return
	}
	args := []any{`go-version`, bi.GoVersion}
	for _, m := range append([]*debug.Module{&bi.Main}, bi.Deps...) {
		if m != nil && strings.HasSuffix(m.Path, consts.LibraryName) {
			if len(m.Version) > 0 {
				args = append(args, consts.LibraryName+`-version`, m.Version)
			}
			if len(m.Sum) > 0 {
				args = append(args, consts.LibraryName+`-checksum`, m.Sum)
			}
		}
	}
	for _, bs := range bi.Settings {
		switch bs.Key {
		case `CGO_ENABLED`,
			`CGO_CFLAGS`, `CGO_CPPFLAGS`, `CGO_CXXFLAGS`, `CGO_LDFLAGS`,
			`GOARCH`,
			`GOAMD64`, `GOARM`, `GO386`,
			`GOOS`,
			`vcs`, `vcs.revision`, `vcs.time`, `vcs.modified`:
			if len(bs.Value) > 0 {
				args = append(args, bs.Key, bs.Value)
			}
		}
	}
	logx.Info(`build info`, loggerProv, args...)
}

var (
	TUIMode              Option = tuiMode
	CLIMode              Option = cliMode
	ManualComposition    Option = manualComposition    // disable terminal detection
	NoCleanUpOnInterrupt Option = noCleanUpOnInterrupt // disable terminal cleanup on Interrupt or TERM signal
)

var tuiMode Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.Mode, `tui`); return nil })
var cliMode Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.Mode, `cli`); return nil })
var manualComposition Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.ManualComposition, `true`); return nil })
var noCleanUpOnInterrupt Option = OptFunc(func(t *Terminal) error { t.SetProperty(propkeys.NoCleanUpOnInterrupt, `true`); return nil })

// replaceTerminal is for injecting an already set up *terminal.Terminal into *termCheckerCore.NewTerminal()
// replacing its dummy one.
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
		if logx.IsErr(err, t, slog.LevelInfo) {
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
			if t.properties != nil {
				composeManuallyStr, composeManually := t.Property(propkeys.ManualComposition)
				if composeManually && composeManuallyStr == `true` {
					envIsLoadedStr, envIsLoaded := t.Property(propkeys.EnvIsLoaded)
					if envIsLoaded && envIsLoadedStr == `true` {
						return nil
					}
				}
			}
		} else {
			if t.properties != nil || t.passages != nil {
				return nil
			}
		}
		ptyName := t.ptyName()
		if len(ptyName) == 0 {
			if t.properties != nil {
				ptyNameDefault := internal.DefaultTTYDevice()
				t.SetProperty(propkeys.PTYName, ptyNameDefault)
				ptyName = ptyNameDefault
			}
		}
		pr, passages, err := advanced.GetEnv(ptyName)
		_ = logx.IsErr(err, t, slog.LevelInfo)
		var skipSettingEnvIsLoaded bool
		if t.properties != nil {
			envIsLoadedStr, _ := t.Property(propkeys.EnvIsLoaded)
			switch envIsLoadedStr {
			case `true`:
				skipSettingEnvIsLoaded = true
			case `merge`:
				// use new Proprietor as receiver to allow implementation choice
				pr.MergeProperties(t.properties)
			}
		}
		if !skipSettingEnvIsLoaded {
			t.properties = pr
		}
		if t.properties != nil {
			t.SetProperty(propkeys.EnvIsLoaded, `true`)
		}
		t.passages = passages

		// X11 Resources

		xdgSessionType, hasXDGSessionType := t.properties.LookupEnv(`XDG_SESSION_TYPE`)
		if !hasXDGSessionType || xdgSessionType != `x11` {
			return nil
		}

		conn, err := wm.NewConn(t.properties)
		if logx.IsErr(err, t, slog.LevelInfo) {
			return err
		}
		res, err := conn.Resources()
		if logx.IsErr(err, t, slog.LevelInfo) {
			return err
		}
		if t.properties != nil {
			t.properties.MergeProperties(res)
		} else {
			t.properties = res
		}

		return nil
	})
}
