package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/exc"
	"github.com/srlehn/termimg/internal/logx"
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
	rootCmd.Flags().StringVarP(&logFileFlag, `log-file`, `l`, ``, `log file`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var (
	// TODO debug flag should also set log level
	debugFlag      bool
	silentFlag     bool
	logFileFlag    string
	logFileOption  term.Option
	cpuProfileFlag string
	cpuProfilefunc func(profileFile string) func()
)

type terminalSwapper func(tm **term.Terminal) error

func run(fn terminalSwapper) {
	var err error
	if fn == nil {
		err = errors.NilParam()
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
	if len(cpuProfileFlag) > 0 && cpuProfilefunc != nil {
		defer cpuProfilefunc(cpuProfileFlag)()
	}
	if len(logFileFlag) > 0 {
		logFileOption = term.SetLogFile(logFileFlag, true)
	}
	if err = fn(&tm); err != nil {
		if tm != nil {
			logx.IsErr(err, tm, slog.LevelError)
		}
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
