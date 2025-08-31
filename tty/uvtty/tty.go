// Package uvtty provides an implementation of term.TTY via ultraviolet Terminal.
package uvtty

import (
	"bytes"
	"context"
	"os"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYUV struct {
	UVTerminal     *uv.Terminal
	fileName       string
	inFile         *os.File
	outFile        *os.File
	winch          chan term.Resolution
	watchWINCHOnce sync.Once
	eventCtx       context.Context
	eventCancel    context.CancelFunc
	eventCh        chan uv.Event
	inputBuf       *bytes.Buffer
	mu             sync.RWMutex

	// Size tracking
	cellW, cellH   int
	pixelW, pixelH int
}

var _ term.TTY = (*TTYUV)(nil)

func New(ttyFile string) (*TTYUV, error) {
	var uvTerm *uv.Terminal
	var inFile, outFile *os.File

	if ttyFile == "" {
		uvTerm = uv.DefaultTerminal()
		if uvTerm == nil {
			return nil, errors.New("failed to create UV terminal")
		}
		// Use default stdin/stdout
		inFile = os.Stdin
		outFile = os.Stdout
	} else {
		// Open the specified tty file
		var err error
		inFile, err = os.OpenFile(ttyFile, os.O_RDWR, 0)
		if err != nil {
			return nil, errors.New(err)
		}
		outFile = inFile // Same file for in/out

		// Create UV terminal with custom files
		uvTerm = uv.NewTerminal(inFile, outFile, os.Environ())
		if uvTerm == nil {
			inFile.Close()
			return nil, errors.New("failed to create UV terminal with custom tty")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	tty := &TTYUV{
		UVTerminal:  uvTerm,
		fileName:    ttyFile,
		inFile:      inFile,
		outFile:     outFile,
		eventCtx:    ctx,
		eventCancel: cancel,
		eventCh:     make(chan uv.Event, 100),
		inputBuf:    bytes.NewBuffer(nil),
	}

	// Start monitoring events immediately to not miss anything
	go func() {
		defer close(tty.eventCh)
		_ = tty.UVTerminal.StreamEvents(tty.eventCtx, tty.eventCh)
	}()

	// Process events in background
	go tty.processEvents()

	return tty, nil
}

func (t *TTYUV) processEvents() {
	for event := range t.eventCh {
		switch ev := event.(type) {
		case uv.KeyPressEvent:
			if text := ev.Text; text != "" {
				t.mu.Lock()
				t.inputBuf.WriteString(text)
				t.mu.Unlock()
			}

		case uv.WindowSizeEvent:
			t.mu.Lock()
			t.cellW, t.cellH = ev.Width, ev.Height
			t.mu.Unlock()

			// Send resize event with current cell and pixel sizes
			t.sendResizeEvent()

		case uv.WindowPixelSizeEvent:
			t.mu.Lock()
			t.pixelW, t.pixelH = ev.Width, ev.Height
			t.mu.Unlock()

			// Send resize event with current cell and pixel sizes
			t.sendResizeEvent()
		}
	}
}

func (t *TTYUV) sendResizeEvent() {
	if t.winch == nil {
		return
	}

	t.mu.RLock()
	cellW, cellH := t.cellW, t.cellH
	pixelW, pixelH := t.pixelW, t.pixelH
	t.mu.RUnlock()

	res := term.Resolution{
		TermInCellsW: uint(cellW),
		TermInCellsH: uint(cellH),
		TermInPxlsW:  uint(pixelW),
		TermInPxlsH:  uint(pixelH),
	}

	// Calculate cell pixel size if we have both dimensions
	if cellW > 0 && cellH > 0 && pixelW > 0 && pixelH > 0 {
		res.CellInPxlsW = float64(pixelW) / float64(cellW)
		res.CellInPxlsH = float64(pixelH) / float64(cellH)
	}

	// Non-blocking send
	select {
	case t.winch <- res:
	default:
	}
}

func (t *TTYUV) Write(b []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.UVTerminal); err != nil {
		return 0, err
	}

	// Write to UV terminal
	n, err = t.UVTerminal.WriteString(string(b))
	if err != nil {
		return n, errors.New(err)
	}

	// Flush output
	if err := t.UVTerminal.Flush(); err != nil {
		return n, errors.New(err)
	}

	return n, nil
}

func (t *TTYUV) Read(p []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.UVTerminal); err != nil {
		return 0, err
	}

	t.mu.Lock()
	n, err = t.inputBuf.Read(p)
	t.mu.Unlock()

	if err != nil {
		return n, errors.New(err)
	}

	return n, nil
}

func (t *TTYUV) TTYDevName() string {
	if t == nil {
		return ""
	}
	if t.fileName != "" {
		return t.fileName
	}
	return internal.DefaultTTYDevice()
}

func (t *TTYUV) Close() error {
	if t == nil || t.UVTerminal == nil {
		return nil
	}

	// Cancel event context
	if t.eventCancel != nil {
		t.eventCancel()
	}

	// Shutdown UV terminal
	ctx := context.Background()
	if err := t.UVTerminal.Shutdown(ctx); err != nil {
		return errors.New(err)
	}

	return nil
}

// ResizeEvents implements the optional resize events interface
func (t *TTYUV) ResizeEvents() (_ <-chan term.Resolution, closeFunc func() error, _ error) {
	if t == nil || t.UVTerminal == nil {
		return nil, nil, errors.NilReceiver()
	}

	if t.winch != nil {
		return t.winch, nil, nil
	}

	var errRet error
	t.watchWINCHOnce.Do(func() {
		t.winch = make(chan term.Resolution)
		closeOnce := sync.Once{}
		closeFunc = func() error {
			closeOnce.Do(func() {
				if t.winch != nil {
					close(t.winch)
					t.winch = nil
				}
			})
			return nil
		}
	})

	if errRet == nil && t.winch == nil {
		errRet = errors.New("unable to receive resize events")
	}

	return t.winch, closeFunc, errRet
}

// SizePixel implements the optional pixel size interface
func (t *TTYUV) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	return t.getWindowSize()
}
