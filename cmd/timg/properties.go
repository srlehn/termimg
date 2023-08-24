package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/testutil/routines"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func init() {
	propCmd.PersistentFlags().StringVarP(&propTTY, `tty`, `t`, ``, `tty for which properties are queried`)
	propCmd.PersistentFlags().BoolVarP(&propTerm, `name`, `n`, false, `likely terminal name`)
	propCmd.PersistentFlags().BoolVarP(&propDrawers, `drawers`, `d`, false, `supported drawers`)
	propCmd.PersistentFlags().BoolVarP(&propQueries, `queries`, `q`, false, `list query replies used for terminal identification, like device attributes, etc`)
	propCmd.PersistentFlags().BoolVarP(&propEnv, `environ`, `e`, false, `list environment variables used for terminal identification`)
	propCmd.PersistentFlags().BoolVarP(&propPassages, `passages`, `p`, false, `tty passages like terminal multiplexers, ssh, ...`)
	propCmd.PersistentFlags().BoolVarP(&propWindow, `window`, `w`, false, `list window properties`)
	propCmd.PersistentFlags().BoolVarP(&propXRes, `resources`, `r`, false, `list X11-Resources`)
	propCmd.PersistentFlags().BoolVar(&propDA1Attrs, `da1-attributes`, false, `print DA1 attributes`)
	propCmd.PersistentFlags().BoolVar(&propDA2ModelLetter, `da2-letter`, false, `print DA2 model letter`)
	propCmd.PersistentFlags().BoolVar(&propDA2Model, `da2`, false, `print DA2 model`)
	propCmd.PersistentFlags().BoolVar(&propDA3, `da3`, false, `print DA3 ID`)
	propCmd.PersistentFlags().BoolVar(&propDA3Hex, `da3-hex`, false, `print DA3 ID (hex)`)
	propCmd.PersistentFlags().BoolVar(&propXTVer, `xtversion`, false, `print XTVERSION`)
	propCmd.PersistentFlags().BoolVar(&propXTGetTCapTN, `xtgettcap-tn`, false, `print XTGETTCAP(TN)`)
	rootCmd.AddCommand(propCmd)
}

var (
	propTTY            string
	propTerm           bool
	propDrawers        bool
	propQueries        bool
	propEnv            bool
	propPassages       bool
	propWindow         bool
	propXRes           bool
	propDA1Attrs       bool
	propDA2ModelLetter bool
	propDA2Model       bool
	propDA3            bool
	propDA3Hex         bool
	propXTVer          bool
	propXTGetTCapTN    bool
)

var propCmd = &cobra.Command{
	Use:   "properties",
	Short: "list terminal properties",
	Long:  "list terminal properties",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		run(propFunc(cmd, args))
	},
}

func propFunc(cmd *cobra.Command, args []string) func(**term.Terminal) error {
	return func(tm **term.Terminal) error {
		wm.SetImpl(wmimpl.Impl())
		var ptyName string
		if len(propTTY) > 0 {
			ptyName = propTTY
		} else {
			ptyName = internal.DefaultTTYDevice()
		}
		opts := []term.Option{
			termimg.DefaultConfig,
			term.SetPTYName(ptyName),
			term.SetResizer(&rdefault.Resizer{}),
		}
		var err error
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2

		var ret string
		var flagCnt int
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
			flagCnt++
			if flagCnt > 1 {
				return errors.New(`only one property flag can be set`)
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

		if err = routines.ListTermProps(tm2, propTerm, propDrawers, propQueries, propEnv, propPassages, propWindow, propXRes); err != nil {
			return err
		}
		return nil
	}
}
