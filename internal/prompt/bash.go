//go:build !wasm && !tinygo
// +build !wasm,!tinygo

package prompt

import (
	"os/exec"
	"strings"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/exc"
)

func getBashVersion() ([]string, error) {
	v := []string{`0`, `0`, `0`}

	// os.Getenv("BASH_VERSION")   // not exported

	bashAbs, err := exc.LookSystemDirs(`bash`)
	if err != nil {
		return v, err
	}
	cmd := exec.Command(bashAbs, `--version`)
	cmd.Env = []string{`LC_ALL=C`} // English - disable localization
	b, err := cmd.Output()
	if err != nil {
		return v, errors.New(err)
	}
	sp := strings.Split(strings.Split(string(b), "\n")[0], ` version `)
	if len(sp) != 2 {
		return v, errors.New(err)
	}

	var j int8
	w := []string{``, ``, ``}
	errStr := `unable to determine bash version`
	for i := 0; i < len(sp[1]); i++ {
		c := sp[1][i]
		switch {
		case c == '.':
			j++
		case strings.ContainsRune(`0123456789`, rune(c)):
			w[j] += string(c)
		default:
			if len(w[0]) > 0 && len(w[1]) > 0 && len(w[2]) > 0 {
				return w, nil
			}
			return v, errors.New(errStr)
		}
	}
	return v, errors.New(errStr)
}
