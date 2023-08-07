package main

import (
	"errors"
	"image"
	"os"
	"time"

	errorsGo "github.com/go-errors/errors"
	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func init() { rootCmd.AddCommand(showCmd) }

var (
	showDrawer       string
	showPosition     string
	showImage        string
	showCaire        bool
	showResizerCaire = func() term.Resizer { return nil }
)

func init() {
	showCmd.PersistentFlags().StringVarP(&showDrawer, `drawer`, `d`, ``, `drawer to use`)
	showCmd.PersistentFlags().StringVarP(&showPosition, `position`, `p`, ``, `image position in cell coordinates <x>,<y>,<w>x<h>`)
}

var showCmd = &cobra.Command{
	Use:   showCmdStr,
	Short: "display image",
	Long: `display image

image position in cell coordinates: <x>,<y>,<w>x<h>

If width or height is missing the image will be scaled while preserving its aspect ratio.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(showFunc(cmd, args))
	},
}

var showCmdStr = "show"

var errShowUsage = errors.New(`usage: ` + os.Args[0] + ` ` + showCmdStr + ` -d <drawer> -p <x>,<y>,<w>x<h> /path/to/image.png`)

func showFunc(cmd *cobra.Command, args []string) func() error {
	return func() error {
		showImage = args[0]

		wm.SetImpl(wmimpl.Impl())
		var rsz term.Resizer
		if showCaire {
			rszCaire := showResizerCaire()
			if rszCaire == nil {
				return errorsGo.New(`caire drawer not set`)
			}
			rsz = rszCaire
		} else {
			rsz = &rdefault.Resizer{}
		}
		opts := &term.Options{
			PTYName:         internal.DefaultTTYDevice(),
			TTYProvFallback: gotty.New,
			Querier:         qdefault.NewQuerier(),
			WindowProvider:  wm.NewWindow,
			Resizer:         rsz,
		}
		tm, err := term.NewTerminal(opts)
		if err != nil {
			return err
		}
		defer tm.Close()

		x, y, w, h, err := splitDimArg(showPosition, tm, showImage)
		if err != nil {
			return err
		}
		bounds := image.Rect(x, y, x+w, y+h)

		dr := tm.Drawers()[0]
		if len(showDrawer) > 0 {
			dr = term.GetRegDrawerByName(showDrawer)
			if dr == nil {
				return errorsGo.New(`unknown drawer "` + showDrawer + `"`)
			}
		}
		if err := dr.Draw(termimg.NewImageFileName(showImage), bounds, rsz, tm); err != nil {
			return err
		}

		time.Sleep(2 * time.Second) // TODO rm
		return nil
	}
}
