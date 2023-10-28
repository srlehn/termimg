package pkgterm

import (
	pkgTerm "github.com/pkg/term"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyPkgTerm struct {
	*pkgTerm.Term
	fileName string
}

var _ term.TTY = (*ttyPkgTerm)(nil)
var _ term.TTYProvider = New

func New(ttyFile string) (term.TTY, error) {
	opts := []func(*pkgTerm.Term) error{
		pkgTerm.CBreakMode,
		// pkgTerm.RawMode,
	}
	t, err := pkgTerm.Open(ttyFile, opts...)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New(`nil tty`)
	}
	return &ttyPkgTerm{Term: t, fileName: ttyFile}, nil
}

func (t *ttyPkgTerm) Write(b []byte) (n int, err error) {
	if t == nil || t.Term == nil {
		return 0, errors.NilReceiver()
	}
	return t.Term.Write(b)
}

func (t *ttyPkgTerm) Read(p []byte) (n int, err error) {
	if t == nil || t.Term == nil {
		return 0, errors.NilReceiver()
	}
	return t.Term.Read(p)
}
func (t *ttyPkgTerm) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *ttyPkgTerm) Close() error {
	if t == nil || t.Term == nil {
		return nil
	}
	defer func() { t.Term = nil; t = nil }()
	t.Term.Close()
	return errors.New(t.Term.Close())
}
