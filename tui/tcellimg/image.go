package tcellimg

import (
	"context"
	"image"
	"image/draw"
	"log/slog"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
)

// TODO handle events

var _ image.Image = (*Image)(nil)
var _ draw.Image = (*Image)(nil)
var _ views.Widget = (*Image)(nil)

type Image struct {
	*views.CellView
	*term.Canvas
	term *term.Terminal
	scr  tcell.Screen
	mdl  linesModel
}

func NewImage(img image.Image, bounds image.Rectangle, tm *term.Terminal, scr tcell.Screen) (*Image, error) {
	// TODO accept
	if img == nil || tm == nil || scr == nil {
		return nil, errors.NilParam()
	}
	canvas, err := tm.NewCanvas(bounds)
	if err != nil {
		return nil, err
	}
	if logx.IsErr(canvas.SetImage(img), tm, slog.LevelError) {
		return nil, err
	}
	m := &Image{
		CellView: views.NewCellView(),
		Canvas:   canvas,
		term:     tm,
		scr:      scr,
	}
	view := views.NewViewPort(scr, bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())
	m.Init()
	m.SetView(view)
	mdl := &linesModel{
		width:  bounds.Dx(),
		height: bounds.Dy(),
		x:      bounds.Min.X,
		y:      bounds.Min.Y,
	}
	m.SetModel(mdl)
	return m, nil
}

func (m *Image) Draw() {
	// m.CellView.Draw() // area not drawn until tcell.Screen.Sync() is called

	mdl := m.CellView.GetModel()
	x, y, _, _ := mdl.GetCursor()
	w, h := mdl.GetBounds()
	bounds := image.Rect(x, y, x+w, y+h)
	if !bounds.Eq(m.Canvas.CellArea()) {
		logx.Error(`bounds changed`, m.term, "canvas-bounds", m.Canvas.CellArea(), "tcell-bounds", bounds)
	}
	m.CellView.Lock()
	defer m.CellView.Unlock()
	m.scr.LockRegion(x, y, w, h, true)
	defer m.scr.LockRegion(x, y, w, h, false)
	err := m.Canvas.Draw(nil)
	_ = logx.IsErr(err, m.term, slog.LevelError)
}

func (m *Image) Video(ctx context.Context, vid <-chan image.Image, frameDur time.Duration) error {
	if m == nil || m.Canvas == nil || ctx == nil {
		return nil
	}
	return m.Canvas.Video(ctx, vid, frameDur)
}

func (m *Image) Close() error {
	if m == nil {
		return nil
	}
	if m.Canvas == nil {
		m = nil
		return nil
	}
	err := m.Canvas.Close()
	m = nil
	return logx.Err(err, m.term, slog.LevelError)
}

// copied from github.com/gdamore/tcell/v2/views/textarea.go (Apache-2.0 license)
type linesModel struct {
	runes  [][]rune
	width  int
	height int
	x      int
	y      int
	hide   bool
	cursor bool
	style  tcell.Style
}

func (m *linesModel) GetCell(x, y int) (rune, tcell.Style, []rune, int) {
	// if x < 0 || y < 0 || y >= m.height || x >= len(m.runes[y]) {
	if x < 0 || y < 0 || y >= m.height || x >= m.width {
		return 0, m.style, nil, 1
	}
	// XXX: extend this to support combining and full width chars
	return 0, tcell.Style{}, nil, 1
}

func (m *linesModel) GetBounds() (int, int) {
	return m.width, m.height
}

func (m *linesModel) limitCursor() {
	if m.x > m.width-1 {
		m.x = m.width - 1
	}
	if m.y > m.height-1 {
		m.y = m.height - 1
	}
	if m.x < 0 {
		m.x = 0
	}
	if m.y < 0 {
		m.y = 0
	}
}

func (m *linesModel) SetCursor(x, y int) {
	m.x = x
	m.y = y
	m.limitCursor()
}

func (m *linesModel) MoveCursor(x, y int) {
	m.x += x
	m.y += y
	m.limitCursor()
}

func (m *linesModel) GetCursor() (int, int, bool, bool) {
	return m.x, m.y, m.cursor, !m.hide
}
