//go:build dev

package bubbleteaimg

import (
	"fmt"
	"image"
	"log/slog"
	"strconv"
	"time"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/bubbleteatty"
)

var _ tea.Model = (*Image)(nil)

type Image struct {
	style *lipgloss.Style
	*term.Canvas
	bounds     image.Rectangle
	term       *term.Terminal
	tty        *bubbleteatty.TTYBubbleTea
	renderer   *lipgloss.Renderer
	scanner    *scanner
	sizeImgPxl image.Point
	id         string
}

func NewImage(tm *term.Terminal, img image.Image, style *lipgloss.Style) (*Image, error) {
	m := &Image{
		style: style,
	}
	m.initStyle()
	m.id = strconv.FormatUint(uint64(uintptr(unsafe.Pointer(m))), 10)
	if m.term == nil {
		if err := errors.NilParam(tm, img); err != nil {
			return nil, err
		}
		m.term = tm
	} else {
		if tm != nil {
			return nil, errors.New(`*term.Terminal can only be set once`)
		}
		if err := errors.NilParam(img); err != nil {
			return nil, err
		}
	}
	m.sizeImgPxl = img.Bounds().Size()
	bounds, err := m.cellBoundsFromStyle()
	if err != nil {
		return nil, err
	}
	canvas, err := tm.NewCanvas(bounds)
	if err != nil {
		return nil, err
	}
	if t := tm.TTY(); t != nil {
		if tb, ok := t.(*bubbleteatty.TTYBubbleTea); ok {
			m.tty = tb
			if tb.Scanner != nil {
				if sc, ok := tb.Scanner.(*scanner); ok {
					m.scanner = sc
				} else {
					return nil, errors.New(`tty already has a scanner of unknown type registered`)
				}
			} else {
				m.scanner = newScanner()
				tb.Scanner = m.scanner
			}
		}
	}
	if logx.IsErr(canvas.SetImage(term.NewImage(img)), tm, slog.LevelError) {
		return nil, err
	}
	if m.Canvas != nil {
		m.Canvas.Close()
	}
	m.Canvas = canvas
	return m, nil
}

func (m *Image) Style() *lipgloss.Style {
	if m == nil {
		return nil
	}
	return m.style
}

func (m *Image) initRenderer() {
	if m == nil || m.tty == nil || m.term == nil || m.renderer != nil {
		return
	}
	// *term.Terminal has to be initiated
	m.renderer = m.tty.LipGlossRenderer()
}

func (m *Image) SetStyle(style *lipgloss.Style) {
	if m == nil {
		return
	}
	m.initRenderer()
	s := style.Copy().Renderer(m.renderer)
	m.style = &s
}

func (m *Image) SetSize(sz image.Point) error {
	if err := errors.NilReceiver(m); err != nil {
		return err
	}
	m.initStyle()
	wMax := m.style.GetMaxWidth()
	hMax := m.style.GetMaxHeight()
	if (wMax > 0 && sz.X > wMax) || (hMax > 0 && sz.Y > hMax) {
		return errors.New(`image size is larger than allowed`)
	}
	m.style.Width(sz.X).Height(sz.Y)
	return nil
}

func (m *Image) SetBounds(bounds image.Rectangle) error {
	if err := errors.NilReceiver(m); err != nil {
		return err
	}
	m.initStyle()
	w := bounds.Dx()
	h := bounds.Dy()
	wMax := m.style.GetMaxWidth()
	hMax := m.style.GetMaxHeight()
	if (wMax > 0 && w > wMax) || (hMax > 0 && h > hMax) {
		return errors.New(`image size is larger than allowed`)
	}
	m.style.Width(w).Height(h)
	m.bounds = bounds
	return nil
}

