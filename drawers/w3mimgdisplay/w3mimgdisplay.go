package w3mimgdisplay

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xgraphics"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/encoder/encpng"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/wm"
)

func init() { term.RegisterDrawer(&drawerW3MImgDisplay{}) }

var _ term.Drawer = (*drawerW3MImgDisplay)(nil)

var (
	exeW3MImgDisplayCommonPaths = []string{
		`/usr/lib/w3m/w3mimgdisplay`,
		`/usr/libexec/w3m/w3mimgdisplay`,
	}
	exeW3MImgDisplay = exeW3MImgDisplayCommonPaths[0]
)

type drawerW3MImgDisplay struct{}

func (d *drawerW3MImgDisplay) Name() string     { return `w3mimgdisplay` }
func (d *drawerW3MImgDisplay) New() term.Drawer { return &drawerW3MImgDisplay{} }

func (d *drawerW3MImgDisplay) IsApplicable(inp term.DrawerCheckerInput) (bool, term.Properties) {
	if inp == nil {
		return false, nil
	}
	// systemd: XDG_SESSION_TYPE == x11
	sessionType, okST := inp.LookupEnv(`XDG_SESSION_TYPE`)
	if okST && sessionType != `x11` {
		// might be `tty`, `wayland`, ...
		return false, nil
	}
	switch inp.Name() {
	case `conhost`,
		`alacritty`,
		`vscode`:
		return false, nil
	case `vte`:
		if _, ok := inp.LookupEnv(`TERMINATOR_UUID`); ok {
			return false, nil
		}
		if _, ok := inp.LookupEnv(`TERMINATOR_DBUS_NAME`); ok {
			return false, nil
		}
		if _, ok := inp.LookupEnv(`TERMINATOR_DBUS_PATH`); ok {
			return false, nil
		}
	}
	for _, pth := range exeW3MImgDisplayCommonPaths {
		fi, err := os.Stat(pth)
		existsAndExecutable := err == nil && fi != nil && fi.Mode()&0b001 == 0b001
		if existsAndExecutable {
			exeW3MImgDisplay = pth
			return true, nil
		}
	}
	return false, nil
}

func (d *drawerW3MImgDisplay) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerW3MImgDisplay) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
	if d == nil || tm == nil || img == nil || ctx == nil {
		return nil, errors.New(`nil parameter`)
	}
	start := time.Now()
	var (
		termOffSet image.Point
		cpw, cph   float64
	)
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return nil, errors.New(consts.ErrNilImage)
	}

	rsz := tm.Resizer()
	if rsz == nil {
		return nil, errors.New(`nil resizer`)
	}
	if err := timg.Fit(bounds, rsz, tm); err != nil {
		return nil, err
	}

	var w3mImgDisplayString string
	w3mImgDisplayObject, err := timg.PosObject(bounds, d, tm)
	if err == nil {
		s, ok := w3mImgDisplayObject.(string)
		if ok {
			w3mImgDisplayString = s
			goto exc
		}
	}

	cpw, cph, err = tm.CellSize()
	if err != nil {
		return nil, err
	}

	_, err = timg.SaveAsFile(tm, `png`, &encpng.PngEncoder{})
	if err != nil {
		return nil, err
	}

	// trying to get window size
	{
		conn, err := wm.NewConn(tm)
		if err != nil {
			goto skipFindingTermOffSet
		}
		connXU, okXU := conn.Conn().(*xgbutil.XUtil)
		if !okXU {
			return nil, errors.New(consts.ErrPlatformNotSupported)
		}
		if connXU == nil {
			return nil, errors.New(`nil connection`)
		}
		termName := tm.Name()
		tChk := term.RegisteredTermChecker(termName)
		if tChk == nil {
			goto skipFindingTermOffSet
		}
		var windowsMatching []wm.Window
		windows, err := conn.Windows()
		if err != nil {
			return nil, err
		}
		for _, wdw := range windows {
			if is, _ := tChk.CheckIsWindow(wdw); !is {
				continue
			}
			windowsMatching = append(windowsMatching, wdw)
		}
		if len(windowsMatching) != 1 {
			goto skipFindingTermOffSet
		}
		w := windowsMatching[0]
		ximg, err := xgraphics.NewDrawable(connXU, xproto.Drawable(w.WindowID()))
		if err != nil {
			goto skipFindingTermOffSet
		}
		wpw := ximg.Bounds().Dx()
		wph := ximg.Bounds().Dy()
		tpw, tph, err := tm.SizeInPixels()
		if err != nil {
			goto skipFindingTermOffSet
		}
		edgeWidth := (wpw - int(tpw)) / 2
		menuBarHeight := wph - int(tph) - edgeWidth
		if edgeWidth > 0 {
			termOffSet.X = edgeWidth
		}
		if menuBarHeight > 0 {
			termOffSet.Y = menuBarHeight
		}
	}
skipFindingTermOffSet:

	{
		imgOffSet := termOffSet
		imgOffSet = imgOffSet.Add(image.Pt(
			int(float64(bounds.Min.X)*cpw),
			int(float64(bounds.Min.Y)*cph),
		))
		areaWidth := int(float64(bounds.Dx()) * cpw)
		areaHeight := int(float64(bounds.Dy()) * cph)
		w3mImgDisplayString = fmt.Sprintf(
			// "0;1;%d;%d;%d;%d;;;%[3]d;%[4]d;%s\n4;\n3;",
			"0;1;%d;%d;%d;%d;;;;;%s\n4;\n3;",
			imgOffSet.X,
			imgOffSet.Y,
			areaWidth,
			areaHeight,
			timg.FileName,
		)
	}

exc:
	timg.SetPosObject(bounds, w3mImgDisplayString, d, tm)

	logx.Debug(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		cmd := exec.Command(exeW3MImgDisplay)
		cmd.Stdin = strings.NewReader(w3mImgDisplayString)
		err := cmd.Run()
		return logx.Err(err, tm, slog.Level(slog.LevelInfo))
	}

	return drawFn, nil
}
