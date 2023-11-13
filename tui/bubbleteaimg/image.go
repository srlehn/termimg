package bubbleteaimg

import (
	"fmt"
	"image"
	"log/slog"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/logx"
	"github.com/srlehn/termimg/term"
	"github.com/srlehn/termimg/tty/bubbleteatty"
)

var _ tea.Model = (*Image)(nil)

type Image struct {
	Style *lipgloss.Style
	*term.Canvas
	term       *term.Terminal
	tty        *bubbleteatty.TTYBubbleTea
	sizeImgPxl image.Point
	id         string
}

func NewImage(style *lipgloss.Style) (*Image, error) {
	mdl := &Image{
		Style: style,
	}
	mdl.initStyle()
	mdl.id = fmt.Sprintf(`img_wdg_%p`, mdl)
	return mdl, nil
}

func (m *Image) Setup(tm *term.Terminal, img image.Image) error {
	if err := errors.NilReceiver(m, m.Style); err != nil {
		return err
	}
	if m.term == nil {
		if err := errors.NilParam(tm, img); err != nil {
			return err
		}
		m.term = tm
	} else {
		if tm != nil {
			return errors.New(`*term.Terminal can only be set once`)
		}
		if err := errors.NilParam(img); err != nil {
			return err
		}
	}
	m.sizeImgPxl = img.Bounds().Size()
	bounds, err := m.cellBounds()
	if err != nil {
		return err
	}
	canvas, err := tm.NewCanvas(bounds)
	if err != nil {
		return err
	}
	if t := tm.TTY(); t != nil {
		if tb, ok := t.(*bubbleteatty.TTYBubbleTea); ok {
			m.tty = tb
		}
	}
	if logx.IsErr(canvas.SetImage(term.NewImage(img)), tm, slog.LevelError) {
		return err
	}
	if m.Canvas != nil {
		m.Canvas.Close()
	}
	m.Canvas = canvas
	return nil
}
func (m *Image) cellBounds() (image.Rectangle, error) {
	offset, err := m.cellOffset()
	if err != nil {
		return image.Rectangle{}, err
	}
	size, err := m.cellSize()
	if err != nil {
		return image.Rectangle{}, err
	}
	bounds := image.Rect(offset.X, offset.Y, offset.X+size.X, offset.Y+size.Y)
	if bounds.Empty() {
		return image.Rectangle{}, errors.New(`unable to determine widget bounds`)
	}
	return bounds, nil
}
func (m *Image) cellSize() (image.Point, error) {
	if err := errors.NilReceiver(m, m.Style); err != nil {
		return image.Point{}, err
	}
	m.initStyle()
	// TODO subtract widget offset and frame
	w := m.Style.GetWidth()
	h := m.Style.GetHeight()
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
			hMax := m.Style.GetMaxHeight()
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
			wMax := m.Style.GetMaxWidth()
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
			wMax := m.Style.GetMaxWidth()
			hMax := m.Style.GetMaxHeight()
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
func (m *Image) cellOffset() (image.Point, error) {
	return image.Point{}, nil // TODO
	// return image.Point{}, errors.New(`unable to determine widget size`)
}

func (m *Image) Init() tea.Cmd {
	return nil
}

func (m *Image) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := func() tea.Msg { return msg }
	return m, cmd
}

func (m *Image) initStyle() {
	if m == nil {
		return
	}
	if m.Style == nil {
		style := lipgloss.NewStyle()
		m.Style = &style
	}
}

func (m *Image) View() string {
	if m == nil {
		return `<nil bubbletea.Model>`
	}
	m.initStyle()
	s := fmt.Sprintf("%+#v", m.Style) // TODO rm
	// APC needs https://github.com/muesli/reflow/pull/65
	// s := "\033_TEST\033\\" // APC ... ST // TODO rm
	// s := "\033[34m" // TODO rm
	ret := m.Style.Render(fmt.Sprintf("%s\nlen %q: %d\n", m.id, s, lipgloss.Width(s))) // TODO rm
	cellBounds, err := m.cellBounds()
	if logx.IsErr(err, m.term, slog.LevelError) {
		return ret
	}
	x, y, err := m.term.Cursor()
	if !logx.IsErr(err, m.term, slog.LevelError) {
		cellBounds = cellBounds.Add(image.Pt(int(x), int(y)))
	}
	err = m.Canvas.SetCellArea(cellBounds)
	if logx.IsErr(err, m.term, slog.LevelError) {
		return ret
	}
	if m.tty != nil {
		// TODO prepare here, draw after render
		m.tty.SetAfterWriteFunc(m.id, func() {
			m.draw()
		})
		// m.tty.Program.Send(tea.WindowSizeMsg{})
	} else {
		if m.term != nil {
			logx.Error(`bubbleteaimg.Image reqires a bubbleteatty.TTYBubbleTea`, m.term, `tty-type`, fmt.Sprintf(`%T`, m.term.TTY()))
		}
		// term.Terminal is using wrong tty implementation
		// fallback, might become spammy & move cursor
		go func() {
			time.Sleep(1000 / 30 * time.Millisecond)
			m.draw()
		}()
	}
	return ret
}

func (m *Image) draw() {
	if m == nil || m.term == nil || m.Canvas == nil {
		return
	}
	x, y, errCursor := m.term.Cursor()
	logx.IsErr(errCursor, m.term, slog.LevelError)
	errDraw := m.Canvas.Draw(nil)
	logx.IsErr(errDraw, m.term, slog.LevelError)
	if errCursor == nil {
		errCursor = m.term.SetCursor(x, y)
		logx.IsErr(errCursor, m.term, slog.LevelError)
	}
}

type ImageDrawMsg struct {
	id     string
	drFn   func()
	bounds image.Rectangle
}

func (m ImageDrawMsg) ID() string { return m.id }

func (m ImageDrawMsg) Draw() {
	if m.drFn == nil {
		return
	}
	m.drFn()
}

func (m ImageDrawMsg) Bounds() image.Rectangle { return m.bounds }

func DrawImagesFrom(msgs ...ImageDrawMsg) {
	var msgsCleaned []ImageDrawMsg
	for i := range msgs {
		if msgs[i].drFn == nil {
			continue
		}
		if len(msgs[i].id) > 0 {
			msgsCleaned = append(slices.DeleteFunc(msgsCleaned, func(msg ImageDrawMsg) bool { return msg.id == msgs[i].id }), msgs[i])
		} else {
			msgsCleaned = append(msgsCleaned, msgs[i])
		}
	}
	for i := range msgsCleaned {
		msgsCleaned[i].drFn()
	}
}
