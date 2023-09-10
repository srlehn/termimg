//go:build !wasm && !tinygo
// +build !wasm,!tinygo

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
	"bytes"
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mvdan.cc/sh/syntax"
	// "mvdan.cc/sh/shell"
	"mvdan.cc/sh/interp"
	// "github.com/mattn/go-shellwords"
	ps "github.com/mitchellh/go-ps"
	"github.com/srlehn/termimg/internal/errors"
)

var (
	commercialMinusSign         = "\u2052"
	rePercent                   = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%(%[^}]*})`)
	reCommercialMinusSign       = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^` + commercialMinusSign + `])?)` + commercialMinusSign)
	reDateYear4Digits           = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%Y([^}]*})`)
	reDateYear2Digits           = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%y([^}]*})`)
	reDateMonth2Digits          = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%m([^}]*})`)
	reDateMonthNameShort        = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%b([^}]*})`)
	reDateMonthName             = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%B([^}]*})`)
	reDateDay2Digits            = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%d([^}]*})`)
	reDateDayNameShort          = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%a([^}]*})`)
	reTimeHour                  = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%H([^}]*})`)
	reTimeHour12H               = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%I([^}]*})`)
	reTimeMinute                = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%M([^}]*})`)
	reTimeSecond                = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%S([^}]*})`)
	reTimeAMPM                  = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%p([^}]*})`)
	reTimeZone                  = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%Z([^}]*})`)
	reTimeZoneISO8601Offset     = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^%])?)%z([^}]*})`)
	reTimeISO8601               = regexp.MustCompile(`([^\\]|^)(\\D{)(([^}]*[^%])?%T([^}]*)?)?(})`)
	reDateStrftimeStartAnchored = regexp.MustCompile(`^\\D{[^}]*}`)
	reDateStrftimeStart         = regexp.MustCompile(`[^\\]\\D{[^}]*}`)
)

// assumes bash as shell
// TODO: disable for windows

