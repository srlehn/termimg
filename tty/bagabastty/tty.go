package bagabastty

import (
	pty "github.com/aymanbagabas/go-pty"
	ptyCreack "github.com/creack/pty"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYBagabas struct {
	pty.Pty
	fileName string
}

var _ term.TTY = (*TTYBagabas)(nil)

func New(ttyFile string) (*TTYBagabas, error) {
	if !internal.IsDefaultTTY(ttyFile) {
		return nil, errors.New(`only default tty supported`)
	}
	p, err := pty.New()
	if err != nil {
		return nil, errors.New(err)
	}
	return &TTYBagabas{
		Pty:      p,
		fileName: ttyFile,
	}, nil
}

func (t *TTYBagabas) Write(b []byte) (n int, err error) {
	if t == nil || t.Pty == nil {
		return 0, errors.NilReceiver()
	}
	return t.Pty.Write(b)
}

func (t *TTYBagabas) Read(p []byte) (n int, err error) {
	if t == nil || t.Pty == nil {
		return 0, errors.NilReceiver()
	}
	return t.Pty.Read(p)
}
func (t *TTYBagabas) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

func (t *TTYBagabas) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	if t == nil || t.Pty == nil {
		return 0, 0, 0, 0, errors.NilReceiver()
	}
	ptyUnix, ok := t.Pty.(pty.UnixPty)
	if !ok {
		return 0, 0, 0, 0, errors.New(consts.ErrPlatformNotSupported)
	}
	m := ptyUnix.Master()
	if m == nil {
		return 0, 0, 0, 0, errors.New(`nil tty`)
	}
	sz, err := ptyCreack.GetsizeFull(m)
	if err != nil {
		return 0, 0, 0, 0, errors.New(err)
	}
	return int(sz.Cols), int(sz.Rows), int(sz.X), int(sz.Y), nil
}

// Close ...
func (t *TTYBagabas) Close() error {
	if t == nil || t.Pty == nil {
		return nil
	}
	err := errors.New(t.Pty.Close())
	t.Pty = nil
	t = nil
	return err
}
