package internal

import (
	"github.com/go-errors/errors"
)

var (
	ErrNotImplemented          = errors.New(`not implemented`)
	ErrNilReceiver             = errors.New(`nil receiver`)
	ErrNilParam                = errors.New(`nil parameter`)
	ErrNilImage                = errors.New(`nil image`)
	ErrPlatformNotSupported    = errors.New(`platform not supported`)
	ErrTimeoutInterval         = errors.New(`terminal querying timeout (interval)`)
	ErrXTGetTCapInvalidRequest = errors.New(`invalid XTGETTCAP request`)

	DrawerGenericName = `generic`
	TermGenericName   = `generic`

	LibraryName = `termimg`
)
