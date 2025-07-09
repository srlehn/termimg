package procextra

import (
	"bufio"
	"bytes"
	"iter"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/process"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/exc"
)

func EnvOfProc(proc *process.Process) ([]string, error) {
	if proc == nil {
		return nil, errors.NilParam(proc)
	}
	/**env, err := proc.Environ()
	if err == nil {
		return env, nil
	}
	// github.com/shirou/gopsutil/internal/common.ErrNotImplementedError
	if err.Error() != `not implemented yet` {
		return nil, err
	}*/
	return envOfProcPlatform(proc)
}

func ParentOfProc(proc *process.Process) (*process.Process, error) {
	if proc == nil {
		return nil, errors.NilParam(proc)
	}
	pproc, err := proc.Parent()
	if err == nil {
		return pproc, nil
	}
	if err.Error() != `could not find parent line` { // TODO
		return nil, err
	}
	psAbs, err := exc.LookSystemDirs(`ps`)
	if err != nil {
		return nil, err
	}
	psOut, err := exec.Command(psAbs, `-o`, `ppid=`, strconv.Itoa(int(proc.Pid))).Output()
	if err != nil {
		return nil, err
	}
	ppidStr := strings.Trim(string(psOut), "\x00\n ")
	ppid, err := strconv.Atoi(ppidStr)
	if err != nil {
		return nil, err
	}
	pproc, err = process.NewProcess(int32(ppid))
	if err != nil {
		return nil, err
	}
	if pproc == nil {
		return nil, errors.New(`received nil parent process`)
	}
	return pproc, err
}

func TTYOfProc(proc *process.Process) (string, error) {
	if proc == nil {
		return ``, errors.NilParam(proc)
	}
	tty, err := proc.Terminal()
	if err == nil {
		return tty, nil
	}
	// github.com/shirou/gopsutil/internal/common.ErrNotImplementedError
	if err.Error() != `not implemented yet` {
		return ``, err
	}
	psAbs, err := exc.LookSystemDirs(`ps`)
	if err != nil {
		return ``, err
	}
	psOut, err := exec.Command(psAbs, `-ewwo`, `pid=,tty=`).Output()
	if err != nil {
		return ``, err
	}
	var pidAndTTYIter iter.Seq2[int, string] = func(yield func(int, string) bool) {
		scanner := bufio.NewScanner(bytes.NewReader(psOut))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			parts := strings.SplitN(line, ` `, 2)
			if len(parts) != 2 {
				// TODO log
				continue
			}
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				// TODO log
				continue
			}
			tty := strings.TrimSpace(parts[1])
			if !yield(pid, tty) {
				return
			}
		}
	}
	pd := int(proc.Pid)
	for pid, tty := range pidAndTTYIter {
		if pid == pd {
			if len(tty) == 0 || strings.HasPrefix(tty, `?`) {
				return ``, nil
				// return ``, errors.New(`no tty`) // TODO
			}
			tty = string(filepath.Separator) + tty
			return tty, nil
		}
	}
	return ``, errors.New(`unable to determine tty`)
}
