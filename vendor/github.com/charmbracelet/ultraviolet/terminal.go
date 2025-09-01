package uv

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/cancelreader"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrNotTerminal is returned when one of the I/O streams is not a terminal.
	ErrNotTerminal = fmt.Errorf("not a terminal")

	// ErrPlatformNotSupported is returned when the platform is not supported.
	ErrPlatformNotSupported = fmt.Errorf("platform not supported")

	// ErrStarted is returned when the terminal has already been started.
	ErrStarted = fmt.Errorf("terminal already started")
)

// Terminal represents a terminal screen that can be manipulated and drawn to.
// It handles reading events from the terminal using [WinChReceiver],
// [SequenceReceiver], and [ConReceiver].
type Terminal struct {
	// Terminal I/O streams and state.
	in          io.Reader
	out         io.Writer
	inTty       term.File
	inTtyState  *term.State
	outTty      term.File
	outTtyState *term.State
	started     bool // Indicates if the terminal has been started.

	// Terminal type, screen and buffer.
	termtype            string            // The $TERM type.
	environ             Environ           // The environment variables.
	buf                 *Buffer           // Reference to the last buffer used.
	scr                 *TerminalRenderer // The actual screen to be drawn to.
	size                Size              // The last known full size of the terminal.
	pixSize             Size              // The last known pixel size of the terminal.
	method              ansi.Method       // The width method used by the terminal.
	profile             colorprofile.Profile
	modes               ansi.Modes  // Keep track of terminal modes.
	useTabs             bool        // Whether to use hard tabs or not.
	useBspace           bool        // Whether to use backspace or not.
	cursorHidden        bool        // The cached state of the cursor visibility.
	altscreen           bool        // Cached state of the alternate screen buffer.
	setFg, setBg, setCc color.Color // The current set foreground, background, and cursor colors.
	curStyle            int         // The encoded cursor style.
	cur                 Position    // The last known cursor position before shutdown.

	// Terminal input stream.
	cr        cancelreader.CancelReader
	rd        *TerminalReader
	winchn    *WindowSizeNotifier // The window size notifier for the terminal.
	evch      chan Event
	evctx     context.Context    // The context for the event channel.
	evcancel  context.CancelFunc // The cancel function for the event channel.
	once      sync.Once
	mouseMode MouseMode // The mouse mode for the terminal.
	wg        sync.WaitGroup
	m         sync.RWMutex // Mutex to protect the terminal state.

	logger Logger // The debug logger for I/O.
}

// DefaultTerminal returns a new default terminal instance that uses
// [os.Stdin], [os.Stdout], and [os.Environ].
func DefaultTerminal() *Terminal {
	return NewTerminal(os.Stdin, os.Stdout, os.Environ())
}

var defaultModes = ansi.Modes{
	// These are modes we care about and want to track.
	ansi.TextCursorEnableMode:    ansi.ModeSet,
	ansi.AltScreenSaveCursorMode: ansi.ModeReset,
	ansi.ButtonEventMouseMode:    ansi.ModeReset,
	ansi.AnyEventMouseMode:       ansi.ModeReset,
	ansi.SgrExtMouseMode:         ansi.ModeReset,
	ansi.BracketedPasteMode:      ansi.ModeReset,
	ansi.FocusEventMode:          ansi.ModeReset,
}

// NewTerminal creates a new Terminal instance with the given terminal size.
// Use [term.GetSize] to get the size of the output screen.
func NewTerminal(in io.Reader, out io.Writer, env []string) *Terminal {
	t := new(Terminal)
	t.in = in
	t.out = out
	if f, ok := in.(term.File); ok {
		t.inTty = f
	}
	if f, ok := out.(term.File); ok {
		t.outTty = f
	}
	t.modes = ansi.Modes{}
	// Initialize the default modes.
	for k, v := range defaultModes {
		t.modes[k] = v
	}
	t.environ = env
	t.termtype = t.environ.Getenv("TERM")
	t.scr = NewTerminalRenderer(t.out, t.environ)
	t.buf = NewBuffer(0, 0)
	t.method = ansi.WcWidth // Default width method.
	t.SetColorProfile(colorprofile.Detect(out, env))
	t.evch = make(chan Event)
	t.once = sync.Once{}

	// Window size changes only for non-Windows platforms.
	if !isWindows {
		// Create default input receivers.
		winchTty := t.inTty
		if winchTty == nil {
			winchTty = t.outTty
		}
		t.winchn = NewWindowSizeNotifier(winchTty)
	}

	// Handle debugging I/O.
	debug, ok := os.LookupEnv("UV_DEBUG")
	if ok && len(debug) > 0 {
		f, err := os.OpenFile(debug, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			panic("failed to open debug file: " + err.Error())
		}

		logger := log.New(f, "uv: ", log.LstdFlags|log.Lshortfile)
		t.SetLogger(logger)
	}
	return t
}

