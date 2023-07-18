package main

import (
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"

	errorsGo "github.com/go-errors/errors"
	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/resize/imaging"
	"github.com/srlehn/termimg/term"
)

func init() { rootCmd.AddCommand(showCmd) }

var showCmd = &cobra.Command{
	Use:   showCmdStr,
	Short: "display image",
	Long:  `display image`,
	Run: func(cmd *cobra.Command, args []string) {
		run(draw)
	},
}

var showCmdStr = "show"

var errShowUsage = errors.New(`usage: ` + os.Args[0] + ` ` + showCmdStr + ` /path/to/image.png <x>,<y>,<w>x<h> (drawer)`)

func draw() error {
	if l := len(os.Args); l < 4 || l > 5 {
		return errorsGo.New(errShowUsage)
	}
	imgFilename := os.Args[2]
	x, y, w, h, err := splitDimArg(os.Args[3])
	if err != nil {
		return errorsGo.New(errShowUsage)
	}
	bounds := image.Rect(x, y, x+w, y+h)
	defer termimg.CleanUp()
	if len(os.Args) == 5 {
		drawer := term.GetRegDrawerByName(os.Args[4])
		if drawer == nil {
			return errorsGo.New(`unknown drawer "` + os.Args[4] + `"`)
		}
		tm, err := termimg.Terminal()
		if err != nil {
			return err
		}
		if err := drawer.Draw(termimg.NewImageFileName(imgFilename), bounds, &imaging.Resizer{}, tm); err != nil {
			return err
		}
	} else {
		if err := termimg.DrawFile(imgFilename, bounds); err != nil {
			return err
		}
	}
	time.Sleep(2 * time.Second) // TODO
	return nil
}

func splitDimArg(dim string) (x, y, w, h int, e error) {
	dimParts := strings.Split(dim, `,`)
	if len(dimParts) > 5 {
		return 0, 0, 0, 0, errorsGo.New(`image position string not "<x>,<y>,<w>x<h>"`)
	}
	var err error
	for i, dimPart := range dimParts {
		if strings.Contains(dimPart, `x`) {
			if i != len(dimParts)-1 {
				return 0, 0, 0, 0, errorsGo.New(err)
			}
			sizes := strings.SplitN(dimPart, `x`, 2)
			w, err = strconv.Atoi(sizes[0])
			if err != nil {
				return 0, 0, 0, 0, errorsGo.New(err)
			}
			h, err = strconv.Atoi(sizes[1])
			if err != nil {
				return 0, 0, 0, 0, errorsGo.New(err)
			}
			break
		}
		var val int
		// default to 0
		if len(dimPart) > 0 {
			val, err = strconv.Atoi(dimPart)
			if err != nil {
				return 0, 0, 0, 0, errorsGo.New(err)
			}
		}
		switch i {
		case 0:
			x = val
		case 1:
			y = val
		case 2:
			w = val
		case 3:
			h = val
		}
	}
	if w == 0 || h == 0 {
		return 0, 0, 0, 0, errorsGo.New(`rectangle side with length 0`)
	}
	return x, y, w, h, nil
}
