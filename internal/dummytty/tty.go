package dummytty

import (
	"io"
	"strings"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYDummy struct {
	rdr      *strings.Reader
	fileName string
}

var _ term.TTY = (*TTYDummy)(nil)

func New(buf string) (*TTYDummy, error) {
	rdr := strings.NewReader(buf)
	return &TTYDummy{
		rdr:      rdr,
		fileName: internal.DefaultTTYDevice(),
	}, nil
}

func (t *TTYDummy) Write(b []byte) (n int, err error) { return io.Discard.Write(b) }

func (t *TTYDummy) Read(p []byte) (n int, err error) {
	if t == nil || t.rdr == nil {
		return 0, errors.NilReceiver()
	}
	return t.rdr.Read(p)
}
func (t *TTYDummy) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *TTYDummy) Close() error { return nil }
