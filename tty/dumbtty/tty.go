package dumbtty

import (
	"os"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYDumb struct {
	f        *os.File
	fileName string
}

var _ term.TTY = (*TTYDumb)(nil)

func New(ttyFile string) (*TTYDumb, error) {
	f, err := os.OpenFile(ttyFile, os.O_RDWR, 0)
	if err != nil {
		return nil, errors.New(err)
	}
	return &TTYDumb{
		f:        f,
		fileName: ttyFile,
	}, nil
}

func (t *TTYDumb) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.NilReceiver()
	}
	if t.f == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.f.Write(b)
}

func (t *TTYDumb) Read(p []byte) (n int, err error) {
	if t == nil || t.f == nil {
		return 0, errors.NilReceiver()
	}
	return t.f.Read(p)
}
func (t *TTYDumb) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *TTYDumb) Close() error {
	if t == nil || t.f == nil {
		return nil
	}
	return t.f.Close()
}
