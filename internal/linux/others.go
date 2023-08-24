//go:build !linux

package linux

import (
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
)

func KDGetMode(fd uintptr) (mode KDMode, isLinuxConsole bool, _ error) {
	return -1, false, errors.New(consts.ErrPlatformNotSupported)
}
