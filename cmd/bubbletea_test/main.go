//go:build dev

package main

import (
	"context"
	"image"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal/assets"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/tui/bubbleteaimg"

	// "github.com/srlehn/termimg/tty/contdtty"
	"github.com/srlehn/termimg/tty/bubbleteatty"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var testImg []byte = assets.SnakePic

func main() {
	f, _ := os.Create("log_bubbletea.log")
	defer f.Close()
	log.Default().SetOutput(f)

	// call os.Exit() after m and its deferred close functions
	err := m()
	if err != nil {
		log.Fatalln("err:", err)
		if es, ok := err.(interface{ ErrorStack() string }); ok {
			log.Fatalln(es.ErrorStack())
		}
	}
}

func m() error {
	// create termimg terminal
	opts := []term.Option{
		term.SetLogFile(`log_termimg.log`, true),
		termimg.DefaultConfig,
		term.SetTTYProvider(bubbleteatty.New, true),
	}
	tm, err := term.NewTerminal(opts...)
	if err != nil {
		return err
	}
	defer tm.Close()

	bounds := image.Rect(10, 10, 60, 35)
	text := `Lorem ipsum dolor sit amet, consectetur adipisici elit, sed eiusmod tempor incidunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquid ex ea commodi consequat. Quis aute iure reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint obcaecat cupiditat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).Margin(2).Padding(0, 1).
		Width(bounds.Dx()).Height(bounds.Dy())
	mdl, err := newModel(&style, bounds, text)
	if err != nil {
		return err
	}

	// finish "tty" setup
	teaOpts := []tea.ProgramOption{tea.WithAltScreen()}
	tty, err := bubbleteatty.TTYOf(tm)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	prog, err := tty.SetProgram(ctx, mdl, teaOpts...)
	if err != nil {
		return err
	}
	defer prog.Quit()

	// image setup
	if err := mdl.img.Setup(tm, termimg.NewImageBytes(testImg)); err != nil {
		return err
	}

	//go func() {
	if _, err = prog.Run(); logx.IsErr(err, tm, slog.LevelError) {
		return err
	}
	//}()

	// time.Sleep(5 * time.Second)

	return nil
}

var _ tea.Model = (*model)(nil)

type model struct {
	img       *bubbleteaimg.Image
	styleText lipgloss.Style
	text      string
}

func newModel(style *lipgloss.Style, bounds image.Rectangle, text string) (*model, error) {
	styleText := style.Copy().Width(bounds.Dx() / 2).Height(bounds.Dy())
	styleImg := style.Copy().Width(bounds.Dx()).Height(bounds.Dy())
	img, err := bubbleteaimg.NewImage(&styleImg)
	if err != nil {
		return nil, err
	}
	mdl := &model{
		styleText: styleText,
		img:       img,
		text:      text,
	}
	return mdl, nil
}

func (m *model) Init() tea.Cmd {
	return func() tea.Msg {
		if m != nil && m.img != nil {
			return m.img.Init()
		}
		return nil
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			log.Printf("tea.Quit called\n")
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

	cmd := func() tea.Msg { return msg }
	return m, cmd
}

func (m *model) View() string {
	if m == nil {
		return `<nil bubbletea.Model>`
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, m.styleText.Render(m.text), m.img.View())
}
