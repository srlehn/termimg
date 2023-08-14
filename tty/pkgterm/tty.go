//go:build dev

package pkgterm

import (
	"errors"

	errorsGo "github.com/go-errors/errors"
	pkgTerm "github.com/pkg/term"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/term"
)

type ttyPkgTerm struct {
	*pkgTerm.Term
	fileName string
}

var _ term.TTY = (*ttyPkgTerm)(nil)

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

func (t *ttyPkgTerm) ReadRune() (r rune, size int, err error) {
	// TODO implement ReadRune()
	r = '\uFFFD'
	return r, 0, errorsGo.New(internal.ErrNotImplemented)
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
	defer func() { t.Term = nil }()
	errRestore := t.Term.Restore()
	errClose := t.Term.Close()
	err := errors.Join(errRestore, errClose)
	if err != nil {
		return errorsGo.New(err)
	}
	return nil
}
