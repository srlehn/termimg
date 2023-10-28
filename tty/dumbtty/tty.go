package dumbtty

import (
	"os"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyDumb struct {
	*os.File
	fileName string
}

var _ term.TTY = (*ttyDumb)(nil)
var _ term.TTYProvider = New

func New(ttyFile string) (term.TTY, error) {
	f, err := os.OpenFile(ttyFile, os.O_RDWR, 0)
	if err != nil {
		return nil, errors.New(err)
	}
	return &ttyDumb{
		File:     f,
		fileName: ttyFile,
	}, nil
}

func (t *ttyDumb) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.NilReceiver()
	}
	if t.File == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.File.Write(b)
}

func (t *ttyDumb) Read(p []byte) (n int, err error) {
	if t == nil || t.File == nil {
		return 0, errors.NilReceiver()
	}
	return t.File.Read(p)
}
func (t *ttyDumb) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *ttyDumb) Close() error {
	if t == nil || t.File == nil {
		return nil
	}
	return t.File.Close()
}
