//go:build dev

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/bubbleteatty"
)

func newTestModel2() *model2 {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".png", ".jpg"}
	fp.CurrentDirectory = filepath.Join(util.Must2(os.UserHomeDir()), `Pictures`)
	fp.Height = 20
	// fp.AutoHeight = true
	fp.FileAllowed = true
	// fp.DirAllowed = true
	m := &model2{
		filepicker: fp,
	}
	return m
}

type model2 struct {
	filepicker   filepicker.Model
	selectedFile string
	quitting     bool
	err          error
	t            string
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m *model2) Setup(tm *term.Terminal) error {
	if err := errors.NilReceiver(m); err != nil {
		return err
	}
	tty, ok := tm.TTY().(*bubbleteatty.TTYBubbleTea)
	if !ok || tty == nil {
		return errors.New(`unsupported tty`)
	}
	renderer := tty.LipGlossRenderer()
	log.Println("REND", renderer)
	m.filepicker.Styles.DisabledCursor = m.filepicker.Styles.DisabledCursor.Renderer(renderer)
	m.filepicker.Styles.Selected = m.filepicker.Styles.Selected.Renderer(renderer)

	for _, d := range tm.Drawers() {
		m.t += d.Name() + `,`
	}
	m.t = strings.TrimSuffix(m.t, `,`)

	return nil
}

func (m *model2) Init() tea.Cmd {
	// return nil
	return m.filepicker.Init()
}

func (m *model2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Printf("%+#v\n", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		log.Printf("%q\t%q\t%t\n", string(msg.Runes), msg.String(), msg.Alt)
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.err = nil
	default:
		tpStr := fmt.Sprintf(`%T`, msg)
		if tpStr == `filepicker.readDirMsg` {
			// log.Printf("MSG: %+#v\n", msg)
			log.Println(`filepicker.readDirMsg`)
			// log.Printf("MSG: %+#v\n", msg)
		} else {
			log.Printf("MSG: %+#v\n", msg)
		}
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)
	log.Println(m.filepicker.Path, m.selectedFile)

	// Did the user select a file?
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		// Get the path of the selected file.
		m.selectedFile = path
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(path + " is not valid.")
		m.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m *model2) View() string {
	// return m.t
	// return m.filepicker.CurrentDirectory
	// return m.filepicker.View()
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		log.Println(m.err)
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
	}
	s.WriteString("\n\n" + m.filepicker.View() + "\n")
	return s.String()
}
