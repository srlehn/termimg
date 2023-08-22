//go:build dev

package main

func init() {
	showCmd.PersistentFlags().StringVarP(&showTTY, `tty`, `t`, ``, `tty to draw on`)
}

// TODO sometimes gibberish ends up in the destination tty
