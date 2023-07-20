//go:build dev

package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Println(a ...any) { fmt.Println(append([]any{fileLinePrefix(2)}, a...)...) }

func Printf(format string, a ...any) { fmt.Printf(fileLinePrefix(2)+format, a...) }

func Printfv(a ...any) {
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%+#v`+sep, len(a)), sep) + "\n"
	fmt.Printf(fileLinePrefix(2)+format, a...)
}

func Printfq(a ...any) {
	sep := ` - `
	format := strings.TrimSuffix(strings.Repeat(`%q`+sep, len(a)), sep) + "\n"
	fmt.Printf(fileLinePrefix(2)+format, a...)
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
