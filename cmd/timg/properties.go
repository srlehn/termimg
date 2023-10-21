package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/testutil/routines"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func init() {
	propCmd.Flags().StringVarP(&propTTY, `tty`, `t`, ``, `tty for which properties are queried`)
	propCmd.Flags().BoolVarP(&propTerm, `name`, `n`, false, `likely terminal name`)
	propCmd.Flags().BoolVarP(&propDrawers, `drawers`, `d`, false, `supported drawers`)
	propCmd.Flags().BoolVarP(&propQueries, `queries`, `q`, false, `list query replies used for terminal identification, like device attributes, etc`)
	propCmd.Flags().BoolVarP(&propEnv, `environ`, `e`, false, `list environment variables used for terminal identification`)
	propCmd.Flags().BoolVarP(&propPassages, `passages`, `p`, false, `tty passages like terminal multiplexers, ssh, ...`)
	propCmd.Flags().BoolVarP(&propWindow, `window`, `w`, false, `list window properties`)
	propCmd.Flags().BoolVarP(&propXRes, `resources`, `r`, false, `list X11-Resources`)
	propCmd.Flags().BoolVar(&propDA1Attrs, propDA1AttrsFlag, false, `print DA1 attributes`)
	propCmd.Flags().BoolVar(&propDA2ModelLetter, propDA2ModelLetterFlag, false, `print DA2 model letter`)
	propCmd.Flags().BoolVar(&propDA2Model, propDA2ModelFlag, false, `print DA2 model`)
	propCmd.Flags().BoolVar(&propDA3, propDA3Flag, false, `print DA3 ID`)
	propCmd.Flags().BoolVar(&propDA3Hex, propDA3HexFlag, false, `print DA3 ID (hex)`)
	propCmd.Flags().BoolVar(&propXTVer, propXTVerFlag, false, `print XTVERSION`)
	propCmd.Flags().BoolVar(&propXTGetTCapTN, propXTGetTCapTNFlag, false, `print XTGETTCAP(TN)`)
	propCmd.MarkFlagsMutuallyExclusive(
		propDA1AttrsFlag,
		propDA2ModelLetterFlag,
		propDA2ModelFlag,
		propDA3Flag,
		propDA3HexFlag,
		propXTVerFlag,
		propXTGetTCapTNFlag,
	)
	rootCmd.AddCommand(propCmd)
}

var (
	propDA1AttrsFlag       = `da1-attributes`
	propDA2ModelLetterFlag = `da2-letter`
	propDA2ModelFlag       = `da2`
	propDA3Flag            = `da3`
	propDA3HexFlag         = `da3-hex`
	propXTVerFlag          = `xtversion`
	propXTGetTCapTNFlag    = `xtgettcap-tn`
	propTTY                string
	propTerm               bool
	propDrawers            bool
	propQueries            bool
	propEnv                bool
	propPassages           bool
	propWindow             bool
	propXRes               bool
	propDA1Attrs           bool
	propDA2ModelLetter     bool
	propDA2Model           bool
	propDA3                bool
	propDA3Hex             bool
	propXTVer              bool
	propXTGetTCapTN        bool
)

var propCmd = &cobra.Command{
	Use:              "properties",
	Short:            "list terminal properties",
	Long:             "list terminal properties",
	Args:             cobra.NoArgs,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		run(propFunc(cmd, args))
	},
}

func propFunc(cmd *cobra.Command, args []string) terminalSwapper {
	return func(tm **term.Terminal) error {
		wm.SetImpl(wmimpl.Impl())
		var ptyName string
		if len(propTTY) > 0 {
			ptyName = propTTY
		} else {
			ptyName = internal.DefaultTTYDevice()
		}
		opts := []term.Option{
			logFileOption,
			termimg.DefaultConfig,
			term.SetPTYName(ptyName),
			term.SetResizer(&rdefault.Resizer{}),
		}
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2

		var ret string
		for _, i := range []struct {
			flag    bool
			propKey string
		}{
			{propDA1Attrs, propkeys.DeviceAttributes},
			{propDA2ModelLetter, propkeys.DA2ModelLetter},
			{propDA2Model, propkeys.DA2Model},
			{propDA3, propkeys.DA3ID},
			{propDA3Hex, propkeys.DA3IDHex},
			{propXTVer, propkeys.XTVERSION},
			{propXTGetTCapTN, propkeys.XTGETTCAPSpecialTN},
		} {
			if !i.flag {
				continue
			}
			s, ok := tm2.Property(i.propKey)
			if !ok || len(s) == 0 {
				return errors.New(`property not found`)
			}
			ret = s
		}
		if len(ret) > 0 {
			fmt.Println(strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf(`%q`, ret), `"`), `"`))
			return nil
		}

		if !propTerm && !propDrawers && !propQueries && !propEnv && !propPassages && !propWindow && !propXRes {
			propTerm = true
			propDrawers = true
			propQueries = true
			propEnv = true
			propWindow = true
			propXRes = true
		}

		if err = routines.ListTermProps(tm2, propTerm, propDrawers, propQueries, propEnv, propPassages, propWindow, propXRes); logx.IsErr(err, tm2, slog.LevelError) {
			return err
		}
		return nil
	}
}
