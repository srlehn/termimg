package main

import (
	"image"
	_ "image/png"
	"log"

	"github.com/gizak/termui/v3"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal/assets"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/tui/termuiimg"
)

func main() {
	if err := termui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer termui.Close()

	tm, err := termimg.Terminal()
	if err != nil {
		log.Fatalf("failed to initialize termimg: %v", err)
	}
	defer tm.Close()

	img := term.NewImageBytes(assets.SnakePic)
	bounds := image.Rect(10, 8, 50, 30)
	imgWidget, err := termuiimg.NewImage(tm, img, bounds)
	if err != nil {
		log.Fatalf("failed to create image widget: %v", err)
	}
	defer imgWidget.Close()

	termui.Render(imgWidget)

	// BUG: stop termimg from consuming input
	imgWidget.Close()
	tm.Close()

	for e := range termui.PollEvents() {
		if e.Type == termui.KeyboardEvent {
			break
		}
	}
}