func (m *Image) fitImage(bounds image.Rectangle) (image.Rectangle, error) {
	if err := errors.NilReceiver(m, m.scanner, m.scanner.rects); err != nil {
		return image.Rectangle{}, err
	}
	bounds, ok := m.scanner.rects[m.id]
	if !ok {
		return image.Rectangle{}, errors.New(`unknown image widget boundary`)
	}
	// remove frame
	wFr, hFr := m.style.GetFrameSize()
	x := m.style.GetBorderLeftSize() + m.style.GetMarginLeft() + m.style.GetPaddingLeft()
	y := m.style.GetBorderTopSize() + m.style.GetMarginTop() + m.style.GetPaddingTop()
	bounds = image.Rect(bounds.Min.X+x, bounds.Min.Y+y, bounds.Max.X+x-wFr, bounds.Max.Y+y-hFr)
	// fit image
	szSc, err := m.term.CellScale(m.sizeImgPxl, image.Pt(bounds.Dx(), 0))
	if err != nil {
		return image.Rectangle{}, err
	}
	if szSc.Y <= bounds.Dy() {
		d := (bounds.Dy() - szSc.Y) / 2
		return image.Rect(bounds.Min.X, bounds.Min.Y+d, bounds.Max.X, bounds.Max.Y-d), nil
	}
	szSc, err = m.term.CellScale(m.sizeImgPxl, image.Pt(0, bounds.Dy()))
	if err != nil {
		return image.Rectangle{}, err
	}
	if szSc.X <= bounds.Dx() {
		d := (bounds.Dx() - szSc.X) / 2
		return image.Rect(bounds.Min.X+d, bounds.Min.Y, bounds.Max.X-d, bounds.Max.Y), nil
	}
	return bounds, nil
}

func (m *Image) cellBoundsFromStyle() (image.Rectangle, error) {
	size, err := m.sizeInCellsFromStyle()
	if err != nil {
		return image.Rectangle{}, err
	}
	offset := image.Point{} // TODO
	bounds := image.Rect(offset.X, offset.Y, offset.X+size.X, offset.Y+size.Y)
	if bounds.Empty() {
		return image.Rectangle{}, errors.New(`unable to determine widget bounds`)
	}
	return bounds, nil
}

func (m *Image) sizeInCellsFromStyle() (image.Point, error) {
	if err := errors.NilReceiver(m, m.style); err != nil {
		return image.Point{}, err
	}
	m.initStyle()
	// TODO subtract widget offset and frame
	w := m.style.GetWidth()
	h := m.style.GetHeight()
	var size image.Point
	if w > 0 {
		if h > 0 {
			size = image.Pt(w, h)
		} else {
			if err := errors.NilReceiver(m.term); err != nil {
				return image.Point{}, err
			}
			szSc, err := m.term.CellScale(m.sizeImgPxl, image.Pt(w, 0))
			if err != nil {
				return image.Point{}, err
			}
			hMax := m.style.GetMaxHeight()
			_, tch, err := m.term.SizeInCells()
			if err != nil {
				return image.Point{}, err
			}
			hMaxReal := int(tch)
			if hMax > 0 && hMax < hMaxReal {
				hMaxReal = hMax
			}
			if hMax > 0 && szSc.Y > hMaxReal {
				szSc, err = m.term.CellScale(m.sizeImgPxl, image.Pt(0, hMaxReal))
				if err != nil {
					return image.Point{}, err
				}
			}
			size = szSc
		}
	} else {
		if h > 0 {
			if err := errors.NilReceiver(m.term); err != nil {
				return image.Point{}, err
			}
			szSc, err := m.term.CellScale(m.sizeImgPxl, image.Pt(0, h))
			if err != nil {
				return image.Point{}, err
			}
			wMax := m.style.GetMaxWidth()
			tcw, _, err := m.term.SizeInCells()
			if err != nil {
				return image.Point{}, err
			}
			wMaxReal := int(tcw)
			if wMax > 0 && wMax < wMaxReal {
				wMaxReal = wMax
			}
			if wMax > 0 && szSc.X > wMaxReal {
				szSc, err = m.term.CellScale(m.sizeImgPxl, image.Pt(wMaxReal, 0))
				if err != nil {
					return image.Point{}, err
				}
			}
			size = szSc
		} else {
			wMax := m.style.GetMaxWidth()
			hMax := m.style.GetMaxHeight()
			tcw, tch, err := m.term.SizeInCells()
			if err != nil {
				return image.Point{}, err
			}
			wMaxReal := int(tcw)
			if wMax > 0 && wMax < wMaxReal {
				wMaxReal = wMax
			}
			hMaxReal := int(tch)
			if hMax > 0 && hMax < hMaxReal {
				hMaxReal = hMax
			}
			szSc, err := m.term.CellScale(m.sizeImgPxl, image.Pt(wMaxReal, hMaxReal))
			if err != nil {
				return image.Point{}, err
			}
			size = szSc
		}
	}

	if size.Eq(image.Point{}) {
		return image.Point{}, errors.New(`unable to determine widget size`)
	}
	return size, nil
}

