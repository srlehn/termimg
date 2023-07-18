package term

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
)

func SizeInCellsAndPixels(tty TTY) (widthCells, heightCells, widthPixels, heightPixels uint, err error) {
	sizerInCellsAndPixels, ok := tty.(interface {
		// ttyMattN github.com/mattn/go-tty
		SizePixel() (cx int, cy int, px int, py int, e error)
	})
	if !ok {
		return 0, 0, 0, 0, errors.New(`SizePixel() not implemented`)
	}
	cxi, cyi, pxi, pyi, err := sizerInCellsAndPixels.SizePixel()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if cxi < 0 || cyi < 0 || pxi < 0 || pyi < 0 {
		return 0, 0, 0, 0, errors.New(`negative integer`)
	}
	return uint(cxi), uint(cyi), uint(pxi), uint(pyi), err
}

// SizeInCellsQuery - dtterm window manipulation CSI 18 t
func SizeInCellsQuery(qu Querier, tty TTY) (widthCells, heightCells uint, e error) {
	// query terminal size in character boxes
	// answer: <termHeightInRows>;<termWidthInColumns>t
	qs := "\033[18t"
	/*if needsWrap {
		qs = mux.Wrap(qs)
	}*/

	repl, err := qu.Query(qs, tty, StopOnAlpha)
	if err != nil {
		return 0, 0, err
	}
	if len(repl) > 1 && repl[len(repl)-1] == 't' {
		repl = repl[:len(repl)-1]
	}
	q := strings.Split(repl, `;`)

	if len(q) != 3 {
		return 0, 0, errors.New(`unknown format`)
	}

	var x, y uint
	if yy, err := strconv.Atoi(string(q[1])); err == nil {
		if xx, err := strconv.Atoi(string(q[2])); err == nil {
			x = uint(xx)
			y = uint(yy)
		} else {
			return 0, 0, errors.New(err)
		}
	} else {
		return 0, 0, errors.New(err)
	}

	return x, y, nil
}

// SizeInPixelsQuery - dtterm window manipulation CSI 14 t
func SizeInPixelsQuery(qu Querier, tty TTY) (widthPixels, heightPixels uint, e error) {
	// query terminal size in pixels
	// answer: <termHeightInPixels>;<termWidthInPixels>t
	qs := "\033[14t"
	/*if needsWrap {
		qs = mux.Wrap(qs)
	}*/
	repl, err := qu.Query(qs, tty, StopOnAlpha)
	if err != nil {
		return 0, 0, err
	}
	if len(repl) > 1 && repl[len(repl)-1] == 't' {
		repl = repl[:len(repl)-1]
	}
	q := strings.Split(repl, `;`)

	if len(q) != 3 {
		return 0, 0, errors.New(`unknown format`)
	}

	var x, y uint
	if yy, err := strconv.Atoi(string(q[1])); err == nil {
		if xx, err := strconv.Atoi(string(q[2])); err == nil {
			x = uint(xx)
			y = uint(yy)
		} else {
			return 0, 0, errors.New(err)
		}
	} else {
		return 0, 0, errors.New(err)
	}

	return x, y, nil
}

// GetCursorQuery
func GetCursorQuery(qu Querier, tty TTY) (widthCells, heightCells uint, err error) {
	// query terminal position in cells
	// answer ?: ESC[<heightCells>;<heightCells>R // TODO
	// answer: !|<alnum>ESC\ESC[<heightCells>;<heightCells>R
	qs := "\033[6n"
	/*if needsWrap {
		qs = mux.Wrap(qs)
	}*/

	/*
	   example answers:
	   mlterm: !|000000\x1b\\\x1b[30;1R
	   terminator (vte): !|7E565445\x1b\\\x1b[48;1R"
	*/

	var stopOnR ParserFunc = func(r rune) bool { return r == 'R' }
	repl, err := qu.Query(qs, tty, stopOnR)
	if err != nil {
		return 0, 0, err
	}
	if len(repl) > 1 && repl[len(repl)-1] == 'R' {
		repl = repl[:len(repl)-1]
	}
	replEscParts := strings.Split(repl, "\033")
	repl = strings.TrimPrefix(replEscParts[len(replEscParts)-1], `[`)
	q := strings.Split(repl, `;`)

	if len(q) != 2 {
		return 0, 0, errors.New(`unknown format`)
	}

	var x, y uint
	if yy, err := strconv.Atoi(string(q[0])); err == nil {
		if xx, err := strconv.Atoi(string(q[1])); err == nil {
			x = uint(xx)
			y = uint(yy)
		} else {
			return 0, 0, errors.New(err)
		}
	} else {
		return 0, 0, errors.New(err)
	}

	return x, y, nil
}

