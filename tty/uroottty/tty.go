package uroottty

import (
	"github.com/u-root/u-root/pkg/termios"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyURoot struct {
	*termios.TTYIO
	*termios.Termios
	fileName string
}

var _ term.TTY = (*ttyURoot)(nil)
var _ term.TTYProvider = New

func New(ttyFile string) (term.TTY, error) {
	t, err := termios.NewWithDev(ttyFile)
	if err != nil {
		return nil, errors.New(err)
	}
	cfg, err := t.Raw()
	if err != nil {
		return nil, errors.New(err)
	}
	return &ttyURoot{
		TTYIO:    t,
		Termios:  cfg,
		fileName: ttyFile,
	}, nil
}

func (t *ttyURoot) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.NilReceiver()
	}
	if t.TTYIO == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.TTYIO.Write(b)
}

func (t *ttyURoot) Read(p []byte) (n int, err error) {
	if t == nil || t.TTYIO == nil {
		return 0, errors.NilReceiver()
	}
	return t.TTYIO.Read(p)
}
func (t *ttyURoot) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}
func (t *ttyURoot) SizePixel() (cw int, ch int, pw int, ph int, e error) {
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
func (t *ttyURoot) Close() error {
	if t == nil || t.TTYIO == nil {
		return nil
	}
	err := errors.New(t.Set(t.Termios))
	t.Termios = nil
	t.TTYIO = nil
	t = nil
	return err
}
