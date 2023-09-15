package main

import (
	"image"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/spf13/cobra"
	"github.com/srlehn/thumbnails"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

var (
	listDrawer string
)

func init() {
	listCmd.Flags().StringVarP(&listDrawer, `drawer`, `d`, ``, `drawer to use`)
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:              listCmdStr,
	Short:            `list images`,
	Long:             `list images and other previewable files`,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		run(listFunc(cmd, args))
	},
}

var (
	listCmdStr = "list"
	// listUsageStr = `usage: ` + os.Args[0] + ` ` + listCmdStr
)

func listFunc(cmd *cobra.Command, args []string) terminalSwapper {
	return func(tm **term.Terminal) error {
		paths := args
		if len(paths) == 0 {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			paths = []string{cwd}
		}

		wm.SetImpl(wmimpl.Impl())
		opts := []term.Option{
			termimg.DefaultConfig,
			term.SetPTYName(internal.DefaultTTYDevice()),
			term.SetResizer(&rdefault.Resizer{}),
		}
		var err error
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2

		var dr term.Drawer
		if len(listDrawer) > 0 {
			dr = term.GetRegDrawerByName(listDrawer)
			if dr == nil {
				return errors.New(`unknown drawer "` + listDrawer + `"`)
			}
		} else {
			dr = tm2.Drawers()[0]
		}

		tcw, tch, err := tm2.SizeInCells()
		if err != nil {
			return err
		}
		_, cph, err := tm2.CellSize()
		if err != nil {
			return err
		}
		var rowCursor uint
		_, rowCursor, errRowCursor := tm2.Cursor() // TODO log error
		if errRowCursor != nil {
			rowCursor = 0
		}

		fg, bg, _ := getForegroundBackground(tm2)

		maxLines := 3
		textHeight := int(math.Ceil(float64(maxLines) * cph))
		tileBaseSize := 128 // 128 is the "small" xdg thumbnail size
		tileWidth := tileBaseSize
		tileHeight := tileBaseSize + textHeight
		szTile, err := tm2.CellScale(image.Point{X: 128, Y: tileHeight}, image.Point{X: 0, Y: 0})
		if err != nil {
			return err
		}
		maxTilesX := int(float64(tcw) / float64(szTile.X+1))

		goFont, err := truetype.Parse(goregular.TTF)
		if err != nil {
			return err
		}
		goFontFace := truetype.NewFace(goFont, &truetype.Options{
			Size: 3 * (cph / 4), // convert from pixels to font points
		})
		defer goFontFace.Close()

		var (
			imgCtr int
			shifts int
			bounds image.Rectangle
		)
		handlePath := func(path string) (err error) {
			var (
				fi  fs.FileInfo
				img image.Image
			)
			path, err = filepath.Abs(path)
			if err != nil {
				return err
			}
			name := filepath.Base(path)
			fi, err = os.Stat(path)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			img, err = thumbnails.OpenThumbnail(path, image.Point{Y: tileBaseSize}, true)
			if err != nil {
				return err
			}
			var imgOffsetX int
			var imgOffsetY int
			{
				imgBounds := img.Bounds()
				dx := imgBounds.Dx()
				dy := imgBounds.Dy()
				if dx > dy {
					// if image can't be resized - it will be cropped by fogleman/gg
					if rsz := tm2.Resizer(); rsz != nil {
						h := int(float64(tileBaseSize*dy) / float64(dx))
						imgOffsetY = (tileBaseSize - h) / 2
						m, err := rsz.Resize(img, image.Point{X: tileBaseSize, Y: h})
						if err == nil && m != nil {
							img = m
						}
					}
				} else if dx < dy {
					if rsz := tm2.Resizer(); rsz != nil {
						w := int(float64(tileBaseSize*dx) / float64(dy))
						imgOffsetX = (tileBaseSize - w) / 2
						m, err := rsz.Resize(img, image.Point{X: w, Y: tileBaseSize})
						if err == nil && m != nil {
							img = m
						}
					}
				}
			}

			{
				offset := image.Point{
					X: (imgCtr % maxTilesX) * (szTile.X + 1),
					Y: (imgCtr/maxTilesX-shifts)*(szTile.Y+1) + int(rowCursor),
				}
				bounds = image.Rectangle{Min: offset, Max: offset.Add(szTile)}
				if bounds.Max.Y >= int(tch) {
					tm2.Scroll(szTile.Y + 1)
					shifts++
					offset.Y = (imgCtr/maxTilesX-shifts)*(szTile.Y+1) + int(rowCursor)
					bounds = image.Rectangle{Min: offset, Max: offset.Add(szTile)}
				}
			}
			c := gg.NewContext(tileWidth, tileHeight)
			c.SetFontFace(goFontFace)
			var lines []string
			var line []rune
			abbrChar := '…'
			for _, r := range name {
				lineNew := append(line, r)
				w, _ := c.MeasureString(string(lineNew))
				if w > float64(tileWidth) {
					if len(lines) == maxLines-1 {
						if len(line) >= 2 {
							line = append(line[:len(line)-2], []rune{abbrChar}...)
							if len(line) >= 3 && len(c.WordWrap(string(line), float64(tileWidth))) > 1 {
								line = append(line[:len(line)-3], []rune{abbrChar}...)
							}
						}
						lines = append(lines, string(line))
						break
					}
					lines = append(lines, string(line))
					line = []rune{r}
				} else {
					line = lineNew
				}
			}
			if len(line) >= 1 && line[len(line)-1] != abbrChar {
				lines = append(lines, string(line))
			}
			c.SetRGB(bg[0], bg[1], bg[2])
			c.Clear()
			c.SetRGB(fg[0], fg[1], fg[2])
			c.DrawImage(img, imgOffsetX, imgOffsetY)
			for i, line := range lines {
				if i >= maxLines {
					break
				}
				c.DrawString(line, 0, float64(tileBaseSize)+float64(i+1)*(c.FontHeight()+1))
			}
			c.Clip()
			img = c.Image()

			if err := term.Draw(img, bounds, tm2, dr); err != nil {
				goto end
			}
		end:
			imgCtr++
			return nil
		}
		for _, path := range paths {
			pathAbs, err := filepath.Abs(path)
			if err != nil {
				log.Println(err)
				continue
			}
			switch fi, err := os.Stat(pathAbs); {
			case err != nil:
			case !fi.IsDir():
				_ = handlePath(pathAbs)
			default:
				dirEntries, err := os.ReadDir(pathAbs)
				if err != nil {
					continue
				}
				for _, de := range dirEntries {
					_ = handlePath(filepath.Join(pathAbs, de.Name()))
				}
			}

		}

		_ = tm2.SetCursor(0, uint(bounds.Max.Y+1))
		pauseVolatile(tm2, dr)

		if errRowCursor != nil {
			return errRowCursor
		}
		return nil
	}
}