// SetCursorQuery
func SetCursorQuery(widthCells, heightCells uint, qu Querier, tty TTY) (err error) {
	// set terminal position in cells
	// empty answer
	// alternatively \033[%d;%df // TODO
	qs := fmt.Sprintf("\033[%d;%dH", heightCells, widthCells)
	/*if needsWrap {
		qs = mux.Wrap(qs)
	}*/
	// _, err := qu.Query(qs, tty, nil) // empty answer
	_, err = tty.Write([]byte(qs))

	if err != nil {
		return err
	}

	return nil
}

// TODO save and restore position: ESC[s ESC[u
// TODO resize terminal: CSI 8;<lines>;<cols>t // for testing (for cropping & more accurate cell size)

////////////////////////////////////////////////////////////////////////////////

var _ PartialSurveyor = (*SurveyorDefault)(nil)
var _ PartialSurveyor = (*SurveyorNoANSI)(nil)
var _ PartialSurveyor = (*SurveyorNoTIOCGWINSZ)(nil)

type SurveyorDefault struct {
	// don't hold additional references to TTY, Querier, Windower, Proprietor, ...
	// let the caller pass it each time
}

// TODO listen to SIGWINCH and only query on new signal otherwise reply with old value
// TODO store terminal size with cell size and keep cell size of largest terminal size (highest accuracy)

func (s *SurveyorDefault) IsPartialSurveyor() {}

// SizeInCellsAndPixels ...
func (s *SurveyorDefault) SizeInCellsAndPixels(tty TTY) (widthCells, heightCells, widthPixels, heightPixels uint, err error) {
	return SizeInCellsAndPixels(tty)
}

// SizeInCellsQuery - dtterm window manipulation CSI 18 t
func (s *SurveyorDefault) SizeInCellsQuery(qu Querier, tty TTY) (widthCells, heightCells uint, e error) {
	return SizeInCellsQuery(qu, tty)
}

// SizeInPixelsQuery - dtterm window manipulation CSI 14 t
func (s *SurveyorDefault) SizeInPixelsQuery(qu Querier, tty TTY) (widthPixels, heightPixels uint, e error) {
	return SizeInPixelsQuery(qu, tty)
}

// GetCursorQuery
func (s *SurveyorDefault) GetCursorQuery(qu Querier, tty TTY) (widthCells, heightCells uint, err error) {
	return GetCursorQuery(qu, tty)
}

// SetCursorQuery
func (s *SurveyorDefault) SetCursorQuery(xPosCells, yPosCells uint, qu Querier, tty TTY) (err error) {
	return SetCursorQuery(xPosCells, yPosCells, qu, tty)
}

////////////////////////////////////////////////////////////////////////////////

// TODO rm: make linter shut up
var _ = (*SurveyorNoANSI)(nil)
var _ = (*SurveyorNoANSI)(nil).SizeInCellsAndPixels

type SurveyorNoANSI struct{}

func (s *SurveyorNoANSI) IsPartialSurveyor() {}

// SizeInCellsAndPixels ...
func (s *SurveyorNoANSI) SizeInCellsAndPixels(tty TTY) (widthCells, heightCells, widthPixels, heightPixels uint, err error) {
	return SizeInCellsAndPixels(tty)
}

////////////////////////////////////////////////////////////////////////////////

type SurveyorNoTIOCGWINSZ struct{}

func (s *SurveyorNoTIOCGWINSZ) IsPartialSurveyor() {}

// SizeInCellsQuery - dtterm window manipulation CSI 18 t
func (s *SurveyorNoTIOCGWINSZ) SizeInCellsQuery(qu Querier, tty TTY) (widthCells, heightCells uint, e error) {
	return SizeInCellsQuery(qu, tty)
}

// SizeInPixelsQuery - dtterm window manipulation CSI 14 t
func (s *SurveyorNoTIOCGWINSZ) SizeInPixelsQuery(qu Querier, tty TTY) (widthPixels, heightPixels uint, e error) {
	return SizeInPixelsQuery(qu, tty)
}

// xterm: cell size in pixels "\033[16t" -> "\033[6;<heightpx>;<widthpx>t"
// https://terminalguide.namepad.de/seq/csi_st-16/