func (m *Image) Init() tea.Cmd { return nil }

func (m *Image) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *Image) initStyle() {
	if m == nil {
		return
	}
	var mod bool
	if m.style == nil {
		style := lipgloss.NewStyle()
		m.style = &style
		mod = true
	}
	if m.renderer == nil {
		m.initRenderer()
		mod = true
	}
	if mod {
		style := m.style.Copy().Renderer(m.renderer)
		m.style = &style
	}
}

var wrapFmt = string(csi) + `%s` + string(csiTerm) + `%s` + string(csi) + `%[1]s` + string(csiTerm)

func (m *Image) View() string {
	if m == nil {
		return `<nil bubbletea.Model>`
	}
	m.initStyle()
	ret := `` // widget text content
	// check for variable position
	if m.bounds == (image.Rectangle{}) {
		ret = fmt.Sprintf(wrapFmt, m.id, m.style.Render(ret))
	} else {
		// TODO needs testing
		ret = lipgloss.Place(
			m.bounds.Min.X, m.bounds.Min.Y,
			lipgloss.Right, lipgloss.Bottom,
			m.style.Render(ret),
		)
		if m.scanner != nil {
			m.scanner.setRectManually(m.id, m.bounds)
		}
	}
	if m.scanner != nil {
		m.scanner.setAfterWriteFunc(m.id, m.draw)
	} else {
		if m.term != nil {
			logx.Error(`bubbleteaimg.Image reqires a bubbleteatty.TTYBubbleTea`, m.term, `tty-type`, fmt.Sprintf(`%T`, m.term.TTY()))
		}
		// fallback, might become spammy & move cursor
		go func() {
			time.Sleep(1000 / 30 * time.Millisecond)
			m.draw(image.Rectangle{})
		}()
	}
	return ret
}

func (m *Image) draw(bounds image.Rectangle) {
	if m == nil || m.term == nil || m.Canvas == nil {
		return
	}

	// If we have valid scanner bounds, use them directly - no cursor queries needed
	if !bounds.Empty() {
		// Scanner provided coordinates - skip cursor operations
	} else {
		// Fallback to cursor queries only when scanner bounds unavailable
		x, y, errCursor := m.term.Cursor()
		if !logx.IsErr(errCursor, m.term, slog.LevelError) {
			// Set up cursor restoration immediately after successful query
			defer func() {
				errCursor = m.term.SetCursor(x, y)
				logx.IsErr(errCursor, m.term, slog.LevelError)
			}()
			// Use style dimensions with cursor position
			w := m.style.GetWidth()
			h := m.style.GetHeight()
			if w == 0 || h == 0 {
				return
			}
			bounds = image.Rect(int(x), int(y), int(x)+w, int(y)+h)
		} else {
			return
		}
	}
	bounds, err := m.fitImage(bounds)
	_ = logx.IsErr(err, m.term, slog.LevelInfo)
	errPos := m.Canvas.SetCellArea(bounds)
	if logx.IsErr(errPos, m.term, slog.LevelError) {
		return
	}
	errDraw := m.Canvas.Draw(nil)
	logx.IsErr(errDraw, m.term, slog.LevelError)
}
