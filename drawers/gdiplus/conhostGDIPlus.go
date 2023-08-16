//go:build windows

package gdiplus

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/wndws"
	"github.com/srlehn/termimg/term"

	errorsGo "github.com/go-errors/errors"
	"github.com/lxn/win"
)

func init() { term.RegisterDrawer(&drawerGDI{}) }

var _ term.Drawer = (*drawerGDI)(nil)

type drawerGDI struct {
	gdiIsStarted bool
	cleanUps     []func() error
}

func (d *drawerGDI) Name() string     { return `conhost_gdi` }
func (d *drawerGDI) New() term.Drawer { return &drawerGDI{} }
func (d *drawerGDI) IsApplicable(inp term.DrawerCheckerInput) bool {
	return inp != nil && inp.Name() == `conhost` && !wndws.RunsOnWine()
}
func (d *drawerGDI) init() error {
	if d == nil {
		return errorsGo.New(internal.ErrNilReceiver)
	}
	var si win.GdiplusStartupInput
	si.GdiplusVersion = 1
	if status := win.GdiplusStartup(&si, nil); status != win.Ok {
		return errorsGo.New(fmt.Sprintf("GdiplusStartup failed with status '%s'", status))
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
	if d == nil || tm == nil || img == nil {
		return errorsGo.New(`nil parameter`)
	}
	if err := d.init(); err != nil {
		return err
	}
	timg, ok := img.(*term.Image)
	if !ok {
		timg = term.NewImage(img)
	}
	if timg == nil {
		return errorsGo.New(internal.ErrNilImage)
	}

	rsz := tm.Resizer()
	if rsz == nil {
		return errorsGo.New(`nil resizer`)
	}
	err := timg.Fit(bounds, rsz, tm)
	if err != nil {
		return err
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

	sp, err := timg.GetDrawerObject(d)
	if err != nil {
		return err
	}
	if sp != nil {
		spt, okTyped := sp.(*GDIImage)
		if !okTyped || spt == nil {
			return errorsGo.New(fmt.Sprintf(`term.DrawerSpec[%s] (%T) is nil or not of type %T`, d.Name(), sp, &GDIImage{}))
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
				return err
			}
			if close != nil {
				timg.OnClose(close)
			}
		} else if len(timg.FileName) > 0 {
			gpBmp, err = wndws.GetGpBitmapFromFile(timg.FileName)
			if err != nil {
				return err
			}
		} else {
			bytBuf := new(bytes.Buffer)
			if err = png.Encode(bytBuf, timg.Cropped); err != nil {
				return errorsGo.New(err)
			}
			timg.Encoded = bytBuf.Bytes()
			if len(timg.Encoded) == 0 {
				return errorsGo.New(`image encoding failed`)
			}
			goto createBitmap
		}
		timg.OnClose(func() error { _ = gpBmp.Dispose(); return nil })
		spTyped.gpBmp = gpBmp

		// create HBITMAP
		if status := win.GdipCreateHBITMAPFromBitmap((*win.GpBitmap)(gpBmp), &hBmp, 0); status != win.Ok {
			return errorsGo.New(fmt.Sprintf("GdipCreateHBITMAPFromBitmap failed with status '%s'", status))
		}
		timg.OnClose(func() error { _ = win.DeleteObject(win.HGDIOBJ(hBmp)); return nil })
		spTyped.hBitmap = &hBmp

		termDC = win.HDC(tm.Window().DeviceContext())
		// create device context
		hdcMem = win.CreateCompatibleDC(termDC)
		hBmpOld = win.SelectObject(hdcMem, win.HGDIOBJ(hBmp))
		if hBmpOld == 0 {
			return errorsGo.New("SelectObject failed")
		}
		timg.OnClose(func() error { _ = win.SelectObject(hdcMem, hBmpOld); return nil })
		timg.OnClose(func() error { _ = win.ReleaseDC(0, hdcMem); return nil })
		spTyped.dc = hdcMem

		// timg.DrawerSpec[d.Name()] = spTyped // TODO rm
		if err := timg.SetPosObject(image.Rectangle{}, spTyped, d, tm); err != nil {
			return err
		}
	}

draw:
	// TODO: win.HALFTONE or win.COLORONCOLOR?
	if 0 == win.SetStretchBltMode(termDC, win.COLORONCOLOR) {
		return errorsGo.New("SetStretchBltMode")
	}
	if !win.StretchBlt(
		termDC, int32(bounds.Min.X*8), int32(bounds.Min.Y*16), int32(bounds.Dx()*8), int32(bounds.Dy()*16),
		spTyped.dc, int32(0), int32(0), int32(timg.Bounds().Dx()), int32(timg.Bounds().Dy()),
		win.SRCCOPY,
	) {
		return errorsGo.New("StretchBlt failed")
	}

	return nil
}

type GDIImage struct {
	encoded []byte
	gpBmp   *wndws.GpBitmap
	hBitmap *win.HBITMAP
	dc      win.HDC
}
