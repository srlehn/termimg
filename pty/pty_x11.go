//go:build unix && !windows && !android && !darwin && !js

package pty

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jezek/xgb/xproto"
	"github.com/shirou/gopsutil/process"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func ptyRun(termCmd []string, f PTYRunFunc) (errRet error) {
	// TODO fix domterm (orphans)
	if len(termCmd) == 0 {
		return errors.New(`no command`)
	}
	cmd := exec.Command(termCmd[0], termCmd[1:]...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	cmd.Env = []string{
		`DISPLAY=` + os.Getenv(`DISPLAY`),
		`XAUTHORITY=` + os.Getenv(`XAUTHORITY`),
		`SHELL=` + os.Getenv(`SHELL`),
		`HOME=` + os.Getenv(`HOME`), // required by mlterm
		`PATH=` + os.Getenv(`PATH`), // required by domterm
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process == nil {
		return errors.New(`nil process`)
	}
	defer func() {
		// TODO multiple domterm instances
		if err := cmd.Process.Kill(); err != nil && errRet == nil {
			errRet = err
		}
	}()

	proc, err := process.NewProcess(int32(cmd.Process.Pid))
	if err != nil {
		return err
	}

	wm.SetImpl(wmimpl.Impl())
	conn, err := wm.NewConn(environ.EnvToProprietor(cmd.Env))
	if err != nil {
		return err
	}
	windowsStart, err := conn.Windows()
	if err != nil {
		return err
	}
	mapWindowsStart := make(map[xproto.Window]wm.Window)
	for _, window := range windowsStart {
		if window == nil {
			continue
		}
		mapWindowsStart[xproto.Window(window.WindowID())] = window
	}
	var (
		windowsNew                     []wm.Window
		newWindowAppeared              bool
		timeFirstNewWindow             time.Time
		tickDurationCheckProcChildren  = 25 * time.Millisecond
		tickDurationCheckNewWindows    = 330 * time.Millisecond
		waitingTimeAfterFirstNewWindow = 3000 * time.Millisecond
		timeoutWaitingShell            = 8000 * time.Millisecond
	)

	// wait for the start of the shell of the pseudo terminal
	ticker := time.NewTicker(tickDurationCheckProcChildren)
	defer ticker.Stop()
	tickerWindows := time.NewTicker(tickDurationCheckNewWindows)
	defer tickerWindows.Stop()
	var children []*process.Process
	var procSh *process.Process

waitForShell:
	for {
		select {
		case <-ticker.C:
			children, _ = proc.Children()
			// printProcWithChildren(proc)
			if len(children) == 0 {
				if newWindowAppeared &&
					time.Since(timeFirstNewWindow) > waitingTimeAfterFirstNewWindow {
					// probably orphaned process
					break waitForShell
				}
			} else {
				ps, err := findShellProc(proc)
				if err != nil {
					return err
				}
				if ps != nil {
					procSh = ps
					break waitForShell
				}
			}
		case <-tickerWindows.C:
			windows, err := conn.Windows()
			if err != nil {
				return err
			}
			for _, window := range windows {
				if window == nil {
					continue
				}
				if _, exists := mapWindowsStart[xproto.Window(window.WindowID())]; exists {
					continue
				}
				if len(windowsNew) == 0 {
					newWindowAppeared = true
					timeFirstNewWindow = time.Now()
				}
				windowsNew = append(windowsNew, window)
				mapWindowsStart[xproto.Window(window.WindowID())] = window
			}
		case <-time.After(timeoutWaitingShell):
			return errors.New(`timeout while waiting for shell`)
		}
	}

	if procSh == nil {
		// process probably orphaned after double fork, etc
		// fallback to window change detection
		if len(windowsNew) == 0 {
			return errors.New(`unable to detect spawned window`)
		}
		for _, w := range windowsNew {
			pr, _ := process.NewProcess(int32(w.WindowPID()))
			printProc(pr)
		}
		window := windowsNew[0]
		procWin, err := process.NewProcess(int32(window.WindowPID()))
		if err != nil {
			return err
		}
		ps, err := findShellProc(procWin)
		if err != nil {
			return err
		}
		if ps == nil {
			return errors.New(`found no shell contained in newly appeared window`)
		}
		procSh = ps
	}

	// printProc(procSh)

	pty, err := procSh.Terminal()
	if err != nil {
		return err
	}
	if len(pty) == 0 {
		return errors.New(`found no tty for assumed shell process`)
	}
	ptyStr := `/dev/` + strings.TrimPrefix(string(pty), `/`) // TODO

	// execute passed function on the pty of the pseudo terminal.
	if err := f(ptyStr, uint(cmd.Process.Pid)); err != nil {
		errRet = err
	}

	return
}

func findShellProc(pr *process.Process) (*process.Process, error) {
	if pr == nil {
		return nil, errors.New(consts.ErrNilParam)
	}
	children, err := pr.Children()
	if err != nil {
		return nil, err
	}
	prTerm, err := pr.Terminal()
	if err != nil {
		return nil, err
	}
	sh := os.Getenv(`SHELL`)
	for _, child := range children {
		if isRunning, err := child.IsRunning(); err != nil {
			continue
		} else if !isRunning {
			continue
		}
		exe, err := child.Exe()
		if err != nil {
			// defunct?
			continue
		}
		childTerm, err := child.Terminal()
		if err != nil {
			return nil, err
		}
		if len(childTerm) > 0 && childTerm != prTerm {
			return child, nil
		}
		cmdAbs, err := exec.LookPath(exe)
		if err != nil {
			return nil, err
		}
		// might fail if containerised, etc
		if cmdAbs == sh {
			return child, nil
		}
	}
	return nil, nil
}

func printProc(pr *process.Process) {
	if pr == nil {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	name, _ := pr.Name()
	term, _ := pr.Terminal()
	ppid, _ := pr.Ppid()
	fmt.Printf("%s:%d: pid:%d ppid:%d %q %q\n", file, line, pr.Pid, ppid, name, term)
}

var _ = printProcWithChildren

func printProcWithChildren(pr *process.Process) {
	if pr == nil {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	name, _ := pr.Name()
	term, _ := pr.Terminal()
	ppid, _ := pr.Ppid()
	fmt.Printf("%s:%d: pid:%d ppid:%d %q %q\n", file, line, pr.Pid, ppid, name, term)

	children, err := pr.Children()
	if err != nil {
		log.Println(err)
	}
	for _, child := range children {
		name, _ := child.Name()
		term, _ := child.Terminal()
		ppid, _ := child.Ppid()
		fmt.Printf("  %s:%d: pid:%d ppid:%d %q %q\n", file, line, child.Pid, ppid, name, term)
	}
}
