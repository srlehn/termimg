package term

import (
	"io"
)

// TTY ...
// optional methods:
//   - ResizeEvents() (_ <-chan Resolution, closeFunc func() error, _ error)
type TTY interface {
	TTYDevName() string
	// TODO -> io.ReadWriteCloser
	io.Writer                                // Write(p []byte) (n int, err error) // io.Writer
	ReadRune() (r rune, size int, err error) // bufio.Reader.ReadRune() // TODO -> io.Reader
	Close() error                            // io.Closer
}
