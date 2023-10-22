package term

import (
	"io"
)

// TTY ...
// optional methods:
//   - ResizeEvents() (_ <-chan Resolution, closeFunc func() error, _ error)
//   - ReadRune() (r rune, size int, err error)
type TTY interface {
	TTYDevName() string
	io.ReadWriteCloser
}
