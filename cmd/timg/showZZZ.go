package main

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/testutil"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

var (
	showDrawer         string
	showPosition       string
	showImageLocalPath string
	showTTY            string
	showURL            string
	showFile           string
	showCaire          bool
	showCoords         bool
	showGrid           bool
	showResizerCaire   = func() term.Resizer { return nil }
)

// file name because of init() call order
func init() {
	showCmd.PersistentFlags().StringVarP(&showDrawer, `drawer`, `d`, ``, `drawer to use`)
	showCmd.PersistentFlags().StringVarP(&showPosition, `position`, `p`, ``, `image position in cell coordinates <x>,<y>,<w>x<h>`)
	showCmd.PersistentFlags().StringVarP(&showURL, `url`, `u`, ``, `image url`)
	showCmd.PersistentFlags().StringVarP(&showFile, `file`, `f`, ``, `image path`)
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   showCmdStr,
	Short: `display image`,
	Long: `display image

` + showUsageStr + `

Image position is given in cell coordinates.
If width or height is missing the image will be scaled while preserving its aspect ratio.`,
	// Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(showFunc(cmd, args))
	},
}

var (
	showCmdStr   = "show"
	showUsageStr = `usage: ` + os.Args[0] + ` ` + showCmdStr + ` -p <x>,<y>,<w>x<h> (-d <drawer>) (-t <tty>) <-f /path/to/image.png|-u http(s)://website.com/image.png>`
)

func showFunc(cmd *cobra.Command, args []string) func(tm **term.Terminal) error {
	return func(tm **term.Terminal) error {
		var timg image.Image
		if len(showFile) > 0 {
			p, err := filepath.Abs(showFile)
			if err != nil {
				return errors.New(err)
			}
			showImageLocalPath = p
		} else if len(showURL) == 0 {
			if len(args) == 1 && len(args[0]) > 0 {
				if strings.HasPrefix(args[0], `https://`) || strings.HasPrefix(args[0], `http://`) {
					showURL = args[0]
				} else {
					p, err := filepath.Abs(args[0])
					if err != nil {
						return errors.New(err)
					}
					if _, err := os.Stat(p); err == nil {
						showImageLocalPath = p
					} else {
						showURL = args[0]
					}
				}
			} else {
				return errors.New(`no image path specified`)
			}
		}
		if len(showImageLocalPath) > 0 {
			timg = termimg.NewImageFileName(showImageLocalPath)
		} else {
			m, err := downloadImage(showURL)
			if err != nil {
				return err
			}
			timg = m
		}
		if timg == nil {
			return errors.New(`nil image`)
		}

		wm.SetImpl(wmimpl.Impl())
		var rsz term.Resizer
		if showCaire {
			rszCaire := showResizerCaire()
			if rszCaire == nil {
				return errors.New(`caire drawer not set`)
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
		opts := []term.Option{
			termimg.DefaultConfig,
			term.SetPTYName(ptyName),
			term.SetResizer(rsz),
		}
		var err error
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2
		x, y, w, h, err := splitDimArg(showPosition, tm2, timg)
		if err != nil {
			return err
		}
		bounds := image.Rect(x, y, x+w, y+h)

		dr := tm2.Drawers()[0]
		if len(showDrawer) > 0 {
			dr = term.GetRegDrawerByName(showDrawer)
			if dr == nil {
				return errors.New(`unknown drawer "` + showDrawer + `"`)
			}
		}
		coordWidth := 2
		gridBorderWidth := 3
		if showCoords {
			cutOff := coordWidth
			if !showGrid {
				cutOff += gridBorderWidth
			}
			tm2.WriteString(testutil.NumberArea(resizeArea(bounds, coordWidth+gridBorderWidth), cutOff))
		}
		if showGrid {
			tm2.WriteString(testutil.ChessPattern(resizeArea(bounds, gridBorderWidth), false))
		}
		if err := dr.Draw(timg, bounds, tm2); err != nil {
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

func downloadImage(u string) (image.Image, error) {
	// https://zpjiang.me/2017/03/10/Download-File-with-Size-Limit/
	repl, err := http.Get(u)
	if err != nil {
		return nil, errors.New(err)
	}
	defer repl.Body.Close()
	buf := &bytes.Buffer{}
	var szLim int64 = 20 * 1024 * 1024 // 20MiB
	_, err = io.CopyN(buf, repl.Body, szLim)
	if err != nil {
		if err == io.EOF {
			return term.NewImageBytes(buf.Bytes()), nil
		} else {
			return nil, errors.New(err)
		}
	}
	if n, _ := io.ReadFull(repl.Body, make([]byte, 1)); n > 0 {
		return nil, errors.New(`image too large`) // TODO show limit
	}
	return term.NewImageBytes(buf.Bytes()), nil
}
