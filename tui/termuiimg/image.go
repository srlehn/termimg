package termuiimg

import (
	"image"

	"github.com/gizak/termui/v3"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type Image struct {
	termui.Block
	Canvas *term.Canvas
}

func NewImage(tm *term.Terminal, img image.Image, bounds image.Rectangle) (*Image, error) {
	if err := errors.NilParam(tm); err != nil {
		return nil, err
	}
	_ = tm.SetOptions(term.TUIMode)
	m := &Image{
		Block: *termui.NewBlock(),
	}
	m.SetRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)
	canvas, err := tm.NewCanvas(bounds)
	if err != nil {
		return nil, err
	}
	if img != nil {
		err := canvas.SetImage(img)
		if err != nil {
			canvas.Close()
			return nil, err
		}
	}
	m.Canvas = canvas
	return m, nil
}

func (m *Image) SetImage(img image.Image) {
	if m == nil || m.Canvas == nil {
		return
	}
	_ = m.Canvas.SetImage(img)
}

func (m *Image) Draw(buf *termui.Buffer) {
	if m == nil {
		return
	}
	m.Block.Draw(buf) // draw border
	if m.Canvas == nil {
		return
	}
	_ = m.Canvas.Draw(nil)
}

func (m *Image) Close() error {
	if m == nil || m.Canvas == nil {
		return nil
	}
	return m.Canvas.Close()
}
