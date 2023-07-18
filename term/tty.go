package term

import (
	"io"

	"github.com/srlehn/termimg/internal"
)

// TTY ...
type TTY interface {
	TTYDevName() string
	// TODO -> io.ReadWriteCloser
	io.Writer                                // Write(p []byte) (n int, err error) // io.Writer
	ReadRune() (r rune, size int, err error) // bufio.Reader.ReadRune() // TODO -> io.Reader
	Close() error                            // io.Closer
}

var _ TTY = (internal.TTY)(nil)
