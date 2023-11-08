//go:build dev

package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"log/slog"
	"os"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/assets"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"

	// "github.com/srlehn/termimg/tty/contdtty"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		// term.SetTTYProvider(contdtty.New, false), // TODO use contdtty instead of gotty
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

	bounds := image.Rect(10, 10, 60, 35)
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		Margin(2).
		Padding(0, 1)
	mdl, _ := newModel(termimg.NewImageBytes(assets.SnakePic), bounds, tm, &style)
	teaOpts := []tea.ProgramOption{
		tea.WithContext(context.TODO()),
		tea.WithAltScreen(),
		tea.WithInput(tm.TTY()),
		tea.WithOutput(tm.TTY()),
	}
	prog := tea.NewProgram(mdl, teaOpts...)
	f, _ := os.Create("logBubbleTest.txt")
	defer f.Close()
	log.Default().SetOutput(f)
	if _, err := prog.Run(); err != nil {
		return errors.New(err)
	}
	return nil
}

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Quit key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

var _ tea.Model = (*model)(nil)

type model struct {
	style lipgloss.Style
	*term.Canvas
	term   *term.Terminal
	bounds image.Rectangle
	keys   keyMap
	width  uint
	height uint
}

func newModel(img image.Image, bounds image.Rectangle, tm *term.Terminal, style *lipgloss.Style) (*model, error) {
	canvas, err := tm.NewCanvas(bounds)
	if err != nil {
		return nil, err
	}
	if logx.IsErr(canvas.SetImage(term.NewImage(img)), tm, slog.LevelError) {
		return nil, err
	}
	mdl := &model{
		style:  style.Width(bounds.Dx()).Height(bounds.Dy()),
		Canvas: canvas,
		term:   tm,
		bounds: bounds,
		keys:   keys,
	}
	return mdl, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("%T %+#[1]v\n", msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate
		// its view as needed.
		m.height = uint(msg.Height)
		m.width = uint(msg.Width)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			log.Printf("tea.Quit called\n")
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	log.Println("View called")
	m.Update(tea.MouseMsg{})
	s := fmt.Sprintf("%+#v", m.style)
	/*s := strings.Repeat(
		strings.Repeat(" ", int(m.width-uint(m.style.GetHorizontalFrameSize())))+"\n",
		int(m.height-uint(m.style.GetVerticalFrameSize())),
	)*/
	_ = s
	x, y, _ := m.term.Cursor()
	_ = m.Canvas.Draw(nil)
	_ = m.term.SetCursor(x, y)
	ret := m.style.Render(fmt.Sprintf("%dx%d\n", x, y))
	log.Printf("%q\n", ret)
	return ret
}
