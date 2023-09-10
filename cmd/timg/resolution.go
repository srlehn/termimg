package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/term"
)

var (
	resolutionFormat string
)

func init() {
	resolutionCmd.Flags().StringVarP(&resolutionFormat, `format`, `f`, ``, `format`)
	rootCmd.AddCommand(resolutionCmd)
}

var resolutionCmd = &cobra.Command{
	Use:   resolutionCmdStr,
	Short: `print terminal resolution`,
	Long: `print terminal resolution

` + resolutionUsageStr + `

    -f <fmt>, --format <fmt>    print format - %<letter> verbs are replaced:
                                %c    terminal resolution in cells (width)
                                %d    terminal resolution in cells (height)
                                %e    terminal resolution in pixels (width)
                                %f    terminal resolution in pixels (height)
                                %a    terminal cell resolution in pixels (width) (floating number)
                                %b    terminal cell resolution in pixels (height) (floating number)`,
	Args:             cobra.NoArgs,
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		run(resolutionFunc(cmd, args))
	},
}

var (
	resolutionCmdStr   = "resolution"
	resolutionUsageStr = `usage: ` + os.Args[0] + ` ` + resolutionCmdStr + ` (-f <format>)`
)

func resolutionFunc(cmd *cobra.Command, args []string) terminalSwapper {
	return func(tm **term.Terminal) error {
		tm2, err := termimg.Terminal()
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2
		fmtFlag := cmd.Flags().Lookup(`format`)
		customFormat := fmtFlag != nil && fmtFlag.Changed
		res := &resolution{}
		resCellsX, resCellsY, errResCells := tm2.SizeInCells()
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
		resPixelsX, resPixelsY, errResPixels := tm2.SizeInPixels()
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

		// TODO tm.Window().Size()

		cpw, cph, errCellRes := tm2.CellSize()
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
			if errCellRes == nil &&
				cpw >= 1 && cph >= 1 &&
				!math.IsNaN(cpw) && !math.IsNaN(cph) &&
				!math.IsInf(cpw, 0) && !math.IsInf(cph, 0) {
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
	resCellsNeeded         bool
	resPixelsX, resPixelsY uint
	errResPixels           error
	resPixelsNeeded        bool
	cpw, cph               float64
	errCellRes             error
	cellResNeeded          bool
}

func (r *resolution) Format(f fmt.State, c rune) {
	if r == nil {
		return
	}
	switch c {
	case 'c':
		r.resCellsNeeded = true
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resCellsX)
	case 'd':
		r.resCellsNeeded = true
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resCellsY)
	case 'e':
		r.resPixelsNeeded = true
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resPixelsX)
	case 'f':
		r.resPixelsNeeded = true
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`d`, r.resPixelsY)
		// TODO don't enforce precision
	case 'a':
		r.cellResNeeded = true
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`.02f`, r.cpw)
	case 'b':
		r.cellResNeeded = true
		fs := fmt.FormatString(f, c)
		fmt.Fprintf(f, `%`+fs[:len(fs)-2]+`.02f`, r.cph)
	}
}

func countFmtVerbs(s string) uint { return uint(strings.Count(strings.ReplaceAll(s, `%%`, ``), `%`)) }
