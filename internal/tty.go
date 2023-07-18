package internal

import "runtime"

func DefaultTTYDevice() string {
	var ptyName string
	if runtime.GOOS == `windows` {
		ptyName = `CON`
	} else {
		// TODO: does /dev/tty cause issues on macos?
		ptyName = `/dev/stdin` // /dev/tty
	}
	return ptyName
}
