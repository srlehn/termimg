//go:build dev

package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal/assets"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"

	// "github.com/srlehn/termimg/tty/contdtty"
	"github.com/srlehn/termimg/tty/bubbleteatty"

	tea "github.com/charmbracelet/bubbletea"
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
	mdl := newTestModel1()
	// mdl := newTestModel2()

	//ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	//defer cancel()
	teaOpts := []tea.ProgramOption{
		//tea.WithContext(ctx),
		tea.WithAltScreen(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	}
	var prog *tea.Program
	timgOpts := []term.Option{
		term.SetLogFile(`log_termimg.log`, true),
		termimg.DefaultConfig,
		bubbleteatty.BubbleTeaProgram(mdl, &prog, teaOpts...),
	}

	tm, err := term.NewTerminal(timgOpts...)
	if err != nil {
		return err
	}
	defer tm.Close()

	if stp, ok := any(mdl).(interface{ Setup(*term.Terminal) error }); ok {
		log.Println("SETUP", tm != nil)
		if err := stp.Setup(tm); err != nil {
			log.Println("SETUP", err)
			return err
		}
	}

	defer prog.Quit()
	if _, err = prog.Run(); logx.IsErr(err, tm, slog.LevelError) {
		return err
	}

	return nil
}
