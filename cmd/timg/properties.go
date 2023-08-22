package main

import (
	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal"
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
	propCmd.PersistentFlags().BoolVarP(&propQueries, `queries`, `q`, false, `list query replies used for terminal differentiation, like device attributes, etc`)
	propCmd.PersistentFlags().BoolVarP(&propEnv, `environ`, `e`, false, `list environment variables used for terminal differentiation`)
	propCmd.PersistentFlags().BoolVarP(&propPassages, `passages`, `p`, false, `tty passages like terminal multiplexers, ssh, ...`)
	propCmd.PersistentFlags().BoolVarP(&propWindow, `window`, `w`, false, `list window properties`)
	propCmd.PersistentFlags().BoolVarP(&propXRes, `resources`, `r`, false, `list X11-Resources`)
	rootCmd.AddCommand(propCmd)
}

var (
	propTTY      string
	propTerm     bool
	propDrawers  bool
	propQueries  bool
	propEnv      bool
	propPassages bool
	propWindow   bool
	propXRes     bool
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
