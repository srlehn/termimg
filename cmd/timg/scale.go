package main

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
)

func init() { rootCmd.AddCommand(scaleCmd) }

var scaleCmd = &cobra.Command{
	Use:   scaleCmdStr,
	Short: `fit pixel area into a cell area while maintaining scale`,
	Long: `Fit pixel area into a cell area while maintaining scale.

` + scaleUsageStr + `

With no passed 0 side length values, the largest subarea is returned.
With one passed 0 side length value, the other side length will be fixed.
With two passed 0 side length values, pixels in source and destination area at the same position correspond to each other.`,
	Args:             cobra.ExactArgs(2),
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		run(scaleFunc(cmd, args))
	},
}

var (
	scaleCmdStr   = "scale"
	scaleUsageStr = `usage: ` + os.Args[0] + ` ` + scaleCmdStr + ` <srcSizePixels(<w>x<h>)> <dstSizeCells(<w>x<h>)>`
	errScaleUsage = errors.New(scaleUsageStr)
)

func scaleFunc(cmd *cobra.Command, args []string) terminalSwapper {
	return func(tm **term.Terminal) error {
		opts := []term.Option{
			logFileOption,
			termimg.DefaultConfig,
		}
		tm2, err := term.NewTerminal(opts...)
		if err != nil {
			return err
		}
		defer tm2.Close()
		*tm = tm2
		srcSizePixelsParts := strings.SplitN(args[0], `x`, 2)
		if len(srcSizePixelsParts) != 2 {
			return logx.Err(errScaleUsage, tm2, slog.LevelError)
		}
		var (
			srcSizePixelsW uint64
			srcSizePixelsH uint64
			dstSizeCellsW  uint64
			dstSizeCellsH  uint64
		)
		if len(srcSizePixelsParts[0]) > 0 {
			srcSizePixelsW, err = strconv.ParseUint(srcSizePixelsParts[0], 10, 64)
			if err != nil {
				return logx.Err(errScaleUsage, tm2, slog.LevelError)
			}
		}
		if len(srcSizePixelsParts[1]) > 0 {
			srcSizePixelsH, err = strconv.ParseUint(srcSizePixelsParts[1], 10, 64)
			if err != nil {
				return logx.Err(errScaleUsage, tm2, slog.LevelError)
			}
		}
		dstSizeCellsParts := strings.SplitN(args[1], `x`, 2)
		if len(dstSizeCellsParts) != 2 {
			return logx.Err(errScaleUsage, tm2, slog.LevelError)
		}
		if len(dstSizeCellsParts[0]) > 0 {
			dstSizeCellsW, err = strconv.ParseUint(dstSizeCellsParts[0], 10, 64)
			if err != nil {
				return logx.Err(errScaleUsage, tm2, slog.LevelError)
			}
		}
		if len(dstSizeCellsParts[1]) > 0 {
			dstSizeCellsH, err = strconv.ParseUint(dstSizeCellsParts[1], 10, 64)
			if err != nil {
				return logx.Err(errScaleUsage, tm2, slog.LevelError)
			}
		}
		ptSrcPx := image.Point{X: int(srcSizePixelsW), Y: int(srcSizePixelsH)}
		ptDstCl := image.Point{X: int(dstSizeCellsW), Y: int(dstSizeCellsH)}
		ptScaledCl, err := tm2.CellScale(ptSrcPx, ptDstCl)
		if logx.IsErr(err, tm2, slog.LevelError) {
			return err
		}
		fmt.Printf("%dx%d", ptScaledCl.X, ptScaledCl.Y)
		return nil
	}
}
