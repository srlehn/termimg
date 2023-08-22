//go:build dev && !windows && !android && !darwin && !js

package x11

import (
	"image"
	"time"

	"github.com/jezek/xgb/render"
	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xgraphics"
	"github.com/srlehn/xgbutil/xwindow"

	. "github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/term"
)

func init() {
	// draw = newDrawMethodDev
}

func newDrawMethodDev(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error {
	Must(render.Init(connXU.Conn()))
	Must(shm.Init(connXU.Conn()))
	// xproto.FreePixmap(connXU.Conn(), ximg.Pixmap)
	// render.FillRectanglesChecked(connXU.Conn(), render.RepeatNormal, )
	wdr := xproto.Drawable(w.Id)
	wimg := Must2(xgraphics.NewDrawable(connXU, wdr))
	_ = wimg
	ximg := xgraphics.NewConvert(connXU, timg.Cropped)
	//ximgpmp := Must2(xproto.NewPixmapId(connXU.Conn()))
	//Must(xproto.CreatePixmapChecked(connXU.Conn(), connXU.Screen().RootDepth, ximgpmp, wdr, uint16(boundsPx.Dx()), uint16(boundsPx.Dy())).Check())
	//ximg.Pixmap = ximgpmp

	// subimg := Must2(xproto.GetImage(connXU.Conn(), xproto.ImageFormatZPixmap, wdr, int16(boundsPx.Min.X), int16(boundsPx.Min.Y), uint16(boundsPx.Dx()), uint16(boundsPx.Dy()), 0).Reply())
	wimgsub := wimg.SubImage(boundsPx).(*xgraphics.Image)
	wimgsub.Destroy()

	// Must(wimg.CreatePixmap()) // TODO rm
	// Must(xproto.FreePixmapChecked(connXU.Conn(), wimg.Pixmap).Check())
	// Must(xproto.FreePixmapChecked(connXU.Conn(), wimg.SubImage(boundsPx).(*xgraphics.Image).Pixmap).Check())
	// if wimg.Pixmap == 0 {Must(wimg.CreatePixmap())}
	var seg shm.Seg
	if ximg.Pixmap == 0 {
		// p := Must2(xproto.NewPixmapId(connXU.Conn()))
		seg = Must2(shm.NewSegId(connXU.Conn()))
		Must(shm.CreatePixmapChecked(connXU.Conn(), ximg.Pixmap, wdr, uint16(ximg.Bounds().Dx()), uint16(ximg.Bounds().Dy()), connXU.Screen().RootDepth, seg, 0).Check())
		//Must(ximg.CreatePixmap())
	}
	wimgsub.Pix = ximg.Pix
	wimgsub.Stride = ximg.Stride
	wimgsub.Rect = ximg.Rect
	Must(ximg.XSurfaceSet(w.Id))
	Must(wimg.XSurfaceSet(w.Id))
	//ximg.XDraw()
	Must(wimgsub.CreatePixmap())
	wimgsub.XDraw()
	// Must(wimgsub.XDrawChecked())
	//ximg.XExpPaint(w.Id, boundsPx.Min.X, boundsPx.Min.Y)
	shm.PutImage(connXU.Conn(), wdr, connXU.GC(),
		uint16(ximg.Bounds().Dx()), uint16(ximg.Bounds().Dy()), uint16(ximg.Bounds().Min.X), uint16(ximg.Bounds().Min.Y),
		uint16(ximg.Bounds().Dx()), uint16(ximg.Bounds().Dy()), 50, 50,
		connXU.Screen().RootDepth, xproto.ImageFormatZPixmap, xproto.SendEventDestItemFocus, seg, 0)

	wimgsub.XExpPaint(w.Id, boundsPx.Min.X, boundsPx.Min.Y)
	// ximg.XExpPaint(w.Id, 10, 10)
	// _ = Must2(render.FillRectanglesChecked(connXU.Conn(), render.RepeatNormal, 0, render.Color{255, 0, 0, 255}, []xproto.Rectangle{{int16(boundsPx.Min.X), int16(boundsPx.Min.Y), uint16(boundsPx.Dx()), uint16(boundsPx.Dy())}}).Reply())
	// w.Clear(boundsPx.Min.X, boundsPx.Min.Y, boundsPx.Dx(), boundsPx.Dy())
	// wimg.SubImage(boundsPx).(*xgraphics.Image).Window(w.Id).ClearAll()
	// repl := Must2(xproto.CreatePixmapChecked(connXU.Conn(), 24, wimg.Pixmap, wdr, uint16(wimg.Bounds().Dx()), uint16(wimg.Bounds().Dy())).Reply()) // fails: Cannot call 'replyChecked' on a cookie that is not expecting a *reply* or an error.

	time.Sleep(1 * time.Second)
	return nil
}

// works in mate-terminal
func newDrawMethodDev2(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error {
	Must(render.Init(connXU.Conn()))
	ximg := xgraphics.NewConvert(connXU, timg.Cropped)
	_ = Must2(xgraphics.NewDrawable(connXU, xproto.Drawable(w.Id))) // without this image disappears directly
	if ximg.Pixmap == 0 {
		Must(ximg.CreatePixmap())
	}
	Must(ximg.XDrawChecked())
	ximg.XExpPaint(w.Id, 10, 10)

	time.Sleep(1 * time.Second)
	return nil
}

// works in xterm
func newDrawMethodDev3(connXU *xgbutil.XUtil, w *xwindow.Window, timg *term.Image, boundsPx image.Rectangle) error {
	Must(render.Init(connXU.Conn()))
	ximg := xgraphics.NewConvert(connXU, timg.Cropped)
	wdr := xproto.Drawable(w.Id)
	wimg := Must2(xgraphics.NewDrawable(connXU, wdr))
	_ = wimg
	if ximg.Pixmap == 0 {
		Must(ximg.CreatePixmap())
	}
	Must(ximg.XDrawChecked())
	ximg.XExpPaint(w.Id, 10, 10)
	ximg.Window(w.Id).Map() // Window() fails on mate-terminal

	time.Sleep(1 * time.Second)
	return nil
}
