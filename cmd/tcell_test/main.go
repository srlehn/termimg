package main

import (
	_ "embed"
	"image"
	_ "image/png"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/tui/tcellimg"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

//go:embed snake.png
var imgBytes []byte

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
	wm.SetImpl(wmimpl.Impl())
	qu := qdefault.NewQuerier()
	opts := []term.Option{
		termimg.DefaultConfig,
		term.SetPTYName(internal.DefaultTTYDevice()),
		// term.SetTTYProvider(tcelltty.New, false),
		// term.SetTTYProvider(tcelltty.New, true),
		term.SetTTYProvider(gotty.New, false),
		term.SetQuerier(qu, true),
		term.SetWindowProvider(wm.NewWindow, true),
		term.SetResizer(&rdefault.Resizer{}),
	}
	tm, err := term.NewTerminal(opts...)
	if err != nil {
		return err
	}
	defer tm.Close()

	scr, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := scr.Init(); err != nil {
		return err
	}
	defer scr.Fini()

	x, y := 10, 10
	w, h := 25, 10
	bounds := image.Rect(x, y, x+w, y+h)
	img, err := tcellimg.NewImage(termimg.NewImageBytes(imgBytes), bounds, tm, scr)
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
