package exc

import (
	"os"
	"os/user"

	"github.com/srlehn/termimg/internal/errors"
)

var systemDirs = []string{
	`/usr/bin/`,
	`/bin/`,
	// likely not in the following
	`/usr/sbin/`,
	`/sbin/`,
}

var (
	// key: rel. path, value: abs. path
	exePaths            = make(map[string]string)
	rootChecked, isRoot bool
)

const ErrRootExecStr = `command execution disabled for root user`

func LookSystemDirs(exe string) (string, error) {
	if len(exe) == 0 {
		return ``, errors.New(`empty executable name`)
	}
	if rootChecked {
		if isRoot {
			return ``, errors.New(ErrRootExecStr)
		}
	} else {
		user, err := user.Current()
		if err != nil {
			return ``, errors.New(err)
		} else {
			if user.Uid == `0` {
				isRoot = true
				rootChecked = true
				return ``, errors.New(ErrRootExecStr)
			}
			rootChecked = true
		}
	}
	if exeAbs, ok := exePaths[exe]; ok && len(exeAbs) > 0 && exeAbs[0] == '/' {
		return exeAbs, nil
	}
	for _, systemDir := range systemDirs {
		exeAbs := systemDir + exe
		fi, err := os.Stat(exeAbs)
		if err != nil || fi == nil {
			continue
		}
		// check if executable for others
		if fi.Mode()&0b001 == 0b001 {
			exePaths[exe] = exeAbs
			return exeAbs, nil
		}
	}
	return ``, errors.Errorf(`executable %q not found in system directories`, exe)
}
