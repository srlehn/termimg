//go:build dev

package main

import (
	"errors"
	"image"
	"os"

	errorsGo "github.com/go-errors/errors"
	"github.com/spf13/cobra"

	_ "github.com/srlehn/termimg/drawers/all"
	"github.com/srlehn/termimg/internal/testutil"
	_ "github.com/srlehn/termimg/terminals"
)

var (
	runTermTerm     string
	runTermDrawer   string
	runTermPosition string
	runTermImage    string
)

func init() {
	runTermCmd.PersistentFlags().StringVarP(&runTermTerm, `term`, `t`, ``, `terminal to run`)
	runTermCmd.PersistentFlags().StringVarP(&runTermDrawer, `drawer`, `d`, ``, `drawer to use`)
	runTermCmd.PersistentFlags().StringVarP(&runTermPosition, `position`, `p`, ``, `image position in cell coordinates <x>,<y>,<w>x<h>`)
	rootCmd.AddCommand(runTermCmd)
}

var runTermCmd = &cobra.Command{
	Use:   runTermCmdStr,
	Short: "open image in new terminal and screenshot",
	Long:  `open image in new terminal and screenshot`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(runTermFunc(cmd, args))
	},
}

var runTermCmdStr = "runterm"

var errRunTermUsage = errors.New(`usage: ` + os.Args[0] + ` ` + runTermCmdStr + ` -t <terminal> -d <drawer> -p <x>,<y>,<w>x<h> /path/to/image.png`)

func runTermFunc(cmd *cobra.Command, args []string) func() error {
	return func() error {
		runTermImage = args[0]
		imgFileBytes, err := os.ReadFile(runTermImage)
		if err != nil {
			return errorsGo.New(err)
		}

		x, y, w, h, err := splitDimArg(runTermPosition, nil, runTermImage) // TODO pass term.Terminal
		if err != nil {
			return errorsGo.New(errRunTermUsage)
		}
		bounds := image.Rect(x, y, x+w, y+h)

		doDisplay := false
		if err := testutil.PTermPrintImageHelper(
			runTermTerm, runTermDrawer,
			testutil.DrawFuncPictureWithFrame,
			imgFileBytes, bounds, ``, doDisplay,
		); err != nil {
			return err
		}
		return nil
	}
}
