//go:build !windows && !plan9

package bubbleteatty

import (
	"syscall"
	"unsafe"

	"github.com/srlehn/termimg/internal/errors"
)

func (t *TTYBubbleTea) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	if err := errors.NilReceiver(t, t.f); err != nil {
		return 0, 0, 0, 0, err
	}
	// copied from https://github.com/creack/pty - MIT License
	var sz struct {
		Rows uint16
		Cols uint16
		X    uint16
		Y    uint16
	}
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, t.f.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&sz))); e != 0 {
		return 0, 0, 0, 0, errors.New(e)
	}
	return int(sz.Cols), int(sz.Rows), int(sz.X), int(sz.Y), nil
}
