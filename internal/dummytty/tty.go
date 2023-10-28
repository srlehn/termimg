package dummytty

import (
	"io"
	"strings"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyDummy struct {
	rdr      *strings.Reader
	fileName string
}

var _ term.TTY = (*ttyDummy)(nil)
var _ term.TTYProvider = New

func New(buf string) (term.TTY, error) {
	rdr := strings.NewReader(buf)
	return &ttyDummy{
		rdr:      rdr,
		fileName: internal.DefaultTTYDevice(),
	}, nil
}

func (t *ttyDummy) Write(b []byte) (n int, err error) { return io.Discard.Write(b) }

func (t *ttyDummy) Read(p []byte) (n int, err error) {
	if t == nil || t.rdr == nil {
		return 0, errors.NilReceiver()
	}
	return t.rdr.Read(p)
}
func (t *ttyDummy) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *ttyDummy) Close() error { return nil }
