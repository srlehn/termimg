package main

import (
	"image"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/spf13/cobra"
	"github.com/srlehn/thumbnails"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   listCmdStr,
	Short: `list images`,
	Long:  `list images and other previewable files`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(listFunc(cmd, args))
	},
}

var (
	listCmdStr = "list"
	// listUsageStr = `usage: ` + os.Args[0] + ` ` + listCmdStr
)

func listFunc(cmd *cobra.Command, args []string) func(**term.Terminal) error {
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

		tcw, _, err := tm2.SizeInCells()
		if err != nil {
			return err
		}
		_, cph, err := tm2.CellSize()
		if err != nil {
			return err
		}

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

		var imgCtr int
		handlePath := func(path string) (err error) {
			var (
				fi     fs.FileInfo
				img    image.Image
				bounds image.Rectangle
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
			var imgOffsetY int
			if bounds := img.Bounds(); bounds.Dx() > bounds.Dy() {
				// if image can't be resized - it will be cropped by fogleman/gg
				if rsz := tm2.Resizer(); rsz != nil {
					h := int(float64(tileBaseSize*bounds.Dy()) / float64(bounds.Dx()))
					imgOffsetY = (tileBaseSize - h) / 2
					m, err := rsz.Resize(img, image.Point{X: tileBaseSize, Y: h})
					if err == nil && m != nil {
						img = m
					}
				}
			}

			{
				offset := image.Point{
					X: (imgCtr % maxTilesX) * (szTile.X + 1),
					Y: (imgCtr / maxTilesX) * (szTile.Y + 1),
				}
				bounds = image.Rectangle{Min: offset, Max: offset.Add(szTile)}
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
			c.SetRGB(1, 1, 1)
			c.NewSubPath()
			c.Clear()
			c.SetRGB(0, 0, 0)
			c.DrawImage(img, 0, imgOffsetY)
			for i, line := range lines {
				if i >= maxLines {
					break
				}
				c.DrawString(line, 0, float64(tileBaseSize)+float64(i+1)*(c.FontHeight()+1))
			}
			c.Clip()
			img = c.Image()

			if err = tm2.Draw(img, bounds); err != nil {
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

		return nil
	}
}
