//go:build !dev

package util

func Println(a ...any) {}

func PrintAt(x, y uint, a ...any) {}

func Printf(format string, a ...any)              {}
func PrintfAt(x, y uint, format string, a ...any) {}

func Printfv(a ...any)              {}
func PrintfvAt(x, y uint, a ...any) {}

func Printfq(a ...any)              {}
func PrintfqAt(x, y uint, a ...any) {}

func Printfj(a ...any) {}
