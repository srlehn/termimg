package keyfile

import (
	"errors"
)

// These errors can be returned if there is a problem processing a
// keyfile's contents.
var (
	ErrInvalid               = errors.New("invalid keyfile format")
	ErrBadEscapeSequence     = errors.New("bad escape sequence")
	ErrUnexpectedEndOfString = errors.New("unexpected end of string")
)
