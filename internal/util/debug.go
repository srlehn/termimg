//go:build dev

package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Println(a ...any) { fmt.Println(append([]any{fileLinePrefix(2)}, a...)...) }

func PrintAt(x, y uint, a ...any) {
	fmt.Print(append(append([]any{storePosAndJumpToPosStr(x, y), fileLinePrefix(2)}, a...), restorePosStr)...)
}

func Printf(format string, a ...any) { fmt.Printf(fileLinePrefix(2)+format, a...) }

func PrintfAt(x, y uint, format string, a ...any) {
	fmt.Printf(storePosAndJumpToPosStr(x, y)+fileLinePrefix(2)+format+restorePosStr, a...)
}

func Printfv(a ...any) {
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%+#v`+sep, len(a)), sep) + "\n"
	fmt.Printf(fileLinePrefix(2)+format, a...)
}

func PrintfvAt(x, y uint, a ...any) {
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%+#v`+sep, len(a)), sep) + "\n"
	fmt.Printf(storePosAndJumpToPosStr(x, y)+fileLinePrefix(2)+format+restorePosStr, a...)
}

func Printfq(a ...any) {
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%q`+sep, len(a)), sep) + "\n"
	fmt.Printf(fileLinePrefix(2)+format, a...)
}

func PrintfqAt(x, y uint, a ...any) {
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%q`+sep, len(a)), sep) + "\n"
	fmt.Printf(storePosAndJumpToPosStr(x, y)+fileLinePrefix(2)+format+restorePosStr, a...)
}

func Printfj(a ...any) {
	sep := "\n"
	prefix := fileLinePrefix(2)
	var strs []string
	for _, o := range a {
		b, err := json.MarshalIndent(o, ``, `  `)
		if err != nil {
			strs = append(strs, prefix+fmt.Sprintf("error: %q\n  %v", err.Error(), o))
		} else {
			strs = append(strs, prefix+string(b))
		}
	}
	fmt.Println(strings.Join(strs, sep))
}
