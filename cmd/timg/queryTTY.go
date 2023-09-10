//go:build dev

package main

func init() {
	queryCmd.Flags().StringVarP(&queryTTY, `tty`, `t`, ``, `tty to query`)
}

// TODO sometimes gibberish ends up in the destination tty
