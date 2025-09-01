// Package uvtty provides an implementation of term.TTY via ultraviolet Terminal.
package uvtty

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"sync"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type TTYUV struct {
	mu             sync.RWMutex
	uvTerminal     *uv.Terminal
	fileName       string
	inFile         *os.File
	outFile        *os.File
	winch          chan term.Resolution
	watchWINCHOnce sync.Once
	eventCtx       context.Context
	eventCancel    context.CancelFunc
	eventCh        chan uv.Event
	inputBuf       *bytes.Buffer
	rawDataCh      chan []byte // Channel for raw input data before parsing

	// Size tracking
	cellW, cellH   int
	pixelW, pixelH int
}

var _ term.TTY = (*TTYUV)(nil)

// UVTerminal returns the underlying UV terminal
func (t *TTYUV) UVTerminal() *uv.Terminal {
	if t == nil {
		return nil
	}
	return t.uvTerminal
}

// rawCapturingReader wraps io.Reader to capture raw data before UV parsing
type rawCapturingReader struct {
	reader    io.Reader
	rawDataCh chan []byte
}

func (r *rawCapturingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		// Create a copy of the data to avoid race conditions
		data := make([]byte, n)
		copy(data, p[:n])

		// Send raw data to channel (non-blocking)
		select {
		case r.rawDataCh <- data:
		default:
			// Channel full, drop data to avoid blocking
		}
	}
	return n, err
}

func New(ttyFile string) (*TTYUV, error) {
	var uvTerm *uv.Terminal
	var inFile, outFile *os.File

	ctx, cancel := context.WithCancel(context.Background())

	tty := &TTYUV{
		fileName:    ttyFile,
		eventCtx:    ctx,
		eventCancel: cancel,
		eventCh:     make(chan uv.Event, 100),
		rawDataCh:   make(chan []byte, 100),
		inputBuf:    bytes.NewBuffer(nil),
	}

	if ttyFile == "" {
		// Create raw capturing reader for stdin
		rawReader := &rawCapturingReader{
			reader:    os.Stdin,
			rawDataCh: tty.rawDataCh,
		}
		uvTerm = uv.NewTerminal(rawReader, os.Stdout, os.Environ())
		if uvTerm == nil {
			cancel()
			return nil, errors.New("failed to create UV terminal")
		}
		inFile = os.Stdin
		outFile = os.Stdout
	} else {
		// Open the specified tty file
		var err error
		inFile, err = os.OpenFile(ttyFile, os.O_RDWR, 0)
		if err != nil {
			cancel()
			return nil, errors.New(err)
		}
		outFile = inFile // Same file for in/out

		// Create raw capturing reader for tty file
		rawReader := &rawCapturingReader{
			reader:    inFile,
			rawDataCh: tty.rawDataCh,
		}
		// Create UV terminal with raw capturing reader
		uvTerm = uv.NewTerminal(rawReader, outFile, os.Environ())
		if uvTerm == nil {
			inFile.Close()
			cancel()
			return nil, errors.New("failed to create UV terminal with custom tty")
		}
	}

	tty.uvTerminal = uvTerm
	tty.inFile = inFile
	tty.outFile = outFile

	// Start the UV terminal
	if err := uvTerm.Start(); err != nil {
		if ttyFile != "" {
			inFile.Close()
		}
		cancel()
		return nil, errors.New(err)
	}

	// Start processing raw data to input buffer
	go tty.processRawData()

	// Start monitoring events immediately to not miss anything
	go func() { _ = tty.uvTerminal.StreamEvents(tty.eventCtx, tty.eventCh) }()

	// Process events in background (for window size, etc.)
	go tty.processEvents()

	return tty, nil
}

func (t *TTYUV) processEvents() {
	for event := range t.eventCh {
		// Only handle resize events - input data is now handled by processRawData
		switch ev := event.(type) {
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
		default:
			// All other events (KeyPress, CursorPosition, DeviceAttributes, etc.)
			// are ignored because we get the raw data via processRawData
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
	if err := errors.NilReceiver(t, t.uvTerminal); err != nil {
		return 0, err
	}

	// Write to UV terminal
	n, err = t.uvTerminal.WriteString(string(b))
	if err != nil {
		return n, errors.New(err)
	}

	// Flush output
	if err := t.uvTerminal.Flush(); err != nil {
		return n, errors.New(err)
	}

	return n, nil
}

func (t *TTYUV) Read(p []byte) (n int, err error) {
	if err := errors.NilReceiver(t, t.uvTerminal); err != nil {
		return 0, err
	}

	// Block until we have data available
	for {
		t.mu.Lock()
		bufLen := t.inputBuf.Len()
		if bufLen > 0 {
			n, err = t.inputBuf.Read(p)
			t.mu.Unlock()
			return n, err
		}
		t.mu.Unlock()

		select {
		case <-t.eventCtx.Done():
			return 0, errors.New("context cancelled")
		case <-time.After(10 * time.Millisecond):
			// Small timeout to check buffer again, avoiding busy waiting
		}
	}
}

func (t *TTYUV) TTYDevName() string {
	if t == nil {
		return ""
	}
	return t.fileName
}

func (t *TTYUV) Close() error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Println("TTYUV.Close():", r)
		}
	}()
	if t.uvTerminal != nil {
		// TODO ioclt fails currently
		err := t.uvTerminal.Shutdown(context.Background())
		if err != nil {
			log.Println(err)
		}
		t.uvTerminal = nil
	}
	if t.eventCancel != nil {
		t.eventCancel()
	}
	if t.eventCh != nil {
		close(t.eventCh)
		t.eventCh = nil
	}
	return nil
}

// ResizeEvents implements the optional resize events interface
func (t *TTYUV) ResizeEvents() (_ <-chan term.Resolution, closeFunc func() error, _ error) {
	if t == nil || t.uvTerminal == nil {
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

	if t.winch == nil {
		errRet = errors.New("unable to receive resize events")
	}

	return t.winch, closeFunc, errRet
}

// SizePixel implements the optional pixel size interface
func (t *TTYUV) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	return t.getWindowSize()
}

// processRawData forwards captured raw bytes to the input buffer
func (t *TTYUV) processRawData() {
	for {
		select {
		case <-t.eventCtx.Done():
			return
		case rawData := <-t.rawDataCh:
			t.mu.Lock()
			t.inputBuf.Write(rawData)
			t.mu.Unlock()
		}
	}
}
