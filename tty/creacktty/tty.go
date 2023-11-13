//go:build !windows

package creacktty

import (
	"os"

	"github.com/creack/pty/v2"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYCreack struct {
	master   *os.File
	slave    *os.File
	fileName string
}

var _ term.TTY = (*TTYCreack)(nil)

func New(ttyFile string) (*TTYCreack, error) {
	if !internal.IsDefaultTTY(ttyFile) {
		return nil, errors.New(`only default tty supported`)
	}
	p, t, err := pty.Open()
	if err != nil {
		return nil, errors.New(err)
	}
	return &TTYCreack{
		master:   p,
		slave:    t,
		fileName: ttyFile,
	}, nil
}

func (t *TTYCreack) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.master); err != nil {
		return 0, err
	}
	return t.master.Write(b)
}

func (t *TTYCreack) Read(p []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.master); err != nil {
		return 0, err
	}
	return t.master.Read(p)
}

func (t *TTYCreack) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

func (t *TTYCreack) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	// TODO bugged
	if t == nil || t.master == nil {
		return 0, 0, 0, 0, errors.NilReceiver()
	}
	sz, err := pty.GetsizeFull(t.master)
	if err != nil {
		return 0, 0, 0, 0, errors.New(err)
	}
	return int(sz.Cols), int(sz.Rows), int(sz.X), int(sz.Y), nil
}

// Close ...
func (t *TTYCreack) Close() error {
	if t == nil {
		return nil
	}
	var errM, errS error
	if t.master != nil {
		errM = t.master.Close()
		t.master = nil
	}
	if t.slave != nil {
		errS = t.slave.Close()
		t.slave = nil
	}
	defer func() { t = nil }()
	return errors.Join(errM, errS)
}
