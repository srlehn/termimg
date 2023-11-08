package uroottty

import (
	"github.com/u-root/u-root/pkg/termios"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYURoot struct {
	*termios.TTYIO
	termios  *termios.Termios
	fileName string
}

var _ term.TTY = (*TTYURoot)(nil)

func New(ttyFile string) (*TTYURoot, error) {
	t, err := termios.NewWithDev(ttyFile)
	if err != nil {
		return nil, errors.New(err)
	}
	cfg, err := t.Raw()
	if err != nil {
		return nil, errors.New(err)
	}
	return &TTYURoot{
		TTYIO:    t,
		termios:  cfg,
		fileName: ttyFile,
	}, nil
}

func (t *TTYURoot) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.NilReceiver()
	}
	if t.TTYIO == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.TTYIO.Write(b)
}

func (t *TTYURoot) Read(p []byte) (n int, err error) {
	if t == nil || t.TTYIO == nil {
		return 0, errors.NilReceiver()
	}
	return t.TTYIO.Read(p)
}
func (t *TTYURoot) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}
func (t *TTYURoot) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	if t == nil || t.TTYIO == nil {
		return 0, 0, 0, 0, errors.NilReceiver()
	}
	sz, err := t.GetWinSize()
	if err != nil {
		return 0, 0, 0, 0, errors.New(err)
	}
	return int(sz.Col), int(sz.Row), int(sz.Xpixel), int(sz.Ypixel), nil
}

// Close ...
func (t *TTYURoot) Close() error {
	if t == nil || t.TTYIO == nil {
		return nil
	}
	err := errors.New(t.Set(t.termios))
	t.termios = nil
	t.TTYIO = nil
	t = nil
	return err
}
