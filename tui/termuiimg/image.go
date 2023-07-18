package termuiimg

import (
	"fmt"
	"image"

	"github.com/gizak/termui/v3"

	_ "github.com/srlehn/termimg/drawers"
	"github.com/srlehn/termimg/query/qdefault"
	"github.com/srlehn/termimg/resize/rdefault"
	"github.com/srlehn/termimg/term"
	_ "github.com/srlehn/termimg/terminals"
	"github.com/srlehn/termimg/tty/gotty"
	"github.com/srlehn/termimg/wm"
	"github.com/srlehn/termimg/wm/wmimpl"
)

type Image struct {
	termui.Block
	image *term.Image
	cfg   *config
}

func NewImage(img image.Image) *Image {
	return &Image{
		Block: *termui.NewBlock(),
		image: term.NewImage(img),
	}
}

func (m *Image) Draw(buf *termui.Buffer) {
	if m == nil {
		return
	}
	m.Block.Draw(buf) // draw border
	if m.image == nil {
		return
	}
	if m.cfg == nil {
		cfg, err := conf()
		if err != nil || cfg == nil {
			return
		}
		m.cfg = cfg
	}

	blockW := m.Inner.Dx()
	blockH := m.Inner.Dy()
	bufW := buf.Dx()
	bufH := buf.Dy()
	pr(fmt.Sprintf("dim %dx%d %dx%d\n", blockW, blockH, bufW, bufH), m, buf)
	pr(fmt.Sprintf("min %dx%d %dx%d\n", m.Min.X, m.Min.Y, buf.Min.X, buf.Min.Y), m, buf)
	pr(fmt.Sprintf("max %dx%d %dx%d\n", m.Max.X, m.Max.Y, buf.Max.X, buf.Max.Y), m, buf)
	dr := m.cfg.term.Drawers()[0]
	pr(fmt.Sprintf("%s %s\n", m.cfg.term.Name(), dr.Name()), m, buf)
	repl, errQu := m.cfg.term.Query("\033[0c", term.StopOnAlpha)
	pr(fmt.Sprintf("%q %v\n", repl, errQu), m, buf)

	err := term.Draw(m.image, m.Block.Inner, m.cfg.rsz, m.cfg.term)
	if err != nil {
		pr(fmt.Sprintf("err %v\n", err), m, buf)
		return
	}
}

func (m *Image) Close() error {
	if m == nil {
		return nil
	}
	if m.cfg != nil && m.cfg.term != nil {
		return m.cfg.term.Close()
	}
	return nil
}

type config struct {
	term *term.Terminal
	rsz  term.Resizer
}

var configDefault *config

func conf() (*config, error) {
	if configDefault != nil {
		return configDefault, nil
	}

	// ptyName := `/dev/stdin`
	ptyName := `/dev/tty`
	ttyProv := func(ptyName string) (term.TTY, error) { return gotty.New(ptyName) }
	rsz := &rdefault.Resizer{}

	wm.SetImpl(wmimpl.Impl())
	cr := &term.Creator{
		PTYName:         ptyName,
		TTYProvFallback: ttyProv,
		Querier:         qdefault.NewQuerier(),
		WindowProvider:  wm.NewWindow,
		Resizer:         rsz,
	}
	tm, err := term.NewTerminal(cr)
	if err != nil {
		return nil, err
	}

	configDefault = &config{
		term: tm,
		rsz:  rsz,
	}

	return configDefault, nil
}

// debug

var prIdx int

func pr(s string, img *Image, buf *termui.Buffer) {
	if img == nil || buf == nil || len(s) == 0 {
		return
	}
	blockH := img.Inner.Dy()
	if blockH != 0 {
		prIdx = prIdx%blockH + 1
	} else {
		prIdx++
	}
	buf.SetString(s, termui.StyleClear, image.Pt(img.Min.X+1, img.Min.Y+prIdx))
}
