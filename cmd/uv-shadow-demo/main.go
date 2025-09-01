//go:build dev

package main

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/sixel"
	_ "github.com/srlehn/termimg/drawers/x11"
	"github.com/srlehn/termimg/internal/assets"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/tty/uvtty"
)

func main() {
	// Get drawer type from command line argument
	drawerType := "sixel" // default
	if len(os.Args) > 1 {
		drawerType = os.Args[1]
	}
	if len(os.Args) < 2 || (os.Args[1] != `sixel` && os.Args[1] != `x11`) {
		fmt.Fprintf(os.Stderr, "Usage: %s <sixel|x11>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	imageArea := uv.Rect(5, 3, 15, 7)
	shadowArea := uv.Rect(imageArea.Min.X+3, imageArea.Min.Y+2, 8, 7)

	// Create termimg terminal with uvtty using default config
	tmg, err := term.NewTerminal(
		termimg.DefaultConfig,
		// reuse input! this opens and manages a uv.Terminal internally
		term.SetTTYProvider(uvtty.New, false),
		term.SetLogFile("./termimg-uv-debug.log", true),
	)
	if err != nil {
		log.Fatalf("Failed to create termimg terminal: %v", err)
	}
	defer tmg.Close()

	// extract managed ultraviolet Terminal
	tmu := tmg.TTY().(*uvtty.TTYUV).UVTerminal()

	// Create termimg drawable that will use specified drawer to draw directly to the terminal window
	termimgDrawable := &TermimgDrawable{
		timg:       termimg.NewImageBytes(assets.SnakePic),
		tmg:        tmg,
		bounds:     imageArea,
		drawerType: drawerType,
	}
	// Create UV text box drawable that should shadow the corner of the image
	var textBoxDrawable TextBoxDrawable = '#'

	// clean screen
	_ = tmg.Scroll(0)
	tmu.Clear()
	tmu.MoveTo(0, 0)
	_ = tmu.Display()

	// Draw the termimg image first (using X11/sixel directly on the terminal window)
	termimgDrawable.Draw(tmu, imageArea)
	time.Sleep(1 * time.Second)

	// Draw the UV text box that should shadow part of the image
	textBoxDrawable.Draw(tmu, shadowArea)
	_ = tmu.Display() // Display the result
	time.Sleep(3 * time.Second)

	fmt.Println("\r\n\ndrawer:", drawerType)
}

// TermimgDrawable wraps a termimg image as a UV drawable
type TermimgDrawable struct {
	timg       *term.Image
	bounds     image.Rectangle
	tmg        *term.Terminal
	drawerType string
}

// Draw implements the UV Drawable interface
func (td *TermimgDrawable) Draw(scr uv.Screen, area uv.Rectangle) {
	// Find the specified drawer type
	var drawer term.Drawer
	for _, dr := range td.tmg.Drawers() {
		if dr.Name() == td.drawerType {
			drawer = dr
			break
		}
	}
	if drawer == nil {
		log.Fatalf("%s drawer not available", td.drawerType)
	}

	// Draw the image using termimg's X11/sixel functionality
	if err := drawer.Draw(td.timg, td.bounds, td.tmg); err != nil {
		log.Fatalf("Failed to draw image: %v", err)
	}

	_ = td.tmg.SetCursor(0, 0)
	/*
		tmu := td.tmg.TTY().(*uvtty.TTYUV).UVTerminal()
		tmu.MoveTo(0, 0)
		tmu.Display()
	*/
}

// TextBoxDrawable creates a simple text box filled with a character
type TextBoxDrawable rune

// Draw implements the UV Drawable interface
func (tb *TextBoxDrawable) Draw(scr uv.Screen, area uv.Rectangle) {
	cell := &uv.Cell{Content: string(*tb), Width: 1}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			scr.SetCell(x, y, cell)
		}
	}
}