// SetLogger sets the debug logger for the terminal. This is used to log debug
// information about the terminal I/O. By default, it is set to a no-op logger.
func (t *Terminal) SetLogger(logger Logger) {
	t.logger = logger
}

// ColorProfile returns the currently used color profile for the terminal.
func (t *Terminal) ColorProfile() colorprofile.Profile {
	return t.profile
}

// SetColorProfile sets a custom color profile for the terminal. This is useful
// for forcing a specific color output. By default, the terminal will use the
// system's color profile inferred by the environment variables.
func (t *Terminal) SetColorProfile(p colorprofile.Profile) {
	t.profile = p
	t.scr.SetColorProfile(p)
}

// ColorModel returns the color model of the terminal screen.
func (t *Terminal) ColorModel() color.Model {
	return t.ColorProfile()
}

// SetWidthMethod sets the width method used by the terminal. This is typically
// used to determine how the terminal calculates the width of a single
// grapheme.
// The default method is [ansi.WcWidth].
func (t *Terminal) SetWidthMethod(method ansi.Method) {
	t.method = method
}

// WidthMethod returns the width method used by the terminal. This is typically
// used to determine how the terminal calculates the width of a single
// grapheme.
func (t *Terminal) WidthMethod() WidthMethod {
	return t.method
}

var _ color.Model = (*Terminal)(nil)

// Convert converts the given color to the terminal's color profile. This
// implements the [color.Model] interface, allowing you to convert any color to
// the terminal's preferred color model.
func (t *Terminal) Convert(c color.Color) color.Color {
	return t.profile.Convert(c)
}

// GetSize returns the size of the terminal screen. It errors if the size
// cannot be determined.
func (t *Terminal) GetSize() (width, height int, err error) {
	w, h, err := t.getSize()
	if err != nil {
		return 0, 0, fmt.Errorf("error getting terminal size: %w", err)
	}
	// Cache the last known size.
	t.m.Lock()
	t.size.Width = w
	t.size.Height = h
	t.m.Unlock()
	return w, h, nil
}

// Bounds returns the bounds of the terminal screen buffer. This is the
// rectangle that contains start and end points of the screen buffer.
// This is different from [Terminal.GetSize] which queries the size of the
// terminal window. The screen buffer can occupy a portion or all of the
// terminal window. Use [Terminal.Resize] to change the size of the screen
// buffer.
func (t *Terminal) Bounds() Rectangle {
	return Rect(0, 0, t.buf.Width(), t.buf.Height())
}

// SetCell sets the cell at the given x, y position in the terminal buffer.
func (t *Terminal) SetCell(x int, y int, c *Cell) {
	t.buf.SetCell(x, y, c)
}

// CellAt returns the cell at the given x, y position in the terminal buffer.
func (t *Terminal) CellAt(x int, y int) *Cell {
	return t.buf.CellAt(x, y)
}

var _ Screen = (*Terminal)(nil)

// Clear fills the terminal screen with empty cells, and clears the
// terminal screen.
//
// This is different from [Terminal.Erase], which only fills the screen
// buffer with empty cells without erasing the terminal screen first.
func (t *Terminal) Clear() {
	t.buf.Clear()
}

// ClearArea fills the given area of the terminal screen with empty cells.
func (t *Terminal) ClearArea(area Rectangle) {
	t.buf.ClearArea(area)
}

// Fill fills the terminal screen with the given cell. If the cell is nil, it
// fills the screen with empty cells.
func (t *Terminal) Fill(c *Cell) {
	t.buf.Fill(c)
}

// FillArea fills the given area of the terminal screen with the given cell.
// If the cell is nil, it fills the area with empty cells.
func (t *Terminal) FillArea(c *Cell, area Rectangle) {
	t.buf.FillArea(c, area)
}

// Clone returns a copy of the terminal screen buffer. This is useful for
// creating a snapshot of the current terminal state without modifying the
// original buffer.
func (t *Terminal) Clone() *Buffer {
	return t.buf.Clone()
}

