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
	return len(ptyName) == 0 || ptyName == `/dev/tty` || ptyName == `/dev/stdin` || ptyName == `CON` // TODO
}
