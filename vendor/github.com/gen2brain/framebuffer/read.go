// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

package framebuffer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

var (
	endmode       = []byte("endmode")
	reg_label     = regexp.MustCompile(`^mode\W+"([^"]+)"`)
	reg_geometry  = regexp.MustCompile(`geometry\W+(\d+)\W+(\d+)\W+(\d+)\W+(\d+)\W+(\d+)`)
	reg_timings   = regexp.MustCompile(`timings\W+(\d+)\W+(\d+)\W+(\d+)\W+(\d+)\W+(\d+)\W+(\d+)\W+(\d+)`)
	reg_hsync     = regexp.MustCompile(`hsync\W+high`)
	reg_vsync     = regexp.MustCompile(`vsync\W+high`)
	reg_csync     = regexp.MustCompile(`csync\W+high`)
	reg_gsync     = regexp.MustCompile(`gsync\W+high`)
	reg_accel     = regexp.MustCompile(`accel\W+true`)
	reg_bcast     = regexp.MustCompile(`bcast\W+true`)
	reg_grayscale = regexp.MustCompile(`grayscale\W+true`)
	reg_extsync   = regexp.MustCompile(`extsync\W+true`)
	reg_nonstd    = regexp.MustCompile(`nonstd\W+(\d+)`)
	reg_laced     = regexp.MustCompile(`laced\W+true`)
	reg_double    = regexp.MustCompile(`double\W+true`)
	reg_format    = regexp.MustCompile(`rgba\W+(\d+)/(\d+),(\d+)/(\d+),(\d+)/(\d+),(\d+)/(\d+)`)
)

func readInt(v []byte, bits int) int {
	n, err := strconv.ParseInt(string(v), 10, bits)
	if err != nil {
		panic(err)
	}
	return int(n)
}

// readFBModes reads display mode data from the given stream.
// This is expected to come in the format defined at
// http://manned.org/fb.modes/81e6dc49
func readFBModes(r io.Reader) (list []*DisplayMode, err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("%v", x)
		}
	}()

	var line []byte

	rdr := bufio.NewReader(r)
	dm := new(DisplayMode)

	for {
		line, err = rdr.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// End of mode?
		if bytes.Index(line, endmode) > -1 {
			list = append(list, dm)
			dm = new(DisplayMode)
			continue
		}

		// Parse name
		matches := reg_label.FindSubmatch(line)
		if len(matches) > 1 {
			dm.Name = string(matches[1])
			continue
		}

		// Parse nonstd
		matches = reg_nonstd.FindSubmatch(line)
		if len(matches) > 1 {
			dm.Nonstandard = readInt(matches[1], 32)
			continue
		}

		// Parse hsync
		if reg_hsync.Match(line) {
			dm.Sync |= SyncHorHighAct
			continue
		}

		// Parse vsync
		if reg_vsync.Match(line) {
			dm.Sync |= SyncVertHighAct
			continue
		}

		// Parse csync
		if reg_csync.Match(line) {
			dm.Sync |= SyncCompHighAct
			continue
		}

		// Parse gsync
		if reg_gsync.Match(line) {
			dm.Sync |= SyncOnGreen
			continue
		}

		// Parse bcast
		if reg_bcast.Match(line) {
			dm.Sync |= SyncBroadcast
			continue
		}

		// Parse extsync
		if reg_extsync.Match(line) {
			dm.Sync |= SyncExt
			continue
		}

		// Parse accel
		if reg_accel.Match(line) {
			dm.Accelerated = true
			continue
		}

		// Parse grayscale
		if reg_grayscale.Match(line) {
			dm.Grayscale = true
			continue
		}

		// Parse laced
		if reg_laced.Match(line) {
			dm.VMode |= VModeInterlaced
			continue
		}

		// Parse double
		if reg_double.Match(line) {
			dm.VMode |= VModeDouble
			continue
		}

		// Parse geometry
		matches = reg_geometry.FindSubmatch(line)
		if len(matches) > 1 {
			dm.Geometry.XRes = readInt(matches[1], 32)
			dm.Geometry.YRes = readInt(matches[2], 32)
			dm.Geometry.XVRes = readInt(matches[3], 32)
			dm.Geometry.YVRes = readInt(matches[4], 32)
			dm.Geometry.Depth = readInt(matches[5], 32)
		}

		// Parse timings
		matches = reg_timings.FindSubmatch(line)
		if len(matches) > 1 {
			dm.Timings.Pixclock = readInt(matches[1], 32)
			dm.Timings.Left = readInt(matches[2], 32)
			dm.Timings.Right = readInt(matches[3], 32)
			dm.Timings.Upper = readInt(matches[4], 32)
			dm.Timings.Lower = readInt(matches[5], 32)
			dm.Timings.HSLen = readInt(matches[6], 32)
			dm.Timings.VSLen = readInt(matches[7], 32)
		}

		// Parse pixel format
		matches = reg_format.FindSubmatch(line)
		if len(matches) > 1 {
			dm.Format.RedBits = uint8(readInt(matches[1], 8))
			dm.Format.RedShift = uint8(readInt(matches[1], 8))
			dm.Format.GreenBits = uint8(readInt(matches[1], 8))
			dm.Format.GreenShift = uint8(readInt(matches[1], 8))
			dm.Format.BlueBits = uint8(readInt(matches[1], 8))
			dm.Format.BlueShift = uint8(readInt(matches[1], 8))
			dm.Format.AlphaBits = uint8(readInt(matches[1], 8))
			dm.Format.AlphaShift = uint8(readInt(matches[1], 8))
		}
	}

	return
}
