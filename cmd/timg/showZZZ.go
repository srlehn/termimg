package main

import (
	"bytes"
	"image"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/propkeys"
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

// file name ending with ZZZ because of init() call order
func init() {
	showCmd.Flags().StringVarP(&showDrawer, `drawer`, `d`, ``, `drawer to use`)
	showCmd.Flags().StringVarP(&showPosition, `position`, `p`, ``, `image position in cell coordinates <x>,<y>,<w>x<h>`)
	showCmd.Flags().StringVarP(&showURL, `url`, `u`, ``, `image url`)
	showCmd.Flags().StringVarP(&showFile, `file`, `f`, ``, `image path`)
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
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		run(showFunc(cmd, args))
	},
}

var (
	showCmdStr   = "show"
	showUsageStr = `usage: ` + os.Args[0] + ` ` + showCmdStr + ` -p <x>,<y>,<w>x<h> (-d <drawer>) (-t <tty>) <-f /path/to/image.png|-u http(s)://website.com/image.png>`
)

func showFunc(cmd *cobra.Command, args []string) terminalSwapper {
	return func(tm **term.Terminal) error {

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
			logFileOption,
			termimg.DefaultConfig,
			term.SetPTYName(ptyName),
			term.SetResizer(rsz),
		}
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2
		env := tm2.Environ()
		if len(showTTY) == 0 {
			env = append(env, os.Environ()...) // TODO rm when PS1 from inner shell included
		}

		var timg image.Image
		if len(showFile) > 0 {
			p, err := filepath.Abs(showFile)
			if logx.IsErr(err, tm2, slog.LevelError) {
				return errors.New(err)
			}
			showImageLocalPath = p
		} else if len(showURL) == 0 {
			if len(args) == 1 && len(args[0]) > 0 {
				if strings.HasPrefix(args[0], `https://`) || strings.HasPrefix(args[0], `http://`) {
					showURL = args[0]
				} else {
					p, err := filepath.Abs(args[0])
					if logx.IsErr(err, tm2, slog.LevelError) {
						return errors.New(err)
					}
					if _, err := os.Stat(p); !logx.IsErr(err, tm2, slog.LevelInfo) {
						showImageLocalPath = p
					} else {
						showURL = args[0]
					}
				}
			} else {
				return logx.Err(errors.New(`no image path specified`), tm2, slog.LevelError)
			}
		}
		if len(showImageLocalPath) > 0 {
			timg = termimg.NewImageFileName(showImageLocalPath)
		} else {
			m, err := downloadImage(showURL)
			if logx.IsErr(err, tm2, slog.LevelError) {
				return err
			}
			timg = m
		}
		if timg == nil {
			return logx.Err(errors.New(`nil image`), tm2, slog.LevelError)
		}
		{
			if timgTyped, ok := timg.(*term.Image); ok && timgTyped != nil {
				if err := timgTyped.Decode(); logx.IsErr(err, tm2, slog.LevelError) {
					return err
				}
			}
		}

		x, y, w, h, autoX, autoY, err := splitDimArg(showPosition, tm2, env, timg)
		if logx.IsErr(err, tm2, slog.LevelError) {
			return err
		}
		bounds := image.Rect(x, y, x+w, y+h)

		var dr term.Drawer
		if len(showDrawer) > 0 {
			dr = term.GetRegDrawerByName(showDrawer)
			if dr == nil {
				return logx.Err(errors.New(`unknown drawer "`+showDrawer+`"`), tm2, slog.LevelError)
			}
		} else if drawers := tm2.Drawers(); len(drawers) > 0 {
			dr = drawers[0]
		} else {
			return logx.Err(errors.New(`no drawer`), tm2, slog.LevelError)
		}
		if autoX && autoY {
			logx.IsErr(tm2.Scroll(0), tm2, slog.LevelInfo)
			logx.IsErr(tm2.SetCursor(0, 0), tm2, slog.LevelInfo)
		}
		coordWidth := 2
		gridBorderWidth := 3
		if showCoords {
			cutOff := coordWidth
			if !showGrid {
				cutOff += gridBorderWidth
			}
			tm2.WriteString(testutil.NumberArea(areaAddBorder(bounds, coordWidth+gridBorderWidth), cutOff))
		}
		if showGrid {
			tm2.WriteString(testutil.ChessPattern(areaAddBorder(bounds, gridBorderWidth), false))
		}
		if err := dr.Draw(timg, bounds, tm2); logx.IsErr(err, tm2, slog.LevelError) {
			return err
		}

		logx.IsErr(tm2.SetCursor(0, uint(bounds.Max.Y)+1), tm2, slog.LevelInfo)
		pauseVolatile(tm2, dr)

		return nil
	}
}

func isVolatileDrawer(tm *term.Terminal, dr term.Drawer) bool {
	if tm == nil {
		return false
	}
	if dr == nil {
		if drawers := tm.Drawers(); len(drawers) > 0 {
			dr = drawers[0]
		} else {
			if logger := tm.Logger(); logger != nil {
				logger.Error(`no drawer`)
			}
			return false
		}
	}
	isVolatileStr, isVolatile := tm.Property(propkeys.DrawerPrefix + dr.Name() + propkeys.DrawerVolatileSuffix)
	if !isVolatile || isVolatileStr != `true` {
		return false
	}
	return true
}

func pauseVolatile(tm *term.Terminal, dr term.Drawer) {
	if isVolatileDrawer(tm, dr) {
		if ptyName, okPTYName := tm.Property(propkeys.PTYName); okPTYName && internal.IsDefaultTTY(ptyName) {
			fi, err := os.Stdout.Stat()
			if !logx.IsErr(err, tm, slog.LevelInfo) && fi.Mode()&os.ModeNamedPipe != os.ModeNamedPipe {
				tm.WriteString(`press any key`)
				_, _ = os.Stdin.Read(make([]byte, 1)) // TODO read only 1 char
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func areaAddBorder(area image.Rectangle, diff int) image.Rectangle {
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
