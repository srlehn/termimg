package pkgterm

import (
	pkgTerm "github.com/pkg/term"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYPkgTerm struct {
	*pkgTerm.Term
	fileName string
}

var _ term.TTY = (*TTYPkgTerm)(nil)

func New(ttyFile string) (*TTYPkgTerm, error) {
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
	return &TTYPkgTerm{Term: t, fileName: ttyFile}, nil
}

func (t *TTYPkgTerm) Write(b []byte) (n int, err error) {
	if t == nil || t.Term == nil {
		return 0, errors.NilReceiver()
	}
	return t.Term.Write(b)
}

func (t *TTYPkgTerm) Read(p []byte) (n int, err error) {
	if t == nil || t.Term == nil {
		return 0, errors.NilReceiver()
	}
	return t.Term.Read(p)
}
func (t *TTYPkgTerm) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *TTYPkgTerm) Close() error {
	if t == nil || t.Term == nil {
		return nil
	}
	defer func() { t.Term = nil; t = nil }()
	t.Term.Close()
	return errors.New(t.Term.Close())
}
