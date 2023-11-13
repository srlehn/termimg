package bubbleteatty

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/contdtty"
)

const ioDeadLine = 100 * time.Millisecond
const maxSizeBuf = 1024 * 1024

type TTYBubbleTea struct {
	program        *tea.Program
	ttyContD       *contdtty.TTYContD
	f              *os.File
	bw             io.Writer
	postWriteFuncs []struct { // ordered list of image widget draw funcs, etc
		id string
		f  func()
	}
	fileName string
	inputRdr io.Reader
	inputBuf bytes.Buffer
	runeBuf  []rune
}

var _ term.TTY = (*TTYBubbleTea)(nil)

func New(ttyFile string) (*TTYBubbleTea, error) {
	ttyContD, err := contdtty.New(ttyFile)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(ttyContD.Fd(), ttyFile)
	t := &TTYBubbleTea{
		ttyContD: ttyContD,
		f:        f,
		fileName: ttyFile,
	}
	return t, nil
}

func (t *TTYBubbleTea) SetProgram(ctx context.Context, model tea.Model, opts ...tea.ProgramOption) (*tea.Program, error) {
	if err := errors.NilReceiver(t); err != nil {
		return nil, err
	}
	if err := errors.NilParam(model); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	t.inputRdr = bufio.NewReader(io.TeeReader(t.ttyContD, &t.inputBuf))
	opts = append(opts, []tea.ProgramOption{
		tea.WithContext(ctx),
		// tea.WithFilter(t.filterMessage),
		// tea.WithInput(&t.inputBuf),
		tea.WithInput(&t.inputBuf),
		tea.WithOutput(t.bubblyWriter()),
	}...)
	t.program = tea.NewProgram(model, opts...)
	if t.program == nil {
		return nil, errors.New(`nil bubbletea Program`)
	}
	return t.program, nil
}

func (t *TTYBubbleTea) Read(p []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.ttyContD, t.f); err != nil {
		return 0, err
	}
	if err := t.f.SetReadDeadline(time.Now().Add(ioDeadLine)); err == nil {
		defer t.f.SetReadDeadline(time.Time{})
	}
	// TODO read t.runeBuf?
	// return t.ttyContD.Read(p)
	if t.inputRdr == nil { // not yet inititated?
		return t.ttyContD.Read(p)
	} else {
		return t.inputRdr.Read(p)
	}
	// return t.inputBuf.Read(p)
	// return t.inputRdr.Read(p)
}

func (t *TTYBubbleTea) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.ttyContD, t.f); err != nil {
		return 0, err
	}
	if err := t.f.SetWriteDeadline(time.Now().Add(ioDeadLine)); err == nil {
		defer t.f.SetWriteDeadline(time.Time{})
	}
	return t.ttyContD.Write(b)
}

func (t *TTYBubbleTea) bubblyWriter() io.Writer {
	if t == nil {
		return nil
	}
	if t.bw != nil {
		return t.bw
	}
	t.bw = &bubblyWriter{t}
	return t.bw
}

type bubblyWriter struct{ *TTYBubbleTea }

func (t *bubblyWriter) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.TTYBubbleTea); err != nil {
		return 0, err
	}
	n, err = t.TTYBubbleTea.Write(b)
	for _, r := range t.TTYBubbleTea.postWriteFuncs {
		r.f() // draw images, etc
	}
	//reset
	t.TTYBubbleTea.postWriteFuncs = nil
	return n, err
}

// SetAfterWriteFunc sets a temporary functions that will be called once after the next Write.
func (t *TTYBubbleTea) SetAfterWriteFunc(id string, f func()) {
	if t == nil {
		return
	}
	t.postWriteFuncs = append(
		// remove previously planned draws for this widget id
		slices.DeleteFunc(t.postWriteFuncs, func(r struct {
			id string
			f  func()
		}) bool {
			return r.id == id
		}),
		struct {
			id string
			f  func()
		}{id: id, f: f},
	)
}

func (t *TTYBubbleTea) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

func (t *TTYBubbleTea) filterMessage(mdl tea.Model, msg tea.Msg) tea.Msg {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		t.runeBuf = append(t.runeBuf, msg.Runes...)
		if len(t.runeBuf) > maxSizeBuf {
			// truncate
			t.runeBuf = t.runeBuf[len(t.runeBuf)-maxSizeBuf:]
		}
	}
	return msg
}

// Close ...
func (t *TTYBubbleTea) Close() error {
	if t == nil {
		return nil
	}
	if t.program != nil {
		t.program.Quit()
		t.program = nil
	}
	var errs []error
	if t.ttyContD != nil {
		errs = append(errs, t.ttyContD.Close())
		t.ttyContD = nil
	}
	if t.f != nil {
		errs = append(errs, t.f.Close())
		t.f = nil
	}
	t = nil
	return errors.Join(errs...)
}

func TTYOf(tm *term.Terminal) (*TTYBubbleTea, error) {
	if err := errors.NilParam(tm); err != nil {
		return nil, err
	}
	tty, ok := tm.TTY().(*TTYBubbleTea)
	if !ok {
		return nil, errors.New(`wrong tty implementation`)
	}
	if tty == nil {
		return nil, errors.New(`nil bubbletea tty`)
	}
	return tty, nil
}

func ProgramOf(tm *term.Terminal) (*tea.Program, *TTYBubbleTea, error) {
	tty, err := TTYOf(tm)
	if err != nil {
		return nil, nil, err
	}
	if tty.program == nil {
		return nil, tty, errors.New(`nil bubbletea Program`)
	}
	return tty.program, tty, nil
}
