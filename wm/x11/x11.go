package x11

import (
	"image"

	errorsGo "github.com/go-errors/errors"
	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xwindow"
	"github.com/srlehn/termimg/internal"
)

func AttachWindow(conn *xgbutil.XUtil, parent *xwindow.Window, pos image.Rectangle) (*xwindow.Window, error) {
	if conn == nil || parent == nil {
		return nil, errorsGo.New(internal.ErrNilParam)
	}
	w, err := xwindow.Generate(conn)
	if err != nil {
		return nil, errorsGo.New(err)
	}
	// https://stackoverflow.com/a/63219573
	if err := w.CreateChecked(conn.RootWin(), pos.Min.X, pos.Min.Y, pos.Dx(), pos.Dy(), xproto.CwOverrideRedirect, 1); err != nil {
		return nil, errorsGo.New(err)
	}
	if err := xproto.ReparentWindowChecked(conn.Conn(), w.Id, parent.Id, int16(pos.Min.X), int16(pos.Min.Y)).Check(); err != nil {
		return nil, errorsGo.New(err)
	}
	return w, nil
}