// CloneArea clones the given area of the terminal screen and returns a new
// buffer with the same size as the area. The new buffer will contain the
// same cells as the area in the terminal screen.
func (t *Terminal) CloneArea(area Rectangle) *Buffer {
	return t.buf.CloneArea(area)
}

// Position returns the last known position of the cursor in the terminal.
func (t *Terminal) Position() (int, int) {
	return t.scr.Position()
}

// SetPosition sets the position of the cursor in the terminal. This is
// typically used when the cursor was moved manually outside of the [Terminal]
// context.
func (t *Terminal) SetPosition(x, y int) {
	t.scr.SetPosition(x, y)
}

// MoveTo moves the cursor to the given x, y position in the terminal.
// This won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) MoveTo(x, y int) {
	t.scr.MoveTo(x, y)
}

func (t *Terminal) configureRenderer() {
	t.scr.SetColorProfile(t.profile)
	if t.useTabs {
		t.m.RLock()
		t.scr.SetTabStops(t.size.Width)
		t.m.RUnlock()
	}
	t.scr.SetBackspace(t.useBspace)
	t.scr.SetRelativeCursor(true) // Initial state is relative cursor movements.
	if t.scr != nil {
		if t.scr.AltScreen() {
			t.scr.EnterAltScreen()
		} else {
			t.scr.ExitAltScreen()
		}
	}
	t.scr.SetLogger(t.logger)
}

// Erase fills the screen buffer with empty cells, and wipe the terminal
// screen. This is different from [Terminal.Clear], which only fills the
// terminal with empty cells.
//
// This won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) Erase() {
	t.buf.Touched = nil
	t.scr.Erase()
	t.Clear()
}

// Render computes the necessary changes to the terminal screen and marks the
// current buffer pending to be rendered to the terminal screen.
//
// Use [Terminal.Display] or [Terminal.Flush] to actually render the buffer to
// the terminal screen.
func (t *Terminal) Render() {
	t.scr.Render(t.buf)
}

// Display computes the necessary changes to the terminal screen and renders
// the current buffer to the terminal screen.
//
// Typically, you would call this after modifying the terminal buffer using
// [Terminal.SetCell] or [Terminal.PrependString].
func (t *Terminal) Display() error {
	t.scr.Render(t.buf)
	return t.scr.Flush()
}

// Flush flushes any pending renders to the terminal screen. This is typically
// used to flush the underlying screen buffer to the terminal.
//
// Use [Terminal.Buffered] to check how many bytes pending to be flushed.
func (t *Terminal) Flush() error {
	return t.scr.Flush()
}

// Buffered returns the number of bytes buffered for the flush operation.
func (t *Terminal) Buffered() int {
	return t.scr.Buffered()
}

// Touched returns the number of touched lines in the terminal buffer.
func (t *Terminal) Touched() int {
	return t.scr.Touched(t.buf)
}

// GetMode returns the current state of the given mode in the terminal. This is
// typically used to check if a specific mode is enabled or disabled on the
// terminal.
func (t *Terminal) GetMode(mode ansi.Mode) ansi.ModeSetting {
	m := t.modes[mode]
	return m
}

// SetMode sets the given mode and its setting in the [Terminal]. This is
// usually used when an [ansi.Mode] was enabled/disabled outside the context of
// [Terminal].
func (t *Terminal) SetMode(mode ansi.Mode, setting ansi.ModeSetting) {
	t.modes[mode] = setting
}

// EnableMode enables the given modes on the terminal. This is typically used
// to enable mouse support, bracketed paste mode, and other terminal features.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) EnableMode(modes ...ansi.Mode) {
	if len(modes) == 0 {
		return
	}
	for _, m := range modes {
		t.modes[m] = ansi.ModeSet
	}
	t.scr.WriteString(ansi.SetMode(modes...)) //nolint:errcheck,gosec
}

// DisableMode disables the given modes on the terminal. This is typically
// used to disable mouse support, bracketed paste mode, and other terminal
// features.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) DisableMode(modes ...ansi.Mode) {
	if len(modes) == 0 {
		return
	}
	for _, m := range modes {
		t.modes[m] = ansi.ModeReset
	}
	t.scr.WriteString(ansi.ResetMode(modes...)) //nolint:errcheck,gosec
}

// RequestMode requests the current state of the given modes from the terminal.
// This is typically used to check if a specific mode is recognized, enabled,
// or disabled on the terminal.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) RequestMode(mode ansi.Mode) {
	t.scr.WriteString(ansi.RequestMode(mode)) //nolint:errcheck,gosec
}

