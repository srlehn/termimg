package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/exc"
	"github.com/srlehn/termimg/term"
)

var rootCmd = &cobra.Command{
	Use:              filepath.Base(os.Args[0]),
	Short:            "timg display terminal graphics",
	Long:             "timg display terminal graphics",
	SilenceUsage:     true,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
		os.Exit(1)
	},
}

func init() {
	cobra.EnablePrefixMatching = true
	// local flags
	rootCmd.Flags().BoolVarP(&debugFlag, `debug`, `d`, false, `debug errors`)
	rootCmd.Flags().BoolVarP(&silentFlag, `silent`, `s`, false, `silence errors`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var (
	debugFlag  bool
	silentFlag bool
)

type terminalSwapper func(tm **term.Terminal) error

func run(fn terminalSwapper) {
	var err error
	if fn == nil {
		err = errors.New(consts.ErrNilParam)
	}
	var tm *term.Terminal
	var exitCode int
	defer func() {
		// catch panics to ascertain the terminal is reset
		if r := recover(); r != nil {
			exitCode = 1
			if !silentFlag {
				if stackFramer, ok := r.(interface{ ErrorStack() string }); ok {
					fmt.Fprintln(os.Stderr, "\n"+stackFramer.ErrorStack())
				} else {
					debug.PrintStack()
				}
			}
		}
		if err := tm.Close(); err != nil {
			// fallback
			sttyAbs, err := exc.LookSystemDirs(`stty`)
			if err == nil {
				_ = exec.Command(sttyAbs, `sane`).Run()
			}
		}
		os.Exit(exitCode)
	}()
	err = fn(&tm)
	if err != nil {
		exitCode = 1
		if !silentFlag {
			if stackFramer, ok := err.(interface{ ErrorStack() string }); debugFlag && ok {
				fmt.Fprintln(os.Stderr, "\n"+stackFramer.ErrorStack())
			} else {
				fmt.Fprintln(os.Stderr, "\n"+err.Error())
			}
		}
	}
}
