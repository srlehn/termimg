//go:build linux

package linux

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/srlehn/termimg/internal/errors"
)

func KDGetMode(fd uintptr) (mode KDMode, isLinuxConsole bool, _ error) {
	const KDGETMODE uintptr = 0x4b3b
	m, err := unix.IoctlGetInt(int(fd), uint(KDGETMODE))
	mode = KDMode(m)
	if err == nil {
		return mode, true, nil
	}
	if errors.Is(err, unix.ENOTTY) {
		return -1, false, nil
	}
	return -1, false, err
}

func (k *KDMode) String() string {
	if k == nil {
		return `<nil>`
	}
	switch *k {
	case 0x0:
		return `KD_TEXT`
	case 0x1:
		return `KD_GRAPHICS`
	case 0x2:
		return `KD_TEXT0`
	case 0x3:
		return `KD_TEXT1`
	}
	if *k > 0 {
		return fmt.Sprintf(`0x%x`, k)
	} else {
		return fmt.Sprintf(`-0x%x`, k)
	}
}