// MouseMode represents the mouse mode for the terminal. It is used to enable
// or disable mouse support on the terminal.
type MouseMode byte

const (
	// ButtonMouseMode enables basic mouse button clicks and releases.
	ButtonMouseMode MouseMode = 1 << iota
	// DragMouseMode enables basic mouse buttons [ButtonMouseMode] as well as
	// click-and-drag mouse motion events.
	DragMouseMode
	// AllMouseMode enables all mouse events including button clicks, releases,
	// and all motion events. This inclodes the [ButtonMouseMode] and
	// [DragMouseMode] modes.
	AllMouseMode
)

// SetForegroundColor sets the terminal default foreground color.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) SetForegroundColor(c color.Color) {
	t.setFg = c
	col, ok := colorful.MakeColor(c)
	if ok {
		t.scr.WriteString(ansi.SetForegroundColor(col.Hex())) //nolint:errcheck,gosec
	}
}

// RequestForegroundColor requests the current foreground color of the terminal.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) RequestForegroundColor() {
	t.scr.WriteString(ansi.RequestForegroundColor) //nolint:errcheck,gosec
}

// ResetForegroundColor resets the terminal foreground color to the
// default color.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) ResetForegroundColor() {
	t.setFg = nil
	t.scr.WriteString(ansi.ResetForegroundColor) //nolint:errcheck,gosec
}

// SetBackgroundColor sets the terminal default background color.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) SetBackgroundColor(c color.Color) {
	t.setBg = c
	col, ok := colorful.MakeColor(c)
	if ok {
		t.scr.WriteString(ansi.SetBackgroundColor(col.Hex())) //nolint:errcheck,gosec
	}
}

// RequestBackgroundColor requests the current background color of the terminal.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) RequestBackgroundColor() {
	t.scr.WriteString(ansi.RequestBackgroundColor) //nolint:errcheck,gosec
}

// ResetBackgroundColor resets the terminal background color to the
// default color.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) ResetBackgroundColor() {
	t.setBg = nil
	t.scr.WriteString(ansi.ResetBackgroundColor) //nolint:errcheck,gosec
}

// SetCursorColor sets the terminal cursor color.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) SetCursorColor(c color.Color) {
	t.setCc = c
	col, ok := colorful.MakeColor(c)
	if ok {
		t.scr.WriteString(ansi.SetCursorColor(col.Hex())) //nolint:errcheck,gosec
	}
}

// RequestCursorColor requests the current cursor color of the terminal.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) RequestCursorColor() {
	t.scr.WriteString(ansi.RequestCursorColor) //nolint:errcheck,gosec
}

// ResetCursorColor resets the terminal cursor color to the
// default color.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) ResetCursorColor() {
	t.setCc = nil
	t.scr.WriteString(ansi.ResetCursorColor) //nolint:errcheck,gosec
}

// SetCursorShape sets the terminal cursor shape and blinking style.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) SetCursorShape(shape CursorShape, blink bool) {
	style := shape.Encode(blink)
	t.curStyle = style
	t.scr.WriteString(ansi.SetCursorStyle(style)) //nolint:errcheck,gosec
}

// EnableMouse enables mouse support on the terminal.
// Calling this without any modes will enable all mouse modes by default.
// The available modes are:
//   - [ButtonMouseMode] Enables basic mouse button clicks and releases.
//   - [DragMouseMode] Enables basic mouse buttons [ButtonMouseMode] as well as
//     click-and-drag mouse motion events.
//   - [AllMouseMode] Enables all mouse events including button clicks, releases,
//     and all motion events. This inclodes the [ButtonMouseMode] and
//     [DragMouseMode] modes.
//
// Note that on Unix, this won't take any effect until the next
// [Terminal.Display] or [Terminal.Flush] call.
func (t *Terminal) EnableMouse(modes ...MouseMode) {
	var mode MouseMode
	for _, m := range modes {
		mode |= m
	}
	if len(modes) == 1 {
		if mode&AllMouseMode != 0 {
			mode |= ButtonMouseMode | DragMouseMode
		}
		if mode&DragMouseMode != 0 {
			mode |= ButtonMouseMode
		}
	}
	if mode == 0 {
		mode = ButtonMouseMode | DragMouseMode | AllMouseMode
	}
	t.mouseMode = mode
	if !isWindows {
		modes := []ansi.Mode{}
		if t.mouseMode&AllMouseMode != 0 {
			modes = append(modes, ansi.AnyEventMouseMode)
		} else if t.mouseMode&DragMouseMode != 0 {
			modes = append(modes, ansi.ButtonEventMouseMode)
		} else if t.mouseMode&ButtonMouseMode != 0 {
			modes = append(modes, ansi.NormalMouseMode)
		}
		modes = append(modes, ansi.SgrExtMouseMode)
		t.EnableMode(modes...)
	}
	t.enableWindowsMouse() //nolint:errcheck,gosec
}

