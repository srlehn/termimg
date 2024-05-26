//go:build !darwin

package procextra

import (
	"github.com/shirou/gopsutil/v3/process"

	"github.com/srlehn/termimg/internal/consts"
)

func envOfProcPlatform(proc *process.Process) ([]string, error) { return nil, consts.ErrNotImplemented }

// Solaris 5.10
// https://serverfault.com/a/298786
// pargs -e <PID>
