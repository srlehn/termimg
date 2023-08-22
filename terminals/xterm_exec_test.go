package terminals_test

import (
	"testing"

	"golang.org/x/exp/slices"

	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/pty"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/tty/gotty"
)

func TestExecXTermQueryDA1(t *testing.T) {
	tn := `xterm`

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
