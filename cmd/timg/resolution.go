package main

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
)

var (
	resolutionFormat string
)

func init() {
	resolutionCmd.PersistentFlags().StringVarP(&resolutionFormat, `format`, `f`, ``, `format`)
	rootCmd.AddCommand(resolutionCmd)
}

var resolutionCmd = &cobra.Command{
	Use:   resolutionCmdStr,
	Short: `print terminal resolution`,
	Long: `print terminal resolution

    -f <fmt>, --format <fmt>    print format - %<letter> verbs are replaced:
                                %c    terminal resolution in cells (width)
                                %d    terminal resolution in cells (height)
                                %e    terminal resolution in pixels (width)
                                %f    terminal resolution in pixels (height)
                                %c    terminal cell resolution in pixels (width) (floating number)
                                %d    terminal cell resolution in pixels (height) (floating number)`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		run(resolutionFunc(cmd, args))
	},
}

var resolutionCmdStr = "resolution"

func resolutionFunc(cmd *cobra.Command, args []string) func() error {
	return func() error {
		tm, err := termimg.Terminal()
		if err != nil {
			return err
		}
		defer tm.Close()
		fmtFlag := cmd.Flags().Lookup(`format`)
		customFormat := fmtFlag != nil && fmtFlag.Changed
		res := &resolution{}
		resCellsX, resCellsY, errResCells := tm.SizeInCells()
		if customFormat {
			res.errResCells = errResCells
			if errResCells == nil {
				res.resCellsX = resCellsX
				res.resCellsY = resCellsY
			}
		} else {
			if errResCells == nil && resCellsX != 0 && resCellsY != 0 {
				fmt.Printf("terminal resolution:      %dx%d (cells)\n", resCellsX, resCellsY)
			}
		}
		resPixelsX, resPixelsY, errResPixels := tm.SizeInPixels()
		if customFormat {
			res.errResPixels = errResPixels
			if errResPixels == nil {
				res.resPixelsX = resPixelsX
				res.resPixelsY = resPixelsY
			}
		} else {
			if errResPixels == nil && resPixelsX != 0 && resPixelsY != 0 {
				fmt.Printf("terminal resolution:      %dx%d (pixels)\n", resPixelsX, resPixelsY)
			}
		}
		cpw, cph, errCellRes := tm.CellSize()
		if customFormat {
			res.errCellRes = errCellRes
			if errCellRes == nil {
				if cpw >= 0 && !math.IsNaN(cpw) {
					res.cpw = cpw
				}
				if cph >= 0 && !math.IsNaN(cph) {
					res.cph = cph
				}
			}

			var resRepeats []any
			cnt := int(countFmtVerbs(resolutionFormat))
			for i := 0; i < cnt; i++ {
				resRepeats = append(resRepeats, res)
			}
			resolutionFormat = strings.NewReplacer(
				`\\`, "\\",
				`\t`, "\t",
				`\n`, "\n",
				`\r`, "\r",
			).Replace(resolutionFormat)
			fmt.Printf(resolutionFormat, resRepeats...)
		} else {
			if errCellRes == nil && cpw >= 1 && cph >= 1 && !math.IsNaN(cpw) && !math.IsNaN(cph) {
				fmt.Printf("terminal cell resolution: %.2fx%.2f (pixels)\n", cpw, cph)
			}
		}
		err = errors.Join(errResCells, errResPixels, errCellRes)
		return err
	}
}

type resolution struct {
	resCellsX, resCellsY   uint
	errResCells            error
	resPixelsX, resPixelsY uint
	errResPixels           error
	cpw, cph               float64
	errCellRes             error
}

func (r *resolution) Format(f fmt.State, c rune) {
	switch c {
	case 'c':
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resCellsX)
	case 'd':
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resCellsY)
	case 'e':
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resPixelsX)
	case 'f':
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resPixelsY)
		// TODO don't enforce precision
	case 'a':
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`.02f`, r.cpw)
	case 'b':
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`.02f`, r.cph)
	}
}

func countFmtVerbs(s string) uint { return uint(strings.Count(strings.ReplaceAll(s, `%%`, ``), `%`)) }
