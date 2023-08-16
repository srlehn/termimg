package x11

import (
	"image"
	"strings"

	errorsGo "github.com/go-errors/errors"
	"github.com/jezek/xgb/xproto"
	"github.com/srlehn/xgbutil"
	"github.com/srlehn/xgbutil/xprop"
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

func XResources(conn *xgbutil.XUtil) ([][2]string, error) {
	if conn == nil {
		return nil, errorsGo.New(`nil window connection`)
	}
	// xproto.AtomResourceManager = 23
	resMgrStr, err := xprop.AtomName(conn, xproto.AtomResourceManager) // "RESOURCE_MANAGER"
	if err != nil {
		return nil, errorsGo.New(err)
	}
	resMgrProp, err := xprop.GetProperty(conn, conn.RootWin(), resMgrStr)
	if err != nil {
		return nil, errorsGo.New(err)
	}
	resources := strings.Split(string(resMgrProp.Value), "\n")
	var xRes [][2]string
	for _, res := range resources {
		if len(res) == 0 {
			continue
		}
		resParts := strings.SplitN(res, ":\t", 2)
		if len(resParts) != 2 {
			continue
		}
		xRes = append(xRes, [2]string{resParts[0], resParts[1]})
	}
	return xRes, nil
}

// TODO SCREEN_RESOURCES (xgb/randr)
