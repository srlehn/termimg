package testutil

import (
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-errors/errors"

	"github.com/srlehn/termimg"
	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/encoder/encmulti"
	"github.com/srlehn/termimg/internal/exc"
	"github.com/srlehn/termimg/pty"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

func PTermPrintImageHelper(
	termName string,
	drawerName string,
	drawFuncProvider pty.DrawFuncProvider,
	imgBytes []byte,
	cellBounds image.Rectangle,
	fileName string,
	doDisplay bool,
) error {
	rsz := &rdefault.Resizer{}
	termProvider := func(ttyFile string) (*term.Terminal, error) {
		if len(ttyFile) == 0 {
			return termimg.Terminal()
		}
		wm.SetImpl(wmimpl.Impl())
		cr := &term.Options{
			PTYName:         ttyFile,
			TTYProvFallback: gotty.New,
			Querier:         qdefault.NewQuerier(),
			WindowProvider:  wm.NewWindow,
			Resizer:         rsz,
		}
		// return term.NewTerminal(cr)
		tm, err := term.NewTerminal(cr)
		return tm, err
	}
	img, err := pty.TakeScreenshot(termName, termProvider, drawerName, drawFuncProvider, imgBytes, cellBounds, rsz)
	if err != nil {
		return err
	}
	if img == nil {
		return errors.New(internal.ErrNilImage)
	}
	termNameStr := termName
	if len(termNameStr) == 0 {
		tn, okTN := img.(interface{ TerminalName() string })
		if okTN {
			termNameStr = tn.TerminalName()
		}
		if len(termNameStr) == 0 {
			termNameStr = `default`
		}
	}
	drawerNameStr := drawerName
	if len(drawerNameStr) == 0 {
		dn, okDN := img.(interface{ DrawerName() string })
		if okDN {
			drawerNameStr = dn.DrawerName()
		}
		if len(drawerNameStr) == 0 {
			drawerNameStr = `default`
		}
	}
	// if tn, okTN := img.(interface{ TerminalName() string }); okTN {termName = tn.TerminalName()}
	if len(fileName) == 0 {
		// git rev-parse --short HEAD
		var commit string
		gitAbs, err := exc.LookSystemDirs(`git`)
		if err == nil {
			commitBytes, err := exec.Command(gitAbs, `rev-parse`, `--short`, `HEAD`).Output()
			if err == nil {
				commit = `_` + strings.TrimSpace(string(commitBytes))
				dirtyBytes, err := exec.Command(gitAbs, `diff`, `HEAD`).Output()
				if err == nil {
					if len(dirtyBytes) > 0 {
						commit += `+dirty`
					}
				}
			}
		}
		tim := time.Now().Format(`_2006.01.02_15:04:05`)

		fileName = `screenshot_` + termNameStr + `_` + drawerNameStr + commit + tim + `.png`
	}
	f, err := os.Create(fileName)
	if err != nil {
		return errors.New(err)
	}
	defer f.Close()
	if err := (&encmulti.MultiEncoder{}).Encode(f, img, fileName); err != nil {
		_ = f.Close()
		_ = os.Remove(fileName)
		return err
	}
	if err := f.Close(); err != nil {
		return errors.New(err)
	}

	if doDisplay {
		conn, err := wm.NewConn()
		if err != nil {
			return err
		}
		defer conn.Close()
		conn.DisplayImage(img, `Screenshot of: "`+termName+`"`)
	}
	return nil
}

var (
	// waitingTimeDrawing = 3000 * time.Millisecond
	waitingTimeDrawing = 3000 * time.Millisecond
)

func DrawFuncOnlyPicture(img image.Image, cellBounds image.Rectangle) pty.DrawFunc {
	return func(tm *term.Terminal, dr term.Drawer, rsz term.Resizer, cpw, cph uint) (areaOfInterest image.Rectangle, scaleX, scaleY float64, e error) {
		if img == nil {
			return image.Rectangle{}, 0, 0, errors.New(internal.ErrNilImage)
		}
		if tm == nil || rsz == nil {
			return image.Rectangle{}, 0, 0, errors.New(internal.ErrNilParam)
		}
		if cpw == 0 || cph == 0 {
			return image.Rectangle{}, 0, 0, errors.New(`cell box side length of 0`)
		}
		if cellBounds.Dx() == 0 || cellBounds.Dy() == 0 {
			return image.Rectangle{}, 0, 0, errors.New(`area of size 0`)
		}

		imgSize := img.Bounds().Size()
		scaleX = float64(imgSize.X) / float64(int(cpw)*cellBounds.Dx())
		scaleY = float64(imgSize.Y) / float64(int(cph)*cellBounds.Dy())

		if err := term.DrawWith(img, cellBounds, dr, rsz, tm); err != nil {
			return image.Rectangle{}, 0, 0, err
		}
		time.Sleep(waitingTimeDrawing)

		return cellBounds, scaleX, scaleY, nil
	}
}

func DrawFuncPictureWithFrame(img image.Image, cellBounds image.Rectangle) pty.DrawFunc {
	return func(tm *term.Terminal, dr term.Drawer, rsz term.Resizer, cpw, cph uint) (areaOfInterest image.Rectangle, scaleX, scaleY float64, e error) {
		if img == nil {
			return image.Rectangle{}, 0, 0, errors.New(internal.ErrNilImage)
		}
		if tm == nil || rsz == nil {
			return image.Rectangle{}, 0, 0, errors.New(internal.ErrNilParam)
		}
		if cpw == 0 || cph == 0 {
			return image.Rectangle{}, 0, 0, errors.New(`cell box side length of 0`)
		}
		if cellBounds.Dx() == 0 || cellBounds.Dy() == 0 {
			return image.Rectangle{}, 0, 0, errors.New(`area of size 0`)
		}

		imgSize := img.Bounds().Size()
		scaleX = float64(imgSize.X) / float64(int(cpw)*cellBounds.Dx())
		scaleY = float64(imgSize.Y) / float64(int(cph)*cellBounds.Dy())

		areaOfInterest = cellBounds
		border := 3
		areaOfInterest.Min.X -= border
		areaOfInterest.Min.Y -= border
		areaOfInterest.Max.X += border
		areaOfInterest.Max.Y += border

		clearScreen := "\033c"
		background := clearScreen + ChessPattern(areaOfInterest, true) + infoHeader(tm, dr, areaOfInterest)

		// print background with regular characters
		if _, err := tm.Printf(`%s`, background); err != nil {
			return image.Rectangle{}, 0, 0, err
		}

		// draw on the terminal over the background
		if err := term.DrawWith(img, cellBounds, dr, rsz, tm); err != nil {
			return image.Rectangle{}, 0, 0, err
		}
		time.Sleep(waitingTimeDrawing)

		return areaOfInterest, scaleX, scaleY, nil
	}
}

func ChessPattern(area image.Rectangle, withBorder bool) string {
	// TODO don't use block char with xterm
	var areaInner image.Rectangle
	if withBorder {
		areaInner = image.Rectangle{
			Min: image.Point{X: area.Min.X + 1, Y: area.Min.Y + 1},
			Max: image.Point{X: area.Max.X - 1, Y: area.Max.Y - 1},
		}
		if areaInner.Dx() < 0 || areaInner.Dy() < 0 {
			return ``
		}
	} else {
		areaInner = area
	}
	if areaInner.Dx() == 0 || areaInner.Dy() == 0 {
		// area of size 0
		return ``
	}
	var (
		charFullBlock  = '▇'
		charWhiteSpace = ' '
	)
	lineWidthIsOdd := areaInner.Dx()%2 == 1
	heightIsOdd := areaInner.Dy()%2 == 1
	var lineChessPatternedHalf1, lineChessPatternedHalf2 []rune
	for i := 0; i < areaInner.Dx()/2; i++ {
		if i%2 == 0 {
			lineChessPatternedHalf1 = append(lineChessPatternedHalf1, charFullBlock)
			lineChessPatternedHalf2 = append(lineChessPatternedHalf2, charWhiteSpace)
		} else {
			lineChessPatternedHalf1 = append(lineChessPatternedHalf1, charWhiteSpace)
			lineChessPatternedHalf2 = append(lineChessPatternedHalf2, charFullBlock)
		}
	}
	lineChessPatterned1 := lineChessPatternedHalf1
	lineChessPatterned2 := lineChessPatternedHalf2
	if lineWidthIsOdd {
		if len(lineChessPatternedHalf1)%2 == 0 {
			lineChessPatterned1 = append(lineChessPatterned1, charFullBlock)
			lineChessPatterned2 = append(lineChessPatterned2, charWhiteSpace)
			lineChessPatterned1 = append(lineChessPatterned1, lineChessPatternedHalf2...)
			lineChessPatterned2 = append(lineChessPatterned2, lineChessPatternedHalf1...)
		} else {
			lineChessPatterned1 = append(lineChessPatterned1, charWhiteSpace)
			lineChessPatterned2 = append(lineChessPatterned2, charFullBlock)
			lineChessPatterned1 = append(lineChessPatterned1, lineChessPatternedHalf1...)
			lineChessPatterned2 = append(lineChessPatterned2, lineChessPatternedHalf2...)
		}
	} else {
		if len(lineChessPatternedHalf1)%2 == 0 {
			lineChessPatterned1 = append(lineChessPatterned1, lineChessPatternedHalf2...)
			lineChessPatterned2 = append(lineChessPatterned2, lineChessPatternedHalf1...)
		} else {
			lineChessPatterned1 = append(lineChessPatterned1, lineChessPatternedHalf1...)
			lineChessPatterned2 = append(lineChessPatterned2, lineChessPatternedHalf2...)
		}
	}
	lineChessPatterned1Str := string(lineChessPatterned1)
	lineChessPatterned2Str := string(lineChessPatterned2)
	var str, edgeHoriz string
	x := areaInner.Min.X
	y := areaInner.Min.Y
	if withBorder {
		x--
		edgeHoriz = strings.Repeat(`─`, areaInner.Dx())
		str += fmt.Sprintf("\033[%d;%dH", y, x+1) + `┌` + edgeHoriz + `┐`
		lineChessPatterned1Str = `│` + lineChessPatterned1Str + `│`
		lineChessPatterned2Str = `│` + lineChessPatterned2Str + `│`
	}
	for ; 2*(y-areaInner.Min.Y) < areaInner.Dy()-1; y++ {
		posStr := fmt.Sprintf("\033[%d;%dH", y+1, x+1)
		if (y-areaInner.Min.Y)%2 == 0 {
			str += posStr + lineChessPatterned1Str
		} else {
			str += posStr + lineChessPatterned2Str
		}
	}
	if heightIsOdd {
		posStr := fmt.Sprintf("\033[%d;%dH", y+1, x+1)
		if (y-areaInner.Min.Y)%2 == 0 {
			str += posStr + lineChessPatterned1Str
		} else {
			str += posStr + lineChessPatterned2Str
		}
		y++
	}
	for ; y < areaInner.Max.Y; y++ {
		posStr := fmt.Sprintf("\033[%d;%dH", y+1, x+1)
		if (areaInner.Max.Y-y)%2 == 0 {
			str += posStr + lineChessPatterned2Str
		} else {
			str += posStr + lineChessPatterned1Str
		}
	}
	if withBorder {
		str += fmt.Sprintf("\033[%d;%dH", area.Max.Y, x+1) + `└` + edgeHoriz + `┘`
	}

	return str
}

func infoHeader(tm *term.Terminal, dr term.Drawer, area image.Rectangle) string {
	if tm == nil {
		return ``
	}
	if dr == nil && len(tm.Drawers()) > 0 {
		dr = tm.Drawers()[0]
	}
	if dr == nil {
		return ``
	}
	str := tm.Name() + ` ` + dr.Name()

	areaWidth := area.Dx()
	strLen := utf8.RuneCountInString(str)
	if strLen > areaWidth {
		// TODO truncate
		return ``
	}

	posX := area.Min.X + 1 + (areaWidth-strLen)/2
	posStr := fmt.Sprintf("\033[%d;%dH", area.Min.Y+1, posX)
	str = posStr + str

	return str
}

func NumberArea(area image.Rectangle) string {
	countDecimals := func(d int) int {
		var sign int
		if d < 0 {
			d = -d
			sign = 1
		} else if d == 0 {
			return 1
		}
		return int(math.Log10(float64(d))) + 1 + sign
	}
	maxDecimalsX := countDecimals(area.Max.X)
	maxDecimalsY := countDecimals(area.Max.Y)
	maxDecimalsXStr := strconv.Itoa(maxDecimalsX)

	var line string
	for y := 0; y < maxDecimalsX; y++ {
		line += fmt.Sprintf("\033[%d;%dH", area.Min.Y+y+1, area.Min.X+maxDecimalsY+1)
		for x := area.Min.X + maxDecimalsY; x <= area.Max.X; x++ {
			decPot := int(math.Pow(float64(10), float64(maxDecimalsX-y-1)))
			digit := strconv.Itoa((x % (decPot * 10)) / decPot)
			if len(digit) != 1 || (digit == `0` && (maxDecimalsX-y) > countDecimals(x)) {
				digit = ` `
			}
			line += digit
		}
	}
	for y := area.Min.Y + maxDecimalsX + 1; y <= area.Max.Y; y++ {
		line += fmt.Sprintf("\033[%d;%dH%"+maxDecimalsXStr+`d`, y, area.Min.X+1, y-1)
	}
	return line
}
