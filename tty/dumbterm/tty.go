package dumbterm

import (
	"os"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttydumb struct {
	*os.File
	fileName string
}

var _ term.TTY = (*ttydumb)(nil)

func New(ttyFile string) (term.TTY, error) { return newTTY(ttyFile) }

func newTTY(ttyFile string) (_ term.TTY, err error) {
	f, err := os.OpenFile(ttyFile, os.O_RDWR, 0)
	if err != nil {
		return nil, errors.New(err)
	}
	return &ttydumb{
		File:     f,
		fileName: ttyFile,
	}, nil
}

func (t *ttydumb) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.New(consts.ErrNilReceiver)
	}
	if t.File == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.File.Write(b)
}

func (t *ttydumb) Read(p []byte) (n int, err error) {
	if t == nil || t.File == nil {
		return 0, errors.New(consts.ErrNilReceiver)
	}
	return t.File.Read(p)
}
func (t *ttydumb) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *ttydumb) Close() error {
	if t == nil || t.File == nil {
		return nil
	}
	return t.File.Close()
}
