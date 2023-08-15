//go:build dev

package mux

import (
	"fmt"
	"log"
	"runtime"

	"github.com/shirou/gopsutil/process"
)

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
