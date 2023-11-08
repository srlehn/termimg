package internal

import "runtime"

func DefaultTTYDevice() string {
	var ptyName string
	switch runtime.GOOS {
	case `windows`:
		ptyName = `CON`
	case `darwin`:
		// TODO: does /dev/tty cause issues on macos?
		ptyName = `/dev/stdin` // /dev/tty
	default:
		ptyName = `/dev/tty` // /dev/stdin
	}
	return ptyName
}
func IsDefaultTTY(ptyName string) bool {
	// TODO add stdout?
	if len(ptyName) == 0 {
		return true
	}
	switch runtime.GOOS {
	case `windows`:
		return ptyName == `CON` || ptyName == `CONIN$`
	default:
		return ptyName == `/dev/tty` || ptyName == `/dev/stdin`
	}
}
