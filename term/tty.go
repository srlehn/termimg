package term

import (
	"io"
)

// TTY ...
// optional methods:
//   - ResizeEvents() (_ <-chan Resolution, closeFunc func() error, _ error)
//   - SizePixel() (cw int, ch int, pw int, ph int, e error)
//   - ReadRune() (r rune, size int, err error)
type TTY interface {
	io.ReadWriteCloser
	TTYDevName() string
}

func (t *Terminal) TTY() TTY {
	if t == nil {
		return nil
	}
	return t.tty
}
