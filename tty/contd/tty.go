package contd

import (
	"os"
	"unicode/utf8"

	"github.com/containerd/console"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyContD struct {
	console.Console
	fileName string
}

var _ term.TTY = (*ttyContD)(nil)

func New(ttyFile string) (term.TTY, error) { return newTTY(ttyFile) }

func newTTY(ttyFile string) (_ term.TTY, err error) {
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
	return &ttyContD{
		Console:  c,
		fileName: ttyFile,
	}, nil
}

func (t *ttyContD) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.New(consts.ErrNilReceiver)
	}
	if t.Console == nil {
		return 0, errors.New(`nil tty`)
	}
	return t.Console.Write(b)
}

func (t *ttyContD) ReadRune() (r rune, size int, err error) {
	// TODO store remaining bytes
	r = utf8.RuneError // '\uFFFD'
	if t == nil {
		return r, len(string(r)), errors.New(consts.ErrNilReceiver)
	}
	if t.Console == nil {
		return r, len(string(r)), errors.New(`nil tty`)
	}
	rb := make([]byte, 4)
	var nTotal int
	for i := 0; i < cap(rb); i++ {
		b := make([]byte, 1)
		n, err := t.Console.Read(b)
		nTotal += n
		if err != nil {
			return r, nTotal, errors.New(err)
		}
		rb[i] = b[0]
		if utf8.Valid(rb) {
			break
		} else if i == cap(rb)-1 {
			return r, nTotal, errors.New(err)
		}
	}
	r, _ = utf8.DecodeRune(rb)
	return r, nTotal, err
}

func (t *ttyContD) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *ttyContD) Close() error {
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
