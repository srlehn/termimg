package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

var (
	queryRaw   bool
	queryBasic bool
	queryHex   bool
	queryST    bool
	queryTTY   string
)

func init() {
	queryCmd.Flags().BoolVarP(&queryRaw, `raw`, `r`, false, `unmodified terminal reply. this will likely be an escape sequence which will affect the terminal if stdout is not caught.`)
	queryCmd.Flags().BoolVarP(&queryBasic, `basic`, `b`, false, `disable replacing escape sequence names in the input with their values.`)
	queryCmd.Flags().BoolVarP(&queryHex, `hex`, `x`, false, `display non-printable characters as hexadecimal values.`)
	queryCmd.Flags().BoolVarP(&queryST, `st`, `T`, false, `don't try terminating rogue escape sequences with an ST at the beginning of the output.`)
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:              queryCmdStr,
	Short:            `query terminal`,
	Long:             `query terminal`,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		run(queryFunc(cmd, args))
	},
}

var (
	queryCmdStr = "query"
	// queryUsageStr = `usage: ` + os.Args[0] + ` ` + queryCmdStr + ` (-r)`
)

func queryFunc(cmd *cobra.Command, args []string) terminalSwapper {
	return func(tm **term.Terminal) error {
		var query string
		if l := len(args); l == 0 || args[0] == `-` {
			fi, err := os.Stdin.Stat()
			if err != nil {
				return errors.New(err)
			}
			if fi.Mode()&os.ModeNamedPipe != os.ModeNamedPipe {
				return errors.New(`stdin is not a pipe`)
			}
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return errors.New(err)
			}
			query = strings.TrimSuffix(string(b), "\n")
		} else if l == 1 {
			query = args[0]
		} else {
			query = strings.Join(args, ` `)
		}
		if len(query) == 0 {
			return errors.New(`empty query`)
		}

		wm.SetImpl(wmimpl.Impl())
		var ptyName string
		if len(queryTTY) > 0 {
			ptyName = queryTTY
		} else {
			ptyName = internal.DefaultTTYDevice()
		}
		opts := []term.Option{
			logFileOption,
			termimg.DefaultConfig,
			term.SetPTYName(ptyName),
			term.ManualComposition, // TODO for tmux parent tty
		}
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2

		if !queryBasic {
			query = strings.TrimSuffix(strings.TrimPrefix(replacerDecode.Replace(` `+query+` `), ` `), ` `)
		}
		p := parser.NewParser(true, false)
		var seqCnt uint
		for _, r := range query {
			if p.Parse(r) {
				seqCnt++
			}
		}
		replCombined, err := tm2.Query(query, parser.NParser(parser.NewParser(true, false), seqCnt))
		var repls []string
		var lastSeqEnd int
		for j, r := range replCombined {
			if p.Parse(r) {
				repls = append(repls, replCombined[lastSeqEnd:j+1])
				lastSeqEnd = j + 1
				if len(repls) == int(seqCnt) {
					break
				}
			}
		}
		if !queryST {
			fmt.Print(queries.ST) // end rogue escape sequences
		}
		for _, repl := range repls {
			if len(repl) == 0 {
				continue
			}
			if !queryRaw {
				if !queryHex {
					repl = strings.TrimSuffix(strings.TrimPrefix(replacerEncode.Replace(repl), ` `), ` `)
				}
				repl = strings.Trim(fmt.Sprintf(`%q`, repl), `"`)
				fmt.Println(repl)
			} else {
				fmt.Print(repl)
			}
		}
		return logx.Err(err, tm2, slog.LevelError, `query`, query, `reply`, replCombined)
	}
}

var replacerDecode = strings.NewReplacer(
	` BEL `, queries.BEL,
	` BS `, queries.BS,
	` HT `, queries.HT,
	` LF `, queries.LF,
	` FF `, queries.FF,
	` CR `, queries.CR,
	` ESC `, queries.ESC,

	` SS2 `, queries.SS2,
	` SS3 `, queries.SS3,
	` DCS `, queries.DCS,
	` CSI `, queries.CSI,
	` ST `, queries.ST,
	` OSC `, queries.OSC,
	` SOS `, queries.SOS,
	` PM `, queries.PM,
	` APC `, queries.APC,

	` SCP `, queries.SCP,
	` SCOSC `, queries.SCOSC,
	` RCP `, queries.RCP,
	` SCORC `, queries.SCORC,

	` RIS `, queries.RIS,

	` DECSC `, queries.DECSC,
	` DECRC `, queries.DECRC,

	` ACS6 `, queries.ACS6,
	` S7C1T `, queries.S7C1T,
	` ACS7 `, queries.ACS7,
	` S8C1T `, queries.S8C1T,
)

var replacerEncode = strings.NewReplacer(
	queries.SS2, ` SS2 `,
	queries.SS3, ` SS3 `,
	queries.DCS, ` DCS `,
	queries.CSI, ` CSI `,
	queries.ST, ` ST `,
	queries.OSC, ` OSC `,
	queries.SOS, ` SOS `,
	queries.PM, ` PM `,
	queries.APC, ` APC `,

	queries.SCP, ` SCP `,
	queries.RCP, ` RCP `,

	queries.RIS, ` RIS `,

	queries.DECSC, ` DECSC `,
	queries.DECRC, ` DECRC `,

	queries.S7C1T, ` S7C1T `,
	queries.S8C1T, ` S8C1T `,

	// the former sequences might contain the following
	queries.BEL, ` BEL `,
	queries.BS, ` BS `,
	queries.HT, ` HT `,
	queries.LF, ` LF `,
	queries.FF, ` FF `,
	queries.CR, ` CR `,
	queries.ESC, ` ESC `,
)

// TODO add more sequences
