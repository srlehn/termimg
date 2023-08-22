package consts

import (
	"errors"
)

var (
	ErrNotImplemented          = errors.New(`not implemented`)
	ErrNilReceiver             = errors.New(`nil receiver`)
	ErrNilParam                = errors.New(`nil parameter`)
	ErrNilImage                = errors.New(`nil image`)
	ErrPlatformNotSupported    = errors.New(`platform not supported`)
	ErrTimeoutInterval         = errors.New(`terminal querying timeout (interval)`)
	ErrXTGetTCapInvalidRequest = errors.New(`invalid XTGETTCAP request`)
)

const (
	DrawerGenericName = `generic`
	TermGenericName   = `generic`

	LibraryName = `termimg`

	CheckTermPassed = `passed`
	CheckTermFailed = `failed`
	CheckTermDummy  = `dummy` // promoted dummy core method
)