// DisableMouse disables mouse support on the terminal. This will disable mouse
// button and button motion events.
//
// Note that on Unix, this won't take any effect until the next
// [Terminal.Display] or [Terminal.Flush] call.
func (t *Terminal) DisableMouse() {
	t.mouseMode = 0
	if !isWindows {
		var modes []ansi.Mode
		if t.modes.Get(ansi.AnyEventMouseMode).IsSet() {
			modes = append(modes, ansi.AnyEventMouseMode)
		}
		if t.modes.Get(ansi.ButtonEventMouseMode).IsSet() {
			modes = append(modes, ansi.ButtonEventMouseMode)
		}
		if t.modes.Get(ansi.NormalMouseMode).IsSet() {
			modes = append(modes, ansi.NormalMouseMode)
		}
		if t.modes.Get(ansi.SgrExtMouseMode).IsSet() {
			modes = append(modes, ansi.SgrExtMouseMode)
		}
		t.DisableMode(modes...)
	}
	t.disableWindowsMouse() //nolint:errcheck,gosec
}

// EnableBracketedPaste enables bracketed paste mode on the terminal. This is
// typically used to enable support for pasting text into the terminal without
// interfering with the terminal's input handling.
func (t *Terminal) EnableBracketedPaste() {
	t.EnableMode(ansi.BracketedPasteMode)
}

// DisableBracketedPaste disables bracketed paste mode on the terminal. This is
// typically used to disable support for pasting text into the terminal.
func (t *Terminal) DisableBracketedPaste() {
	t.DisableMode(ansi.BracketedPasteMode)
}

// EnableFocusEvents enables focus/blur receiving notification events on the
// terminal.
func (t *Terminal) EnableFocusEvents() {
	t.EnableMode(ansi.FocusEventMode)
}

// DisableFocusEvents disables focus/blur receiving notification events on the
// terminal.
func (t *Terminal) DisableFocusEvents() {
	t.DisableMode(ansi.FocusEventMode)
}

// EnterAltScreen enters the alternate screen buffer. This is typically used
// for applications that want to take over the entire terminal screen.
//
// The [Terminal] manages the alternate screen buffer for you based on the
// [Viewport] used during [Terminal.Display]. This means that you don't need to
// call this unless you know what you're doing.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) EnterAltScreen() {
	t.enterAltScreen(true)
}

// cursor indicates whether we want to set the cursor visibility state after
// entering the alt screen.
// We do this because some terminals maintain a separate cursor visibility
// state for the alt screen and the normal screen.
func (t *Terminal) enterAltScreen(cursor bool) {
	altscreen := t.scr.AltScreen()
	t.scr.EnterAltScreen()
	if cursor && !altscreen {
		if t.scr.CursorHidden() {
			t.hideCursor()
		} else {
			t.showCursor()
		}
	}
	t.scr.SetRelativeCursor(false)
	t.modes[ansi.AltScreenSaveCursorMode] = ansi.ModeSet
}

// ExitAltScreen exits the alternate screen buffer and returns to the normal
// screen buffer.
//
// The [Terminal] manages the alternate screen buffer for you based on the
// [Viewport] used during [Terminal.Display]. This means that you don't need to
// call this unless you know what you're doing.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) ExitAltScreen() {
	t.exitAltScreen(true)
}

// cursor indicates whether we want to set the cursor visibility state after
// exiting the alt screen.
// We do this because some terminals maintain a separate cursor visibility
// state for the alt screen and the normal screen.
func (t *Terminal) exitAltScreen(cursor bool) {
	altscreen := t.scr.AltScreen()
	t.scr.ExitAltScreen()
	if cursor && altscreen {
		if t.scr.CursorHidden() {
			t.hideCursor()
		} else {
			t.showCursor()
		}
	}
	t.scr.SetRelativeCursor(true)
	t.modes[ansi.AltScreenSaveCursorMode] = ansi.ModeReset
}

