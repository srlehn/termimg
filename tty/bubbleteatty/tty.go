//go:build dev

package bubbleteatty

import (
	"io"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/contdtty"
)

const ioDeadLine = 100 * time.Millisecond

type TTYBubbleTea struct {
	Program          *tea.Program
	ttyContD         *contdtty.TTYContD
	file             *os.File
	wrappedWriter    io.Writer
	fileName         string
	reader           *teaReader
	subReader        io.Reader
	lipGlossRenderer *lipgloss.Renderer
	Scanner          Scanner
	enver            environ.Enver
	initOnce         sync.Once
	initFunc         func()
}

var _ term.TTY = (*TTYBubbleTea)(nil)

func New(ttyFile string) (*TTYBubbleTea, error) {
	return newWithProgram(nil, nil)(ttyFile)
}

func newWithProgram(model tea.Model, prog **tea.Program, opts ...tea.ProgramOption) func(string) (*TTYBubbleTea, error) {
	return func(ttyFile string) (*TTYBubbleTea, error) {
		ttyContD, err := contdtty.New(ttyFile)
		if err != nil {
			return nil, err
		}
		r := newTeaReader(ttyContD)
		t := &TTYBubbleTea{
			ttyContD:  ttyContD,
			file:      os.NewFile(ttyContD.Fd(), ttyFile),
			fileName:  ttyFile,
			reader:    r,
			subReader: r.NewSubReader(),
		}
		if prog != nil {
			t.initFunc = func() {
				opts = append(opts, t.BubbleTeaOptions()...)
				t.Program = tea.NewProgram(model, opts...)
				if t.Program == nil {
					return
				}
				*prog = t.Program
			}
		}
		return t, nil
	}
}

func (t *TTYBubbleTea) init() {
	if t == nil || t.initFunc == nil {
		return
	}
	t.initOnce.Do(t.initFunc)
}

func BubbleTeaProgram(model tea.Model, prog **tea.Program, opts ...tea.ProgramOption) term.Option {
	enforceBubbleTeaTTY := true
	return term.Options{
		term.SetTTYProvider(newWithProgram(model, prog, opts...), enforceBubbleTeaTTY),
		term.TUIMode,
		term.OptFunc(func(tm *term.Terminal) error {
			onClose := func() error {
				if tm == nil {
					return nil
				}
				t, ok := tm.TTY().(*TTYBubbleTea)
				if !ok || t == nil {
					return nil
				}
				if t.Program == nil {
					return nil
				}
				go t.Program.Quit()
				time.Sleep(1000 / 3 * time.Millisecond)
				t.Program.Kill()
				return nil
			}
			tm.OnClose(onClose)
			return nil
		}),
		term.AfterSetup(func(tm *term.Terminal) {
			t, ok := tm.TTY().(*TTYBubbleTea)
			if !ok || t == nil {
			}
			t.enver = tm.Env()
		}),
	}
}

func (t *TTYBubbleTea) TermEnvOptions() []termenv.OutputOption {
	if t == nil || t.enver == nil {
		return nil
	}
	termenvOpts := []termenv.OutputOption{
		termenv.WithUnsafe(), // termimg requires a tty
		termenv.WithColorCache(true),
		termenv.WithEnvironment(t.enver),
		termenv.WithProfile(termenv.EnvColorProfile()),
	}
	return termenvOpts
}

func (t *TTYBubbleTea) LipGlossRenderer() *lipgloss.Renderer {
	if t == nil || t.enver == nil {
		return nil
	}
	if t.lipGlossRenderer != nil {
		return t.lipGlossRenderer
	}
	termenvOpts := t.TermEnvOptions()
	t.lipGlossRenderer = lipgloss.NewRenderer(t.bubblyWriter(), termenvOpts...)
	return t.lipGlossRenderer
}

func (t *TTYBubbleTea) BubbleTeaOptions() []tea.ProgramOption {
	if t == nil {
		return nil
	}
	return []tea.ProgramOption{
		tea.WithInput(t.reader.NewSubReader()),
		// tea.WithInput(t.ttyContD),
		tea.WithOutput(t.bubblyWriter()),
		// tea.WithOutput(t.ttyContD),
	}
}

func (t *TTYBubbleTea) Read(p []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.ttyContD, t.file); err != nil {
		return 0, err
	}
	t.init()
	if err := t.file.SetReadDeadline(time.Now().Add(ioDeadLine)); err == nil {
		defer t.file.SetReadDeadline(time.Time{})
	}
	return t.subReader.Read(p)
}

func (t *TTYBubbleTea) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.ttyContD, t.file); err != nil {
		return 0, err
	}
	t.init()
	if err := t.file.SetWriteDeadline(time.Now().Add(ioDeadLine)); err == nil {
		defer t.file.SetWriteDeadline(time.Time{})
	}
	return t.ttyContD.Write(b)
}

func (t *TTYBubbleTea) bubblyWriter() io.Writer {
	// called from t.init() - don't call t.init() here
	if t == nil {
		return nil
	}
	if t.wrappedWriter == nil {
		t.wrappedWriter = &bubblyWriter{TTYBubbleTea: t}
	}
	return t.wrappedWriter
}

type bubblyWriter struct{ *TTYBubbleTea }

func (t *bubblyWriter) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.TTYBubbleTea); err != nil {
		return 0, err
	}
	// return os.Stdout.Write(b)
	if t.Scanner != nil {
		b = t.Scanner.Scan(b)
	}
	n, err = t.TTYBubbleTea.Write(b)
	if t.Scanner != nil {
		t.Scanner.PostWrite()
	}
	return n, err
}

type Scanner interface {
	Scan(b []byte) []byte
	PostWrite()
}

func (t *TTYBubbleTea) SetScanner(scanner Scanner) {
	if t == nil {
		return
	}
	t.Scanner = scanner
}

func (t *TTYBubbleTea) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	t.init()
	return t.fileName
}

// Close ...
func (t *TTYBubbleTea) Close() error {
	if t == nil {
		return nil
	}
	if t.Program != nil {
		t.Program.Quit()
		t.Program = nil
	}
	var errs []error
	if t.ttyContD != nil {
		errs = append(errs, t.ttyContD.Close())
		t.ttyContD = nil
	}
	if t.file != nil {
		errs = append(errs, t.file.Close())
		t.file = nil
	}
	t = nil
	return errors.Join(errs...)
}
