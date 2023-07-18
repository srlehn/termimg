package advanced

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	"github.com/shirou/gopsutil/process"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/exc"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/mux"
)

func GetEnv(ptyName string) (envInner environ.Proprietor, passages mux.Muxers, e error) {
	// Steps:
	// 1.) find process providing tty
	// 2.) walk process tree until terminal, note muxers, etc
	// 3.) diff provided env from inherited env
	var proc, procTerm *process.Process
	var ttyInner string
	var errRet, err error
	isDefaultTTY := func(ptyName string) bool {
		return ptyName == `/dev/tty` || ptyName == `/dev/stdin` || ptyName == `CON` // TODO
	}
	if isDefaultTTY(ptyName) { // TODO
		pid := os.Getppid() // probably a shell
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			errRet = errors.New(err)
			goto end
		}
		proc = p
	} else {
		// TODO: different terminal
		// return errNotImplemented
		//
		// list terminal, shell, pts
		// ps -o ppid=,pid=,cmd= -t /dev/pts/4
		//
		// Linux: /proc/tty/drivers

		// TODO: find alternative
		// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/ps.html
		// psOut, err := exec.Command(`ps`, `-o`, `ppid=,pid=,cmd=`, `-t`, ptyName).Output()
		psAbs, err := exc.LookSystemDirs(`ps`)
		if err != nil {
			errRet = err
			goto end
		}
		psOut, err := exec.Command(psAbs, `-o`, `pid=`, `-t`, ptyName).Output()
		if err != nil {
			errRet = errors.New(err)
			goto end
		}
		pidMap := make(map[string]int)
		for _, procStr := range strings.Split(strings.TrimSpace(string(psOut)), "\n") {
			ppidStr := strings.Split(procStr, ` `)[0]
			if len(ppidStr) == 0 {
				errRet = errors.New(`unable to find process`)
				goto end
			}
			if _, ok := pidMap[ppidStr]; ok {
				continue
			}
			ppidInt, err := strconv.Atoi(ppidStr)
			if err != nil {
				errRet = errors.New(err)
				goto end
			}
			ppid := int32(ppidInt)
			for {
				pProc, err := process.NewProcess(ppid)
				if err != nil {
					break
				}
				ppid, err = pProc.Ppid()
				if err != nil || ppid < 1 {
					break
				}
				if _, ok := pidMap[strconv.Itoa(int(ppid))]; ok {
					break
				}
				pidMap[ppidStr]++
			}
		}
		var termPIDStr string
		var cntLowest int
		for pidStr, cnt := range pidMap {
			if cntLowest == 0 || cnt < cntLowest {
				cntLowest = cnt
				termPIDStr = pidStr
			}
		}
		if cntLowest == 0 {
			errRet = errors.New(`unable to determine terminal pid`)
			goto end
		}
		termPID, err := strconv.Atoi(termPIDStr)
		if err != nil {
			errRet = errors.New(err)
			goto end
		}
		proc, err = process.NewProcess(int32(termPID))
		if err != nil {
			errRet = errors.New(err)
			goto end
		}
	}
	if proc == nil {
		errRet = errors.New(`nil proc`)
		goto end
	}
	procTerm, ttyInner, envInner, passages, err = mux.FindTerminalProcess(proc.Pid)
	if err != nil {
		errRet = err
		goto end
	}

end:
	if errRet != nil {
		if isDefaultTTY(ptyName) {
			// fallback
			return environ.EnvToProprietor(os.Environ()), nil, nil
		}
		return nil, nil, errRet
	}

	if passages != nil && passages.IsRemote() {
		envInner.SetProperty(propkeys.IsRemote, ``)
	}
	if procTerm != nil {
		envInner.SetProperty(propkeys.TerminalPID, strconv.Itoa(int(procTerm.Pid)))
	}
	if len(ttyInner) > 0 {
		if runtime.GOOS != `windows` {
			ttyInner = `/dev` + ttyInner
		}
		envInner.SetProperty(propkeys.TerminalTTY, ttyInner)
	}

	return envInner, passages, nil
}