// ShowCursor shows the terminal cursor.
//
// The [Terminal] manages the visibility of the cursor for you based on the
// [Viewport] used during [Terminal.Display]. This means that you don't need to
// call this unless you know what you're doing.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) ShowCursor() {
	t.showCursor()
}

func (t *Terminal) showCursor() {
	t.scr.ShowCursor()
	t.modes[ansi.TextCursorEnableMode] = ansi.ModeSet
}

// HideCursor hides the terminal cursor.
//
// The [Terminal] manages the visibility of the cursor for you based on the
// [Viewport] used during [Terminal.Display]. This means that you don't need to
// call this unless you know what you're doing.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) HideCursor() {
	t.hideCursor()
}

func (t *Terminal) hideCursor() {
	t.scr.HideCursor()
	t.modes[ansi.TextCursorEnableMode] = ansi.ModeReset
}

// SetTitle sets the title of the terminal window. This is typically used to
// set the title of the terminal window to the name of the application.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) SetTitle(title string) {
	_, _ = t.scr.WriteString(ansi.SetWindowTitle(title))
}

// Resize resizes the terminal screen buffer to the given width and height.
// This won't affect [Terminal.Size] or the terminal size, but it will resize
// the screen buffer used by the terminal.
func (t *Terminal) Resize(width, height int) error {
	// We need to reset the touched lines buffer to match the new height.
	t.buf.Touched = nil
	t.buf.Resize(width, height)
	t.scr.Resize(width, height)
	return nil
}

// Start prepares the terminal for use. It starts the input reader and
// initializes the terminal state. This should be called before using the
// terminal.
func (t *Terminal) Start() error {
	if t.started {
		return ErrStarted
	}

	if t.inTty == nil && t.outTty == nil {
		return ErrNotTerminal
	}

	// Store the initial terminal size.
	_, _, err := t.GetSize()
	if err != nil {
		return fmt.Errorf("error getting initial terminal size: %w", err)
	}

	// Initialize the terminal IO streams.
	if err := t.makeRaw(); err != nil {
		return fmt.Errorf("error entering raw mode: %w", err)
	}

	// Create a new context to manage input events.
	t.evctx, t.evcancel = context.WithCancel(context.Background())

	// Initialize input.
	cr, err := NewCancelReader(t.in)
	if err != nil {
		return fmt.Errorf("error creating cancel reader: %w", err)
	}
	t.cr = cr
	t.rd = NewTerminalReader(t.cr, t.termtype)
	t.rd.MouseMode = &t.mouseMode
	t.rd.SetLogger(t.logger)

	// Start the window size notifier if it is available.
	if t.winchn != nil {
		if err := t.winchn.Start(); err != nil {
			return fmt.Errorf("error starting window size notifier: %w", err)
		}
	}

	// Send the initial window size to the event channel.
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		var cells, pixels Size
		if t.winchn != nil {
			cells, pixels, _ = t.winchn.GetWindowSize()
		} else {
			w, h, _ := t.GetSize()
			cells = Size{Width: w, Height: h}
		}

		events := []Event{
			WindowSizeEvent(cells),
		}
		if pixels.Width > 0 && pixels.Height > 0 {
			events = append(events, WindowPixelSizeEvent(pixels))
		}
		for _, c := range events {
			select {
			case <-t.evctx.Done():
				return
			case t.evch <- c:
			}
		}
	}()

	if t.buf.Width() == 0 && t.buf.Height() == 0 {
		// If the buffer is not initialized, set it to the terminal size.
		t.buf.Resize(t.size.Width, t.size.Height)
		t.scr.Erase()
	}

	// We need to call [Terminal.optimizeMovements] before creating the screen
	// to populate [Terminal.useBspace] and [Terminal.useTabs].
	t.optimizeMovements()
	t.configureRenderer()

	if t.altscreen {
		t.enterAltScreen(true)
	} else if !t.cursorHidden {
		t.hideCursor()
	} else {
		t.showCursor()
	}
	// Restore terminal modes.
	for m, s := range t.modes {
		switch m {
		case ansi.TextCursorEnableMode, ansi.AltScreenSaveCursorMode:
			// These modes are handled by the renderer above.
			continue
		default:
			if s.IsSet() {
				t.scr.WriteString(ansi.SetMode(m)) //nolint:errcheck,gosec
			}
		}
	}
	// Restore fg, bg, cursor colors, and cursor shape.
	for _, c := range []struct {
		setter func(string) string
		colorp *color.Color
	}{
		{ansi.SetForegroundColor, &t.setFg},
		{ansi.SetBackgroundColor, &t.setBg},
		{ansi.SetCursorColor, &t.setCc},
	} {
		if c.colorp != nil && *c.colorp != nil {
			col, ok := colorful.MakeColor(*c.colorp)
			if ok {
				t.scr.WriteString(c.setter(col.Hex())) //nolint:errcheck,gosec
			}
		}
	}
	if t.curStyle > 1 {
		t.scr.WriteString(ansi.SetCursorStyle(t.curStyle)) //nolint:errcheck,gosec
	}

	return nil
}

