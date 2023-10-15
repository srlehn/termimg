package testutil

import (
	"fmt"

	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/wm"
)

func ListWindows(env environ.Properties) error {
	// TODO -> testutil
	fmt.Println("name\tclass\tinstance")
	fmt.Println() // make the linter shut up by separating terminal LF
	conn, err := wm.NewConn(env)
	if err != nil {
		return err
	}
	defer conn.Close()
	windows, err := conn.Windows()
	if err != nil {
		return err
	}
	for _, window := range windows {
		fmt.Printf("%s\t%s\t%s\n", window.WindowName(), window.WindowClass(), window.WindowInstance())
	}
	return nil
}
