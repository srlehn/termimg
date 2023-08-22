package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/term"
)

var (
	queryRaw   bool
	queryBasic bool
	queryHex   bool
	queryST    bool
)

func init() {
	queryCmd.PersistentFlags().BoolVarP(&queryRaw, `raw`, `r`, false, `unmodified terminal reply. this will likely be an escape sequence which will affect the terminal if stdout is not caught.`)
	queryCmd.PersistentFlags().BoolVarP(&queryBasic, `basic`, `b`, false, `disable replacing escape sequence names in the input with their values.`)
	queryCmd.PersistentFlags().BoolVarP(&queryHex, `hex`, `x`, false, `display non-printable characters as hexadecimal values.`)
	queryCmd.PersistentFlags().BoolVarP(&queryST, `st`, `t`, false, `don't try terminating rogue escape sequences with an ST at the beginning of the output.`)
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   queryCmdStr,
	Short: `query terminal`,
	Long:  `query terminal`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(queryFunc(cmd, args))
	},
}

var (
	queryCmdStr = "query"
	// queryUsageStr = `usage: ` + os.Args[0] + ` ` + queryCmdStr + ` (-r)`
)

func queryFunc(cmd *cobra.Command, args []string) func(**term.Terminal) error {
	return func(tm **term.Terminal) error {
		var query string
		if len(args) == 0 || args[0] == `-` {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return errors.New(err)
			}
			query = strings.TrimSuffix(string(b), "\n")
		} else {
			query = args[0]
		}
		if len(query) == 0 {
			return errors.New(`empty query`)
		}

		tm2, err := termimg.Terminal()
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
		return err
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