// StreamEvents streams input events from the terminal to the event channel.
func (t *Terminal) StreamEvents(ctx context.Context, evch chan<- Event) error {
	errg, ctx := errgroup.WithContext(ctx)

	if t.winchn != nil {
		// Windows does not support SIGWINCH.
		winchc := make(chan Event)
		errg.Go(func() error { return t.winchn.StreamEvents(ctx, winchc) })
		errg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case ev := <-winchc:
					// We need to intercept window size events to update the
					// terminal size. We use the size to help us determine the
					// terminal window width so that we can truncate long lines
					// when using inline mode.
					switch ev := ev.(type) {
					case WindowSizeEvent:
						t.m.Lock()
						t.size = Size(ev)
						t.m.Unlock()
					case WindowPixelSizeEvent:
						t.m.Lock()
						t.pixSize = Size(ev)
						t.m.Unlock()
					}
					select {
					case <-ctx.Done():
						return nil
					case evch <- ev:
					}
				}
			}
		})
	}

	errg.Go(func() error { return t.rd.StreamEvents(ctx, t.evch) })
	errg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case ev := <-t.evch:
				select {
				case <-ctx.Done():
					return nil
				case evch <- ev:
				}
			}
		}
	})

	return errg.Wait() //nolint:wrapcheck
}

// Restore restores the terminal to its original state. This can be called
// after [Terminal.MakeRaw] to restore the terminal to its original state.
// It will also disable any modes that were enabled by the terminal, such as
// exiting the alternate screen buffer, showing the cursor, and resetting
// terminal modes.
//
// Most of the time, you don't need to call this manually, as it is called
// automatically when the terminal is shutdown or closed using [Terminal.Close]
// or [Terminal.Shutdown].
func (t *Terminal) Restore() error {
	if t.inTtyState != nil {
		if err := term.Restore(t.inTty.Fd(), t.inTtyState); err != nil {
			return fmt.Errorf("error restoring input terminal state: %w", err)
		}
		t.inTtyState = nil
	}
	if t.outTtyState != nil {
		if err := term.Restore(t.outTty.Fd(), t.outTtyState); err != nil {
			return fmt.Errorf("error restoring output terminal state: %w", err)
		}
		t.outTtyState = nil
	}
	t.started = false
	t.altscreen = t.modes.Get(ansi.AltScreenSaveCursorMode).IsSet()
	t.cursorHidden = t.modes.Get(ansi.TextCursorEnableMode).IsReset()
	if t.cursorHidden {
		t.showCursor()
		t.cursorHidden = false
	}
	if t.altscreen {
		t.exitAltScreen(false)
	}

	// Store the last known cursor position.
	x, y := t.scr.Position()
	t.cur = Pos(x, y)

	var buf bytes.Buffer
	for m, s := range t.modes {
		switch m {
		case ansi.TextCursorEnableMode, ansi.AltScreenSaveCursorMode:
			// These modes are handled by the renderer.
			continue
		}
		var reset bool
		ds, ok := defaultModes[m]
		if ok && s != ds {
			reset = s.IsSet() != ds.IsSet()
		} else {
			reset = s.IsSet()
		}
		if reset {
			buf.WriteString(ansi.ResetMode(m))
		}
	}
	if t.setFg != nil {
		buf.WriteString(ansi.ResetForegroundColor)
	}
	if t.setBg != nil {
		buf.WriteString(ansi.ResetBackgroundColor)
	}
	if t.setCc != nil {
		buf.WriteString(ansi.ResetCursorColor)
	}
	if t.curStyle > 1 {
		buf.WriteString(ansi.SetCursorStyle(0))
	}
	if _, err := t.scr.WriteString(buf.String()); err != nil {
		return fmt.Errorf("error resetting terminal modes: %w", err)
	}
	return t.scr.Flush()
}

