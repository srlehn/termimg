package main

import (
	"image"
	"os"
	"time"

	errorsGo "github.com/go-errors/errors"
	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/testutil"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

var (
	showDrawer       string
	showPosition     string
	showImage        string
	showTTY          string
	showCaire        bool
	showCoords       bool
	showGrid         bool
	showResizerCaire = func() term.Resizer { return nil }
)

// file name because of init() call order
func init() {
	showCmd.PersistentFlags().StringVarP(&showDrawer, `drawer`, `d`, ``, `drawer to use`)
	showCmd.PersistentFlags().StringVarP(&showPosition, `position`, `p`, ``, `image position in cell coordinates <x>,<y>,<w>x<h>`)
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   showCmdStr,
	Short: `display image`,
	Long: `display image

` + showUsageStr + `

Image position is given in cell coordinates.
If width or height is missing the image will be scaled while preserving its aspect ratio.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(showFunc(cmd, args))
	},
}

var (
	showCmdStr   = "show"
	showUsageStr = `usage: ` + os.Args[0] + ` ` + showCmdStr + ` -p <x>,<y>,<w>x<h> (-d <drawer>) (-t <tty>) /path/to/image.png`
	errShowUsage = errorsGo.New(showUsageStr)
)

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
		var ptyName string
		if len(showTTY) > 0 {
			ptyName = showTTY
		} else {
			ptyName = internal.DefaultTTYDevice()
		}
		opts := &term.Options{
			PTYName:         ptyName,
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
		if showCoords {
			tm.WriteString(testutil.NumberArea(resizeArea(bounds, 5)))
		}
		if showGrid {
			tm.WriteString(testutil.ChessPattern(resizeArea(bounds, 3), false))
		}
		if err := dr.Draw(termimg.NewImageFileName(showImage), bounds, rsz, tm); err != nil {
			return err
		}

		time.Sleep(2 * time.Second) // TODO rm when relevant drawers are persistent
		return nil
	}
}

func resizeArea(area image.Rectangle, diff int) image.Rectangle {
	ret := area
	ret.Min.X -= diff
	ret.Min.Y -= diff
	ret.Max.X += diff
	ret.Max.Y += diff
	return ret
}
