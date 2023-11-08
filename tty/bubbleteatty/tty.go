package bubbleteatty

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/contdtty"
)

type TTYBubbleTea struct {
	*tea.Program
	ttyContD *contdtty.TTYContD
	fileName string
	// buf             []rune
}

// const maxSizeBuf = 1024 * 1024

var _ term.TTY = (*TTYBubbleTea)(nil)

// func tea.NewProgram(model tea.Model, opts ...tea.ProgramOption) *tea.Program

func New(ttyFile string) (*TTYBubbleTea, error) {
	var model tea.Model
	return NewWithOptions(model)(ttyFile)
}

func NewWithOptions(model tea.Model, opts ...tea.ProgramOption) func(ttyFile string) (*TTYBubbleTea, error) {
	return func(ttyFile string) (*TTYBubbleTea, error) {
		if model == nil {
			return nil, errors.New(`nil tea.Model`)
		}
		ttyContD, err := contdtty.New(ttyFile)
		if err != nil {
			return nil, err
		}
		var model tea.Model
		t := &TTYBubbleTea{
			ttyContD: ttyContD,
			fileName: ttyFile,
		}
		/*flt := func(_ tea.Model, msg tea.Msg) tea.Msg {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				t.buf = append(t.buf, msg.Runes...)
				if len(t.buf) > maxSizeBuf { // TODO truncate before append
					t.buf = t.buf[len(t.buf)-maxSizeBuf:]
				}
			}
			return msg
		}*/
		opts = append(opts, []tea.ProgramOption{
			// tea.WithFilter(flt),
			tea.WithInput(ttyContD),
			tea.WithOutput(ttyContD),
		}...)
		t.Program = tea.NewProgram(model, opts...)
		return t, nil
	}
}

func (t *TTYBubbleTea) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.ttyContD); err != nil {
		return 0, err
	}
	return t.ttyContD.Write(b)
}

func (t *TTYBubbleTea) Read(p []byte) (n int, err error) {
	if t == nil || t.ttyContD == nil {
		return 0, errors.NilReceiver()
	}
	// TODO read t.buf?
	return t.ttyContD.Read(p)
}

func (t *TTYBubbleTea) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
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
	var err error
	if t.ttyContD != nil {
		err = t.ttyContD.Close()
		t.ttyContD = nil
	}
	t = nil
	return err
}
