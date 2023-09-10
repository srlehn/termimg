//go:build !wasm
// +build !wasm

package prompt

/*
test state
----------
todo:
bash specific: "\j", "\l", "\!", "\#" not implemented
shell variable expansion needs checking
unescaping order needs checking
regex compilation takes time, replace if possible, compile static ones one time
go test with PS1
restructure to methods: PS1.Get(), PS1.Length(), ...
*/

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	// "mvdan.cc/sh/shell"
	// "github.com/mattn/go-shellwords"
)

func GetPromptCleaned(env []string) (p string, err error) {
	p, err = GetPrompt(env)
	if err != nil {
		return ``, err
	}

	// remove some common ANSI escape sequences
	p = regexp.MustCompilePOSIX(`(`+"\033"+`|\\e|\\033|\\x1[bB])\[[0-9;]*[[:alpha:]]`).ReplaceAllLiteralString(p, ``)
	p = regexp.MustCompile(`([^\\])(\\[a]|\\007)`).ReplaceAllString(strings.ReplaceAll(p, "\007", ``), `$1`)
	p = applyControlChars(p)

	return
}

func GetPromptLength(env []string) (uint, error) {
	prompt, err := GetPromptCleaned(env)
	return uint(len(prompt)) + 1, err
}

func GetPromptLineLengths(env []string) (l []uint, err error) {
	prompt, e := GetPromptCleaned(env)
	if err != nil {
		return l, e
	}

	// add spot for cursor (which could also cause a line break)
	for _, p := range strings.Split(prompt+` `, "\n") {
		l = append(l, uint(len(p)))
	}
	return l, err
}

func GetPromptSize(env []string) (width, height uint, err error) {
	l, e := GetPromptLineLengths(env)
	err = e
	if err != nil {
		width = 80 // assume 80x24 terminal size
		height = 1
		return
	}

	height = uint(len(l))

	for _, i := range l {
		if i > width {
			width = i
		}
	}

	return
}

func getEnv(env []string, key string) (string, bool) {
	// if env == nil {env = os.Environ()}
	key += `=`
	for _, e := range env {
		e, isHome := strings.CutPrefix(e, key)
		if !isHome {
			continue
		}
		return e, true
	}
	return ``, false
}

func getWDTrimmed(env []string, onlyBase bool) string {
	/* PROMPT_DIRTRIM
	** If set to a number greater than zero,
	** the value is used as the number of trailing directory components to retain
	** when expanding the \w and \W prompt string escapes (see PROMPTING below).
	** Characters removed are replaced
	** with an ellipsis. */

	sep := string(os.PathSeparator)

	home, _ := getEnv(env, `HOME`)
	if len(home) == 0 {
		h, err := os.UserHomeDir()
		if err != nil || len(h) == 0 {
			return ``
		}
		home = h
	}
	home = strings.TrimRight(home, sep)

	wd, err := os.Getwd()
	if err != nil {
		return ``
	}

	if len(wd) > len(home) && wd[:len(home)] == home {
		wd = `~` + sep + strings.TrimLeft(wd[len(home):], sep)
	} else if len(wd) == len(home) && wd[:len(home)] == home {
		wd = `~`
	}

	if onlyBase {
		s := strings.Split(wd, sep)
		return s[len(s)-1]
	}

	pdts := os.Getenv(`PROMPT_DIRTRIM`)
	if pdt, err := strconv.Atoi(pdts); err == nil && pdt > 0 {
		wdp := strings.Split(wd, sep)
		l := len(wdp)
		// if l <= pdt || l == pdt+1 && (wdp[0] == `` || wdp[0][0] == '~') {return wd}
		if l < pdt {
			return wd
		}
		wd = strings.Join(wdp[l-pdt:], sep)
		//if (l > pdt && wdp[0][0] != '~') || (l > pdt+1 && wdp[0][0] == '~') {
		if len(wdp[0]) > 0 && wdp[0][0] == '~' {
			if l > pdt+1 {
				wd = `...` + sep + wd
			}
			if wd != `~` {
				wd = `~` + sep + wd
			}
		} else {
			wd = sep + wd
			if l > pdt+1 {
				wd = `...` + wd
			}
		}
	}

	return wd
}

func applyControlChars(s string) (ret string) {
	var line string
	var j int

	for i := 0; i < len(s); i++ {
		switch r := s[i]; r {
		case '\r':
			j = 0
		case '\n':
			ret += line + "\n"
			line = ``
			j = 0
		case '\b':
			if j > 0 {
				j--
			}
		default:
			if len(line) <= j {
				line += string(r)
			} else {
				line = line[:j] + string(r) + line[j+1:]
			}
			j++
		}
	}

	if j != 0 {
		ret += line
	}
	return ret
}
