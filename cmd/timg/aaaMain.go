package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"

	"github.com/srlehn/termimg/internal"
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
	rootCmd.PersistentFlags().BoolVar(&debug, `debug`, false, `debug errors`)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var debug bool

func run(fn func() error) {
	var err error
	if fn == nil {
		err = errors.New(internal.ErrNilParam)
	}
	err = fn()
	if err != nil {
		if stackFramer, ok := err.(interface{ ErrorStack() string }); debug && ok {
			fmt.Println(stackFramer.ErrorStack())
		} else {
			log.Fatal(err)
		}
	}
}
