package gotty

import (
	"testing"

	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/query/qdefault"
)

func TestQueryTermNew(t *testing.T) {
	qu := qdefault.NewQuerier()
	defer qu.(interface{ Close() error }).Close()
	ttyM := &ttyMattN{}
	defer ttyM.Close()
	qs := "\033[0c"
	repl, err := qu.Query(qs, ttyM, parser.StopOnAlpha)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: %q\n", "test", repl)
}

// TODO open fresh terminal for test
