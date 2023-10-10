package tcellimg

import (
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

// TODO handle events

var _ views.Widget = (*Image)(nil)

type Image struct {
	*views.CellView
	img  *term.Image
	term *term.Terminal
	scr  tcell.Screen
	mdl  linesModel
}

func NewImage(img *term.Image, bounds image.Rectangle, tm *term.Terminal, scr tcell.Screen) (*Image, error) {
	if img == nil || tm == nil || scr == nil {
		return nil, errors.New(consts.ErrNilParam)
	}
	m := &Image{
		CellView: views.NewCellView(),
		img:      img,
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

	m.CellView.Lock()
	defer m.CellView.Unlock()
	m.scr.LockRegion(x, y, w, h, true)
	defer m.scr.LockRegion(x, y, w, h, false)
	for _, dr := range m.term.Drawers() {
		// TODO log
		if err := dr.Draw(m.img, bounds, m.term); err == nil {
			break
		}
	}
}

// copied from github.com/gdamore/tcell/v2/views/textarea.go
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
