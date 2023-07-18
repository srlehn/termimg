package internal

import (
	"image"
	"io"
)

// Arger is for passing extra arguments to the default terminal execution call
// e.g. disabling daemonizing (domterm)
type Arger interface{ Args() []string }

// TTY ...
type TTY interface {
	// TODO rm duplicate
	TTYDevName() string
	// TODO -> io.ReadWriteCloser
	io.Writer                                // Write(p []byte) (n int, err error) // io.Writer
	ReadRune() (r rune, size int, err error) // bufio.Reader.ReadRune() // TODO -> io.Reader
	Close() error                            // io.Closer
}

type ImageEncoder interface {
	Encode(w io.Writer, img image.Image, fileExt string) error
}