// Shutdown restores the terminal to its original state and stops the event
// channel in a graceful manner.
// This waits for any pending events to be processed or the context to be
// done before closing the event channel.
func (t *Terminal) Shutdown(ctx context.Context) (rErr error) {
	defer func() {
		err := t.close(false)
		if rErr == nil {
			rErr = err
		}
	}()

	// Cancel the input reader.
	t.cr.Cancel()
	t.evcancel()

	var winchErr error
	if t.winchn != nil {
		winchErr = t.winchn.Stop()
	}

	if !t.altscreen {
		// Go to the bottom of the screen.
		t.scr.MoveTo(0, t.buf.Height()-1)
	}

	donec := make(chan struct{})
	go func() {
		defer close(donec)
		t.wg.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return err //nolint:wrapcheck
			} else if winchErr != nil {
				return err //nolint:wrapcheck
			}
			return nil
		case <-donec:
			return nil
		}
	}
}

// close closes any resources used by the terminal. This is typically used to
// close the terminal when it is no longer needed. When reset is true, it will
// also reset the terminal screen.
func (t *Terminal) close(reset bool) (rErr error) {
	t.evcancel()
	t.cr.Cancel()
	_ = t.cr.Close()
	err := t.Restore()
	if err != nil {
		rErr = fmt.Errorf("error restoring terminal state: %w", err)
	}
	if reset {
		// Reset screen.
		t.scr = NewTerminalRenderer(t.out, t.environ)
		t.configureRenderer()
	}

	return
}

// Close close any resources used by the terminal and restore the terminal to
// its original state.
func (t *Terminal) Close() error {
	return t.close(true)
}

// Events returns the event channel for the terminal. This channel is used to
// receive and send events from the terminal.
func (t *Terminal) Events() chan Event {
	return t.evch
}

// SendEvent is a helper function to send an event to the event channel. It
// blocks until the event is sent or the context is done. If the context is
// done, it will not send the event and will return immediately.
// This is useful to control the terminal from outside the event loop.
func (t *Terminal) SendEvent(ctx context.Context, ev Event) {
	select {
	case <-ctx.Done():
	case t.evch <- ev:
	}
}

// PrependString adds the given string to the top of the terminal screen. The
// string is split into lines and each line is added as a new line at the top
// of the screen. The added lines are not managed by the terminal and will not
// be cleared or updated by the [Terminal].
//
// This will truncate each line to the terminal width, so if the string is
// longer than the terminal width, it will be truncated to fit.
//
// Using this when the terminal is using the alternate screen or when occupying
// the whole screen may not produce any visible effects. This is because once
// the terminal writes the prepended lines, they will get overwritten by the
// next frame.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) PrependString(str string) error {
	// We truncate the string to the terminal width.
	var sb strings.Builder
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		if ansi.StringWidth(line) > t.size.Width {
			sb.WriteString(ansi.Truncate(line, t.size.Width, ""))
		} else {
			sb.WriteString(line)
		}
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	t.scr.PrependString(sb.String())
	return nil
}

// PrependLines adds lines of cells to the top of the terminal screen. The
// added line is unmanaged and will not be cleared or updated by the
// [Terminal].
//
// This will truncate each line to the terminal width, so if the string is
// longer than the terminal width, it will be truncated to fit.
//
// Using this when the terminal is using the alternate screen or when occupying
// the whole screen may not produce any visible effects. This is because once
// the terminal writes the prepended lines, they will get overwritten by the
// next frame.
func (t *Terminal) PrependLines(lines ...Line) error {
	truncatedLines := make([]Line, 0, len(lines))
	for _, l := range lines {
		// We truncate the line to the terminal width.
		if len(l) > t.size.Width {
			truncatedLines = append(truncatedLines, l[:t.size.Width])
		} else {
			truncatedLines = append(truncatedLines, l)
		}
	}

	t.scr.PrependLines(truncatedLines...)
	return nil
}

// Write writes the given bytes to the underlying terminal renderer.
// This is typically used to write arbitrary data to the terminal, usually
// escape sequences or control characters.
//
// This can affect the renderer state and the terminal screen, so it should be
// used with caution.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) Write(p []byte) (n int, err error) {
	return t.scr.Write(p)
}

// WriteString writes the given string to the underlying terminal renderer.
// This is typically used to write arbitrary data to the terminal, usually
// escape sequences or control characters.
//
// This can affect the renderer state and the terminal screen, so it should be
// used with caution.
//
// Note that this won't take any effect until the next [Terminal.Display] or
// [Terminal.Flush] call.
func (t *Terminal) WriteString(s string) (n int, err error) {
	return t.scr.WriteString(s)
}
