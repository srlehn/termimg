package gotty_test

import (
	"testing"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/tty/gotty"
)

func TestQueryTermNew(t *testing.T) {
	qu := qdefault.NewQuerier()
	defer qu.(interface{ Close() error }).Close()
	ttyM, err := gotty.New(internal.DefaultTTYDevice())
	if err != nil {
		t.Fatal(err)
	}
	defer ttyM.Close()
	qs := "\033[0c"
	repl, err := qu.Query(qs, ttyM, parser.StopOnAlpha)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: %q\n", "test", repl)
}

// TODO open fresh terminal for test
