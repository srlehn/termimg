package terminals_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/pty"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/tty/gotty"
)

func TestExecMltermQueryDA1(t *testing.T) {
	tn := `mlterm`

	var repl string
	qt := func(pty string, pid uint) error {
		tty, err := gotty.New(pty)
		if err != nil {
			return err
		}
		qu := qdefault.NewQuerier()
		// termCaps := term.QueryTermName("\033[0c", pty)
		// termCaps := term.Query("\033[0c", pty)
		termCaps, err := qu.Query("\033[0c", tty, parser.StopOnAlpha)
		if err != nil {
			return err
		}

		for _, c := range termCaps {
			repl += string(c)
		}
		return nil
	}

	if err := pty.PTYRun(qt, tn); err != nil {
		t.Fatal(err)
	}
	t.Logf("query reply: %q\n", repl)
	caps := capToUintSlice(repl)
	if !slices.Contains(caps, 4) {
		t.Fatal(`could not detect sixel capability of mlterm`)
	}
}

func capToUintSlice(c string) []uint {
	// get numbers from DA1 reply
	capStr := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(c, "\033["), `?`), `c`)
	capsStrSl := strings.Split(capStr, `;`)
	if len(capsStrSl) == 1 {
		return nil
	}
	capsStrSl = capsStrSl[1:] // remove device class
	capsSl := make([]uint, 0, len(capsStrSl))
	for _, cc := range capsStrSl {
		u, err := strconv.ParseUint(cc, 10, 64)
		if err != nil {
			continue
		}
		capsSl = append(capsSl, uint(u))
	}
	return capsSl
}
