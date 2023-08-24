//go:build dev

package term

import (
	"runtime"

	"github.com/srlehn/termimg/internal/propkeys"
)

func (t *Terminal) Exe() string {
	if t == nil || t.proprietor == nil {
		return ``
	}
	exe, _ := t.Property(propkeys.Executable)
	var suffix string
	if runtime.GOOS == `windows` {
		suffix = `.exe`
	}
	if len(exe) > 0 {
		return exe + suffix
	}
	return t.Name() + suffix
}

func XTGetTCap(tm *Terminal, tcap string) (string, error) {
	return xtGetTCap(tcap, tm.querier, tm.tty, tm.proprietor, tm.proprietor)
}
