package main

import (
	"image"
	_ "image/png"
	"log"
	"log/slog"

	"github.com/gdamore/tcell/v2"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/assets"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/tty/gotty"

	// "github.com/srlehn/termimg/tty/tcelltty"
	"github.com/srlehn/termimg/tui/tcellimg"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func main() {
	// call os.Exit() after m and its deferred close functions
	if err := m(); err != nil {
		if es, ok := err.(interface{ ErrorStack() string }); ok {
			log.Fatalln(es.ErrorStack())
		}
		log.Fatalln(err)
	}
}

func m() error {
	qu := qdefault.NewQuerier()
	opts := []term.Option{
		// termimg.DefaultConfig,
		term.SetPTYName(internal.DefaultTTYDevice()),
		term.SetTTYProvider(gotty.New, false),
		// term.SetTTYProvider(tcelltty.New, false),
		term.SetQuerier(qu, true),
		term.SetWindowProvider(wm.SetImpl(wmimpl.Impl()), true),
		term.SetResizer(&rdefault.Resizer{}),
		term.SetLogFile(`log.txt`, true),
	}
	tm, err := term.NewTerminal(opts...)
	if err != nil {
		return err
	}
	defer tm.Close()

	var scr tcell.Screen
	if ttyTCell, ok := tm.TTY().(interface{ TCellScreen() (tcell.Screen, error) }); ok {
		logx.Info("using tcell.Tty", tm)
		s, err := ttyTCell.TCellScreen()
		if !logx.IsErr(err, tm, slog.LevelInfo) {
			scr = s
		}
	}
	if scr == nil {
		s, err := tcell.NewScreen()
		if err != nil {
			return err
		}
		scr = s
	}
	if err := scr.Init(); err != nil {
		return err
	}
	defer scr.Fini()

	x, y := 10, 10
	w, h := 25, 10
	bounds := image.Rect(x, y, x+w, y+h)

	// mark the area
	scr.SetContent(x-1, y-1, '#', nil, tcell.StyleDefault)
	scr.SetContent(x+w, y-1, '#', nil, tcell.StyleDefault)
	scr.SetContent(x-1, y+h, '#', nil, tcell.StyleDefault)
	scr.SetContent(x+w, y+h, '#', nil, tcell.StyleDefault)
	scr.Sync()

	img, err := tcellimg.NewImage(termimg.NewImageBytes(assets.SnakePic), bounds, tm, scr)
	if err != nil {
		return err
	}
	scr.Clear()

	img.Draw()

	_ = qu.(interface{ Close() error }).Close() // stop stealing input from tcell
outer:
	for {
		ev := scr.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEnter: // break on Enter key event
				break outer
			}
		}
	}

	return nil
}
