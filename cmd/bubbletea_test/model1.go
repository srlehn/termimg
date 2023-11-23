//go:build dev

package main

import (
	"image"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/bubbleteatty"
	"github.com/srlehn/termimg/tui/bubbleteaimg"
)

func newTestModel1() *model1 {
	bounds := image.Rect(10, 10, 60, 30)
	text := `Lorem ipsum dolor sit amet, consectetur adipisici elit, sed eiusmod tempor incidunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquid ex ea commodi consequat. Quis aute iure reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint obcaecat cupiditat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`
	style := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder(), true).Margin(2).Padding(0, 1).
		Width(bounds.Dx()).Height(bounds.Dy())
	return newModel1(&style, bounds, text)
}

var _ tea.Model = (*model1)(nil)

type model1 struct {
	img       *bubbleteaimg.Image
	styleText lipgloss.Style
	styleImg  lipgloss.Style
	text      string
}

func newModel1(style *lipgloss.Style, bounds image.Rectangle, text string) *model1 {
	styleText := style.Copy().Width(bounds.Dx() / 2).Height(bounds.Dy())
	styleImg := style.Copy().Width(bounds.Dx()).Height(bounds.Dy())
	mdl := &model1{
		styleText: styleText,
		styleImg:  styleImg,
		text:      text,
	}
	return mdl
}

func (m *model1) Setup(tm *term.Terminal) error {
	if err := errors.NilReceiver(m); err != nil {
		return err
	}
	if err := errors.NilParam(tm); err != nil {
		return err
	}
	img, err := bubbleteaimg.NewImage(tm, termimg.NewImageBytes(testImg), &m.styleImg)
	if err != nil {
		return err
	}
	m.img = img

	tty, ok := tm.TTY().(*bubbleteatty.TTYBubbleTea)
	if !ok || tty == nil {
		log.Println(`unsupported tty`)
		return errors.New(`unsupported tty`)
	}
	renderer := tty.LipGlossRenderer()
	log.Printf("REND %+#v\n", renderer)
	log.Printf("REND color profile %v\n", renderer.ColorProfile())
	log.Printf("REND output %+#v\n", renderer.Output())
	renderer.SetColorProfile(termenv.ANSI256)
	// lipgloss.SetDefaultRenderer(renderer)
	m.styleText = m.styleText.
		Copy().
		Renderer(renderer).
		BorderForeground(lipgloss.Color(`32`)) // 63, 228
		// BorderBackground(lipgloss.Color(`34`))  // 63, 228
	return nil
}

func (m *model1) Init() tea.Cmd {
	return func() tea.Msg {
		if m != nil && m.img != nil {
			return m.img.Init()
		}
		return nil
	}
}

func (m *model1) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	default:
		log.Printf("tea msg: %T %+#[1]v\n", msg)
		if m != nil && m.img != nil {
			mImg, cmd := m.img.Update(msg)
			m.img = mImg.(*bubbleteaimg.Image)
			return m, cmd
		}
	}

	// cmd := func() tea.Msg { return msg }
	// return m, cmd
	return m, nil
}

func (m *model1) View() string {
	if m == nil || m.img == nil {
		return `<nil bubbletea.Model>`
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, m.styleText.Render(m.text), m.img.View())
}