func GetPrompt(env []string) (string, error) {
	printfs := func(s string) string { return `printf %s "` + s + `"` }

	ps1, okPS1 := getEnv(env, `PS1`)
	if !okPS1 {
		return ``, errors.New(`no PS1 env var`)
	}

	for {
		pold := ps1
		// special prompt escaping for non printing characters
		// p = regexp.MustCompile(`([^\\])\\[][]`).ReplaceAllString(p, `$1`)

		// golang reference date: "Jan 2, 2006 at 3:04pm (MST)"
		// some date strftime() "conversion specifications"
		// p = regexp.MustCompile(`([^\\]|^)(\\D{([^}]*[^}\\])?)(;[^}]*})`).ReplaceAllString(p, `$1$2\$4`)   // ;
		ps1 = rePercent.ReplaceAllString(ps1, `$1${2}`+commercialMinusSign+`$4`)
		ps1 = reDateYear4Digits.ReplaceAllString(ps1, `$1${2}2006$4`)        // %Y
		ps1 = reDateYear2Digits.ReplaceAllString(ps1, `$1${2}06$4`)          // %y
		ps1 = reDateMonth2Digits.ReplaceAllString(ps1, `$1${2}01$4`)         // %m
		ps1 = reDateMonthNameShort.ReplaceAllString(ps1, `$1${2}Jan$4`)      // %b
		ps1 = reDateMonthName.ReplaceAllString(ps1, `$1${2}January$4`)       // %B
		ps1 = reDateDay2Digits.ReplaceAllString(ps1, `$1${2}02$4`)           // %d
		ps1 = reDateDayNameShort.ReplaceAllString(ps1, `$1${2}Mon$4`)        // %a
		ps1 = reTimeHour.ReplaceAllString(ps1, `$1${2}15$4`)                 // %H
		ps1 = reTimeHour12H.ReplaceAllString(ps1, `$1${2}03$4`)              // %I
		ps1 = reTimeMinute.ReplaceAllString(ps1, `$1${2}04$4`)               // %M
		ps1 = reTimeSecond.ReplaceAllString(ps1, `$1${2}05$4`)               // %S
		ps1 = reTimeAMPM.ReplaceAllString(ps1, `$1${2}PM$4`)                 // %p
		ps1 = reTimeZone.ReplaceAllString(ps1, `$1${2}MST$4`)                // %Z
		ps1 = reTimeZoneISO8601Offset.ReplaceAllString(ps1, `$1${2}-0700$4`) // %z
		ps1 = reTimeISO8601.ReplaceAllString(ps1, `$1$2${4}15:04:05$5$6`)    // %T = %H:%M:%S
		if ps1 == pold {
			break
		}
	}
	// replace temporary commercial minus signs with original real percent signs
	for {
		pold := ps1
		ps1 = reCommercialMinusSign.ReplaceAllString(ps1, `$1${2}%`)
		if ps1 == pold {
			break
		}
	}
	ps1 = reDateStrftimeStartAnchored.ReplaceAllStringFunc(ps1, func(s string) string { return time.Now().Format(s[len(`\D{`) : len(s)-len(`}`)]) })
	ps1 = reDateStrftimeStart.ReplaceAllStringFunc(ps1, func(s string) string { return s[:1] + time.Now().Format(s[len(`?\D{`):len(s)-len(`}`)]) })

	/*
		\j     the number of jobs currently managed by the shell
		\l     the basename of the shell's terminal device name
		\!     the history number of this command
		\#     the command number of this command
	*/
	var bashVer []string
	getBashVer := func() ([]string, error) {
		if len(bashVer) > 0 {
			return bashVer, nil
		}
		bv, err := getBashVersion()
		bashVer = bv
		return bashVer, err
	}
	var q, oct string
	var b bool
	var o int8
	for i := 0; i < len(ps1); i++ {
		// octal \0xx
		if o > 0 {
			oct += string(ps1[i])
			if !strings.ContainsRune(`01234567`, rune(ps1[i])) {
				o = 0
				q += `\` + oct
				continue
			}
			if o == 2 {
				o = 0
				if n, err := strconv.ParseUint(oct, 8, 64); err == nil {
					q += string(rune(n))
				} else {
					o = 0
					q += `\` + oct
					continue
				}
				continue
			}
			o++
			continue
		}
		if b {
			b = false
			d := ``
			switch c := ps1[i]; c {
			case 'a':
				d = "\007"
			case 'd':
				d = time.Now().Format(`Mon Jan 02`) // not localized
			// case 'D':   // handled above
			case 'e':
				d = "\033"
			case 'h':
				if h, err := os.Hostname(); err == nil {
					d = strings.Split(h, `.`)[0]
				}
			case 'H':
				if h, err := os.Hostname(); err == nil {
					d = h
				}
			case 'j': // not implemented
			case 'l': // not implemented
			case 'n':
				d = "\n"
			case 'r':
				d = "\r"
			case 's':
				if pr, err := ps.FindProcess(syscall.Getppid()); err == nil {
					prn := strings.Split(pr.Executable(), string(os.PathSeparator))
					d = prn[len(prn)-1]
				} else {
					d = `sh`
				}
			case 't':
				d = time.Now().Format(`15:04:05`)
			case 'T':
				d = time.Now().Format(`03:04:05`)
			case '@':
				d = time.Now().Format(`03:04 PM`) // not localized
			case 'A':
				d = time.Now().Format(`15:04`)
			case 'u':
				d, _ = getEnv(env, `USER`)
			case 'v':
				bv, err := getBashVer()
				_ = err // TODO log error
				d = strings.Join(bv[0:2], `.`)
			case 'V':
				bv, err := getBashVer()
				_ = err // TODO log error
				d = strings.Join(bv, `.`)
			case 'w':
				d = getWDTrimmed(env, false)
			case 'W':
				d = getWDTrimmed(env, true)
			case '!': // not implemented
			case '#': // not implemented
			case '$':
				if os.Getuid() == 0 {
					d = `#`
				} else {
					d = `$`
				}
			case '0':
				o = 1
				oct = string(c)
			case '\\':
				d = `\`
			case '[': // non printing characters - start
			case ']': // non printing characters - end
			default:
				d = `\` + string(c)
			}
			q += d
			continue
		}
		if c := ps1[i]; c == '\\' {
			b = true
		} else {
			q += string(c)
		}
	}
	ps1 = q

	// prepare for CODE EXECUTION
	ps1 = printfs(ps1)
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(ps1), "") // LangPOSIX, LangMirBSDKorn

	buf := new(bytes.Buffer)
	runner, _ := interp.New(
		interp.StdIO(nil, buf, nil),
		// interp.WithExecModules(),   // temporarily disabled because of upstream API changes
	)
	// ARBITRARY CODE EXECUTION (needed for subprocesses)
	// safer: shell.Expand from "mvdan.cc/sh/shell"
	runner.Run(context.TODO(), f)

	prompt := buf.String()

	return prompt, err
}
