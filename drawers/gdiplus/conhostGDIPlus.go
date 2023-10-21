//go:build windows

package gdiplus

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"time"

	"github.com/lxn/win"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/internal/wndws"
	"github.com/srlehn/termimg/term"
)

func init() { term.RegisterDrawer(&drawerGDI{}) }

var _ term.Drawer = (*drawerGDI)(nil)

type drawerGDI struct {
	gdiIsStarted bool
	cleanUps     []func() error
}

func (d *drawerGDI) Name() string     { return `conhost_gdi` }
func (d *drawerGDI) New() term.Drawer { return &drawerGDI{} }
func (d *drawerGDI) IsApplicable(inp term.DrawerCheckerInput) (bool, environ.Properties) {
	return inp != nil && inp.Name() == `conhost` && !wndws.RunsOnWine(), nil
}
func (d *drawerGDI) init() error {
	if d == nil {
		return errors.New(consts.ErrNilReceiver)
	}
	var si win.GdiplusStartupInput
	si.GdiplusVersion = 1
	if status := win.GdiplusStartup(&si, nil); status != win.Ok {
		return errors.New(fmt.Sprintf("GdiplusStartup failed with status '%s'", status))
	}
	d.gdiIsStarted = true
	d.cleanUps = append(d.cleanUps, func() error { win.GdiplusShutdown(); return nil })
	return nil
}
func (d *drawerGDI) Close() error {
	if d == nil {
		return nil
	}
	var errs []error
	for i := len(d.cleanUps) - 1; i > -1; i-- {
		cl := d.cleanUps[i]
		if cl == nil {
			continue
		}
		err := cl()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
func (d *drawerGDI) Draw(img image.Image, bounds image.Rectangle, tm *term.Terminal) error {
	drawFn, err := d.Prepare(context.Background(), img, bounds, tm)
	if err != nil {
		return err
	}
	return logx.TimeIt(drawFn, `image drawing`, tm, `drawer`, d.Name())
}

func (d *drawerGDI) Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, tm *term.Terminal) (drawFn func() error, _ error) {
	if d == nil || tm == nil || img == nil || ctx == nil {
		return nil, errors.New(`nil parameter`)
	}
	start := time.Now()
	if err := d.init(); err != nil {
		return nil, err
	}
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
	err := timg.Fit(bounds, rsz, tm)
	if err != nil {
		return nil, err
	}

	var (
		spTyped *GDIImage
		gpBmp   *wndws.GpBitmap
		hBmp    win.HBITMAP
		hdcMem  win.HDC
		termDC  win.HDC
		hBmpOld win.HGDIOBJ
		close   func() error
	)

	sp, err := timg.DrawerObject(d)
	if err != nil {
		return nil, err
	}
	if sp != nil {
		spt, okTyped := sp.(*GDIImage)
		if !okTyped || spt == nil {
			return nil, errors.New(fmt.Sprintf(`term.DrawerSpec[%s] (%T) is nil or not of type %T`, d.Name(), sp, &GDIImage{}))
		}
		spTyped = spt
		goto draw
	} else {
		spTyped = &GDIImage{}
	}

createBitmap:
	{
		if len(timg.Encoded) > 0 {
			gpBmp, close, err = wndws.GetGpBitmapFromBytes(timg.Encoded)
			if err != nil {
				return nil, err
			}
			if close != nil {
				timg.OnClose(close)
			}
		} else if len(timg.FileName) > 0 {
			gpBmp, err = wndws.GetGpBitmapFromFile(timg.FileName)
			if err != nil {
				return nil, err
			}
		} else {
			bytBuf := new(bytes.Buffer)
			if err = png.Encode(bytBuf, timg.Cropped); err != nil {
				return nil, errors.New(err)
			}
			timg.Encoded = bytBuf.Bytes()
			if len(timg.Encoded) == 0 {
				return nil, errors.New(`image encoding failed`)
			}
			goto createBitmap
		}
		timg.OnClose(func() error { _ = gpBmp.Dispose(); return nil })
		spTyped.gpBmp = gpBmp

		// create HBITMAP
		if status := win.GdipCreateHBITMAPFromBitmap((*win.GpBitmap)(gpBmp), &hBmp, 0); status != win.Ok {
			return nil, errors.New(fmt.Sprintf("GdipCreateHBITMAPFromBitmap failed with status '%s'", status))
		}
		timg.OnClose(func() error { _ = win.DeleteObject(win.HGDIOBJ(hBmp)); return nil })
		spTyped.hBitmap = &hBmp

		termDC = win.HDC(tm.Window().DeviceContext())
		// create device context
		hdcMem = win.CreateCompatibleDC(termDC)
		hBmpOld = win.SelectObject(hdcMem, win.HGDIOBJ(hBmp))
		if hBmpOld == 0 {
			return nil, errors.New("SelectObject failed")
		}
		timg.OnClose(func() error { _ = win.SelectObject(hdcMem, hBmpOld); return nil })
		timg.OnClose(func() error { _ = win.ReleaseDC(0, hdcMem); return nil })
		spTyped.dc = hdcMem

		// timg.DrawerSpec[d.Name()] = spTyped // TODO rm
		if err := timg.SetPosObject(image.Rectangle{}, spTyped, d, tm); err != nil {
			return nil, err
		}
	}

draw:
	// TODO: win.HALFTONE or win.COLORONCOLOR?
	if 0 == win.SetStretchBltMode(termDC, win.COLORONCOLOR) {
		return nil, errors.New("SetStretchBltMode")
	}

	logx.Info(`image preparation`, tm, `drawer`, d.Name(), `duration`, time.Since(start))

	drawFn = func() error {
		if !win.StretchBlt(
			termDC, int32(bounds.Min.X*8), int32(bounds.Min.Y*16), int32(bounds.Dx()*8), int32(bounds.Dy()*16),
			spTyped.dc, int32(0), int32(0), int32(timg.Bounds().Dx()), int32(timg.Bounds().Dy()),
			win.SRCCOPY,
		) {
			return logx.Err(errors.New("StretchBlt failed"), tm, slog.LevelInfo)
		}
		return nil
	}

	return drawFn, nil
}

type GDIImage struct {
	encoded []byte
	gpBmp   *wndws.GpBitmap
	hBitmap *win.HBITMAP
	dc      win.HDC
}
