package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/exc"
	"github.com/srlehn/termimg/term"
)

var rootCmd = &cobra.Command{
	Short:        "timg display terminal graphics",
	Long:         "timg display terminal graphics",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
		os.Exit(1)
	},
}

func init() {
	cobra.EnablePrefixMatching = true
	rootCmd.PersistentFlags().BoolVar(&debugFlag, `debug`, false, `debug errors`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var debugFlag bool

func run(fn func(tm **term.Terminal) error) {
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
			if stackFramer, ok := r.(interface{ ErrorStack() string }); ok {
				fmt.Fprintln(os.Stderr, "\n"+stackFramer.ErrorStack())
			} else {
				debug.PrintStack()
			}
		}
		if err := tm.Close(); err != nil {
			// fallback
			sttyAbs, err := exc.LookSystemDirs(`stty`)
			if err == nil {
				// _ = exec.Command(sttyAbs, `echo`).Run()
				_ = exec.Command(sttyAbs, `sane`).Run()
			}
		}
		os.Exit(exitCode)
	}()
	err = fn(&tm)
	if err != nil {
		exitCode = 1
		if stackFramer, ok := err.(interface{ ErrorStack() string }); debugFlag && ok {
			fmt.Fprintln(os.Stderr, "\n"+stackFramer.ErrorStack())
		} else {
			fmt.Fprintln(os.Stderr, "\n"+err.Error())
		}
	}
}
