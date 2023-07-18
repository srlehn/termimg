package testutil

import (
	"fmt"

	"github.com/srlehn/termimg/wm"
)

func ListWindows() error {
	// TODO -> testutil
	fmt.Println("name\tclass\tinstance")
	fmt.Println() // make the linter shut up by separating terminal LF
	conn, err := wm.NewConn()
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
