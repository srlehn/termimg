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

var (
	showDrawer   string
	showPosition string
	showImage    string
)

func init() {
	showCmd.PersistentFlags().StringVarP(&showDrawer, `drawer`, `d`, ``, `drawer to use`)
	showCmd.PersistentFlags().StringVarP(&showPosition, `position`, `p`, ``, `image position in cell coordinates <x>,<y>,<w>x<h>`)
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   showCmdStr,
	Short: "display image",
	Long:  `display image`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(showFunc(cmd, args))
	},
}

var showCmdStr = "show"

var errShowUsage = errors.New(`usage: ` + os.Args[0] + ` ` + showCmdStr + `-d <drawer> -p <x>,<y>,<w>x<h> /path/to/image.png`)

func showFunc(cmd *cobra.Command, args []string) func() error {
	return func() error {
		showImage = args[0]

		x, y, w, h, err := splitDimArg(showPosition)
		if err != nil {
			return errorsGo.New(errShowUsage)
		}
		bounds := image.Rect(x, y, x+w, y+h)

		defer termimg.CleanUp()
		if len(showDrawer) == 0 {
			if err := termimg.DrawFile(showImage, bounds); err != nil {
				return err
			}
		} else {
			drawer := term.GetRegDrawerByName(showDrawer)
			if drawer == nil {
				return errorsGo.New(`unknown drawer "` + showDrawer + `"`)
			}
			tm, err := termimg.Terminal()
			if err != nil {
				return err
			}
			if err := drawer.Draw(termimg.NewImageFileName(showImage), bounds, &imaging.Resizer{}, tm); err != nil {
				return err
			}

		}
		time.Sleep(2 * time.Second) // TODO rm
		return nil
	}
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
