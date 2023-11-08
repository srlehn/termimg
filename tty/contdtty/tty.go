package contdtty

import (
	"os"

	"github.com/containerd/console"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYContD struct {
	console.Console
	fileName string
}

var _ term.TTY = (*TTYContD)(nil)

func New(ttyFile string) (*TTYContD, error) { return newTTY(ttyFile) }

func newTTY(ttyFile string) (_ *TTYContD, err error) {
	defer func() {
		if r := recover(); r != nil {
			// err, e = errors.ParsePanic(string(debug.Stack()))
			err = errors.New(r)
		}
	}()
	f, err := os.OpenFile(ttyFile, os.O_RDWR, 0)
	if err != nil {
		return nil, errors.New(err)
	}
	// this panics on many platforms for whatever reason...
	c, err := console.ConsoleFromFile(f)
	/*var err error
	c := console.Current()*/
	if err != nil {
		return nil, errors.New(err)
	}
	if err := c.SetRaw(); err != nil {
		return nil, errors.New(err)
	}
	return &TTYContD{
		Console:  c,
		fileName: ttyFile,
	}, nil
}

func (t *TTYContD) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.NilReceiver()
	}
	if t.Console == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.Console.Write(b)
}

func (t *TTYContD) Read(p []byte) (n int, err error) {
	if t == nil || t.Console == nil {
		return 0, errors.NilReceiver()
	}
	return t.Console.Read(p)
}
func (t *TTYContD) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *TTYContD) Close() error {
	if t == nil || t.Console == nil {
		return nil
	}
	defer func() { t.Console = nil }()
	errReset := t.Console.Reset()
	errClose := t.Console.Close()
	err := errors.Join(errReset, errClose)
	if err != nil {
		return errors.New(err)
	}
	return nil
}
