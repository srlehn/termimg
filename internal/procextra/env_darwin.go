//go:build darwin

package procextra

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/process"

	"github.com/srlehn/termimg/internal/exc"
)

func envOfProcPlatform(proc *process.Process) ([]string, error) {
	psAbs, err := exc.LookSystemDirs(`ps`)
	if err != nil {
		return nil, err
	}
	pidStr := strconv.Itoa(int(proc.Pid))
	// command with environ
	psOutEnv, err := exec.Command(psAbs, `-Eww`, `-o`, `command=`, pidStr).Output()
	if err != nil {
		return nil, err
	}
	// command without environ
	psOutNoEnv, err := exec.Command(psAbs, `-ww`, `-o`, `command=`, pidStr).Output()
	if err != nil {
		return nil, err
	}
	// strip command
	envStr := strings.TrimSpace(
		strings.TrimPrefix(
			strings.Trim(string(psOutEnv), "\x00"),
			strings.Trim(string(psOutNoEnv), "\x00\n "),
		),
	)
	envParts := strings.Split(envStr, ` `)

	var envPre [][]string
	var endOfLastPart int = -1
Outer:
	for i, envPart := range envParts {
		eq := strings.SplitN(envPart, `=`, 2)
		if len(eq) != 2 || len(eq[0]) == 0 {
			continue
		}
		for _, r := range eq[0] {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || (r == '_') {
				continue
			}
			continue Outer
		}
		// we have verified that this part starts a new env var
		if endOfLastPart >= 0 {
			envPre = append(envPre, envParts[endOfLastPart:i])
		}
		endOfLastPart = i
	}
	var env []string
	for _, ep := range envPre {
		env = append(env, strings.Join(ep, ` `))
	}
	return env, nil
}