func getForegroundBackground(tm *term.Terminal) (fg, bg [3]float64, _ error) {
	fg = [3]float64{1, 1, 1} // default
	if tm == nil {
		return fg, bg, errors.New(`nil terminal`)
	}
	// DECSCNM - https://vt100.net/docs/vt510-rm/DECSCNM.html
	prs := parser.NewParser(false, true)
	replFG, err := tm.Query(queries.Foreground+queries.DA1, prs)
	if err != nil {
		return fg, bg, err
	}
	replBG, err := tm.Query(queries.Background+queries.DA1, prs)
	if err != nil {
		return fg, bg, err
	}
	parseRGB := func(s string) (rgb [3]float64, _ error) {
		parts := strings.SplitN(s, queries.ST, 2)
		if len(parts) < 2 {
			return rgb, errors.New(`no reply`)
		}
		s, okFG := strings.CutPrefix(parts[0], queries.OSC+"10;rgb:")
		s, okBG := strings.CutPrefix(s, queries.OSC+"11;rgb:")
		if !okFG && !okBG {
			return rgb, errors.New(`invalid reply`)
		}
		cols := strings.SplitN(s, `/`, 3)
		if len(cols) < 3 {
			return rgb, errors.New(`invalid reply`)
		}
		for i, col := range cols {
			h, err := strconv.ParseUint(col, 16, 64)
			if err != nil {
				return rgb, errors.New(err)
			}
			rgb[i] = float64(h) / float64(1<<16)
		}
		return rgb, nil
	}
	fg, err = parseRGB(replFG)
	if err != nil {
		return fg, bg, err
	}
	bg, err = parseRGB(replBG)
	if err != nil {
		return fg, bg, err
	}
	replRevVid, err := tm.Query(queries.ReverseVideo+queries.DA1, parser.StopOnC)
	if err != nil {
		return fg, bg, err
	}
	{
		parts := strings.SplitN(replRevVid, `$y`, 2)
		if len(parts) != 2 {
			return fg, bg, errors.New(`invalid reply`)
		}
		parts = strings.SplitN(parts[0], `;`, 2)
		if len(parts) != 2 {
			return fg, bg, errors.New(`invalid reply`)
		}
		if parts[1] == `1` || parts[1] == `3` {
			fg, bg = bg, fg
		}
	}
	return fg, bg, nil
}
