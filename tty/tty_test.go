package tty_test

import (
	"testing"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/bagabastty"
	"github.com/srlehn/termimg/tty/contdtty"
	"github.com/srlehn/termimg/tty/creacktty"
	"github.com/srlehn/termimg/tty/dumbtty"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/tty/pkgterm"
	"github.com/srlehn/termimg/tty/tcelltty"
	"github.com/srlehn/termimg/tty/uroottty"
	// "github.com/srlehn/termimg/tty/bubbleteatty"
)

type ttyProvFunc func(ptyName string) (term.TTY, error)

func wrapTTYProv[T term.TTY, F func(ptyName string) (T, error)](ttyProvFunc F) ttyProvFunc {
	return func(ptyName string) (term.TTY, error) { return ttyProvFunc(ptyName) }
}

// TODO drain input between tests

func TestTTYNewAll(t *testing.T) {
	tests := map[string]ttyProvFunc{
		"gotty": wrapTTYProv(gotty.New),
		// "bubbletea":  wrapTTYProv(bubbleteatty.New),
		"bagabas":    wrapTTYProv(bagabastty.New),
		"containerd": wrapTTYProv(contdtty.New),
		"creack":     wrapTTYProv(creacktty.New),
		"pkgterm":    wrapTTYProv(pkgterm.New),
		"tcell":      wrapTTYProv(tcelltty.New),
		"uroot":      wrapTTYProv(uroottty.New),
		"dumb-tty":   wrapTTYProv(dumbtty.New),
		// "dummy-tty": wrapTTYProv(dummytty.New),
	}
	for name, ttyProv := range tests {
		t.Run(name, func(t *testing.T) {
			testDeviceName(t, ttyProv)
			testQuery(t, ttyProv)
			testSizePixel(t, ttyProv)
		})
		// time.Sleep(1 * time.Second)
	}
}

func testDeviceName(t *testing.T, ttyProv ttyProvFunc) {
	t.Run(`device_name_test`, func(t *testing.T) {
		tty, err := ttyProv(internal.DefaultTTYDevice())
		if err != nil {
			t.Fatal(err)
		}
		defer tty.Close()
		devName := tty.TTYDevName()
		if !internal.IsDefaultTTY(devName) {
			t.Fatalf("tty device \"%s\" not a default tty\n", devName)
		}
		t.Logf("tty device name: %s\n", devName)
		tty.Close()
	})
}

func testQuery(t *testing.T, ttyProv ttyProvFunc) {
	t.Run(`query_test`, func(t *testing.T) {
		qu := qdefault.NewQuerier()
		defer util.TryClose(qu)
		tty, err := ttyProv(internal.DefaultTTYDevice())
		if err != nil {
			t.Fatal(err)
		}
		defer tty.Close()
		qs := "\033[0c"
		repl, err := qu.Query(qs, tty, parser.StopOnAlpha)
		if err != nil {
			t.Fatal(err)
		}
		if repl == qs {
			t.Fatal("reply is query")
		}
		t.Logf("%q -> %q %[2]s\n", qs, repl)
		tty.Close()
		util.TryClose(qu)
	})
}

func testSizePixel(t *testing.T, ttyProv ttyProvFunc) {
	t.Run(`size_pixel_test`, func(t *testing.T) {
		tty, err := ttyProv(internal.DefaultTTYDevice())
		if err != nil {
			t.Fatal(err)
		}
		defer tty.Close()
		szr, ok := tty.(interface {
			term.TTY
			SizePixel() (cw int, ch int, pw int, ph int, e error)
		})
		if !ok {
			return
		}
		cw, ch, pw, ph, err := szr.SizePixel()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("cells: %dx%d, pixels: %dx%d\n", cw, ch, pw, ph)
		if cw < 1 || ch < 1 {
			t.Fatal("terminal cell size is 0")
		}
		/* if pw < 1 || ph < 1 {
			t.Fatal("terminal pixel size is 0")
		} */
		tty.Close()
	})
}

/*
// TODO test optional term.TTY methods
// optional methods:
//   - ResizeEvents() (_ <-chan Resolution, closeFunc func() error, _ error)
//   - SizePixel() (cw int, ch int, pw int, ph int, e error)
//   - ReadRune() (r rune, size int, err error)  // io.RuneReader
*/

// TODO open fresh terminal for test
