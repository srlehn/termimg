# termimg Library Architecture

## Overview

**termimg** is a Go library for displaying images in terminal emulators.
It provides a unified API that automatically detects terminal capabilities
and selects the optimal image display protocol.
The library uses **cell coordinates** (not pixel coordinates) for positioning
and can fit images into cell boundaries while preserving aspect ratio.

## Status & Design Philosophy

- **Status**: Experimental (API may change)
- **Positioning**: Uses cell coordinates with origin at upper-left terminal corner
- **Modularity**: Highly modular design with exchangeable components
- **Protocol Support**: Multiple drawing methods with automatic best-fit selection
- **TUI Integration**: Compatible with common TUI frameworks via TTY multiplexing (incomplete)

## Core Architecture

### Main Package (Root)

- **Import Path**: `github.com/srlehn/termimg`
- **Purpose**: Public API and configuration

#### Key Functions

- `Draw(img image.Image, bounds image.Rectangle) error` - Draw image to terminal
- `DrawFile(imgFile string, bounds image.Rectangle) error` - Draw image from file
- `DrawBytes(imgBytes []byte, bounds image.Rectangle) error` - Draw image from bytes
- `NewImage(img image.Image) *term.Image` - Create image instance
- `NewImageBytes(imgBytes []byte) *term.Image` - Create from bytes (requires decoder registration)
- `NewImageFileName(imgfile string) *term.Image` - Create from file path
- `Query(qs string, p term.Parser) (string, error)` - Send terminal query
- `Terminal() (*term.Terminal, error)` - Get terminal instance
- `CleanUp() error` - Resource cleanup

#### Configuration

```go
var DefaultConfig = term.Options{
    term.SetPTYName(ttyDefault),
    term.SetTTYProvider(ttyProvider, false),
    term.SetQuerier(querier, true),
    term.SetWindowProvider(wm.SetImpl(wmImplementation), true),
    term.SetResizer(resizer),
}
```

#### Default Providers

- `ttyDefault` - `internal.DefaultTTYDevice()`
- `ttyProvider` - `gotty.New`
- `querier` - `qdefault.NewQuerier()`
- `wmImplementation` - `wmimpl.Impl()`
- `resizer` - `&rdefault.Resizer{}`

## Core Packages

### term/ - Terminal Abstraction Layer

#### Terminal Type

- `NewTerminal(opts ...Option) (*Terminal, error)` - Create terminal with options
- `Draw(img image.Image, bounds image.Rectangle) error` - Draw image
- `Query(qs string, p Parser) (string, error)` - Send terminal query
- `NewCanvas(bounds image.Rectangle) (*Canvas, error)` - Create drawing canvas
- `SizeInCells() (width, height uint, err error)` - Get terminal size in cells
- `SizeInPixels() (width, height uint, err error)` - Get terminal size in pixels
- `CellSize() (width, height float64, _ error)` - Get cell dimensions

#### Image Type

- `NewImage(img image.Image) *Image` - Create from Go image
- `NewImageBytes(imgBytes []byte) *Image` - Create from byte data
- `NewImageFilename(imgFile string) *Image` - Create from file path
- `Fit(bounds image.Rectangle, rsz Resizer, sv Surveyor) error` - Resize to fit bounds
- `Decode() error` - Decode image data
- `SaveAsFile(t *Terminal, fileExt string, enc ImageEncoder) (rm func() error, err error)` - Save to temp file

#### Canvas Type

- `Draw(img image.Image) error` - Draw image to canvas
- `SetImage(img image.Image) error` - Set canvas image
- `Screenshot() (image.Image, error)` - Take screenshot
- `Video(ctx context.Context, vid <-chan image.Image, frameDur time.Duration) error` - Play video

#### Configuration Options

- `SetPTYName(ptyName string) Option` - Set PTY device name
- `SetTTYProvider[T TTY, F func(ptyName string) (T, error)](ttyProv F, enforce bool) Option` - Set TTY provider
- `SetQuerier(qu Querier, enforce bool) Option` - Set terminal querier
- `SetResizer(rsz Resizer) Option` - Set image resizer
- `SetWindowProvider(wProv wm.WindowProvider, enforce bool) Option` - Set window provider
- `SetDrawers(drs []Drawer) Option` - Set available drawers
- `TUIMode Option` - Enable TUI mode
- `CLIMode Option` - Enable CLI mode

#### Interfaces

**Drawer Interface**

```go
type Drawer interface {
    Name() string
    New() Drawer
    IsApplicable(DrawerCheckerInput) (bool, environ.Properties)
    Draw(img image.Image, bounds image.Rectangle, term *Terminal) error
    Prepare(ctx context.Context, img image.Image, bounds image.Rectangle, term *Terminal) (drawFn func() error, _ error)
}
```

**TTY Interface**

```go
type TTY interface {
    io.ReadWriteCloser
    TTYDevName() string
    // Optional: ResizeEvents(), SizePixel(), ReadRune()
}
```

**Querier Interface**

```go
type Querier interface {
    Query(string, TTY, Parser) (string, error)
}
```

**Resizer Interface**

```go
type Resizer interface {
    Resize(img image.Image, size image.Point) (image.Image, error)
}
```

### tty/ttymux/ - TTY Multiplexing System

#### TTYMultiplexer

- `New(ttyPath string) (*TTYMultiplexer, error)` - Create multiplexer
- `NewWithOpener(ttyPath string, opener func(string) (*os.File, error)) (*TTYMultiplexer, error)` - Create with custom opener
- `NewReader(prefix string) *MultiplexedTTY` - Create reader
- `NewReaderWithID(id string) *MultiplexedTTY` - Create reader with ID
- `CreatePTYSlave() (*PTYSlaveWrapper, error)` - Create PTY slave
- `Close() error` - Shutdown multiplexer

#### MultiplexedTTY (implements term.TTY)

- `Read(p []byte) (n int, err error)` - Read from buffer
- `Write(p []byte) (n int, err error)` - Write to original TTY
- `Available() int` - Available bytes in buffer
- `SizePixel() (cw int, ch int, pw int, ph int, e error)` - Terminal size info

#### BufferedReader

- `Read(p []byte) (n int, err error)` - Blocking read with buffering
- `Available() int` - Available bytes
- `Close() error` - Close reader

### wm/ - Window Manager Integration

#### Interfaces

```go
type Window interface {
    WindowFind() error
    WindowName() string
    WindowClass() string
    WindowID() uint64
    Screenshot() (image.Image, error)
    Close() error
}

type Connection interface {
    Windows() ([]Window, error)
    DisplayImage(img image.Image, windowName string)
    Resources() (environ.Properties, error)
    Close() error
}
```

#### Functions

- `NewConn(env environ.Properties) (Connection, error)` - Create connection
- `CreateWindow(name, class, instance string) Window` - Create window
- `SetImpl(impl Implementation) WindowProvider` - Set implementation

## Specialized Packages

### drawers/ - Image Display Protocols

- `sixel/` - Sixel graphics protocol
- `kitty/` - Kitty terminal graphics protocol
- `iterm2/` - iTerm2 inline images
- `w3mimgdisplay/` - w3m image display
- `urxvt/` - urxvt pixbuf extension
- `terminology/` - Terminology image protocol
- `framebuffer/` - Direct framebuffer access
- `x11/` - X11 window system
- `domterm/` - DomTerm HTML rendering
- `generic/`, `generic2/` - Generic fallbacks
- `gdiplus/` - Windows GDI+
- `sane/` - Sane choice of image protocols
- `all/` - All available drawers

### tty/ - TTY Implementations

- `creacktty/` - Uses github.com/creack/pty
- `gotty/` - Default TTY implementation
- `bubbleteatty/` - Bubble Tea integration
- `tcelltty/` - tcell integration
- `ttyhook/` - TTY hooking system
- `ttymux/` - TTY multiplexing (documented above)
- `dumbtty/` - Simple TTY implementation
- `uroottty/` - u-root integration
- `bagabastty/` - Custom TTY implementation
- `contdtty/` - Container TTY
- `pkgterm/` - Package terminal

### query/ - Terminal Querying

- `qdefault/` - `NewQuerier() term.Querier` - Default querier implementation

### resize/ - Image Resizing

- `rdefault/` - Default resizer implementation
- `gift/` - Uses github.com/disintegration/gift
- `imaging/` - Uses github.com/kovidgoyal/imaging
- `bild/` - Uses github.com/anthonynsimon/bild
- `caire/` - Uses github.com/esimov/caire
- `nfnt/` - Uses github.com/nfnt/resize
- `rez/` - Uses github.com/bamiaux/rez

### tui/ - TUI Framework Integrations

- `termuiimg/` - termui integration
  - `NewImage(tm *term.Terminal, img image.Image, bounds image.Rectangle) (*Image, error)`
  - `Draw(buf *termui.Buffer)` - Render to termui buffer
- `bubbleteaimg/` - Bubble Tea integration
- `tcellimg/` - tcell integration
- `tviewimg/` - tview integration

### terminals/ - Terminal Detection

- Terminal-specific implementations for detection and configuration
- Files: `alacritty.go`, `kitty.go`, `xterm.go`, `konsole.go`, `foot.go`, `mintty.go`, etc.

### Additional Packages

- `mux/` - Terminal multiplexer handling
- `pty/` - Pseudo-terminal utilities
- `video/ffmpeg/` - FFmpeg video processing
- `env/` - Environment detection
- `cmd/` - Command-line tools and examples

## Registration Pattern

The library uses Go's init() function pattern where implementation packages register themselves:

1. Import side-effect packages (e.g., `_ "github.com/srlehn/termimg/drawers/all"`)
2. Packages register implementations during init()
3. Main package provides unified access through registered implementations
4. Runtime selection based on terminal capabilities and environment

## Data Flow

1. **Terminal Detection** - Identify terminal type and capabilities
2. **Drawer Selection** - Choose optimal image display protocol
3. **Image Processing** - Resize and encode image as needed
4. **Protocol Execution** - Send image data using selected drawer
5. **TTY Multiplexing** - Handle concurrent terminal access if needed

## TTY Multiplexing (Work in Progress)

The library is developing a TTY multiplexing system to enable concurrent access to `/dev/tty` by both termimg and TUI libraries.
This addresses the fundamental problem where multiple libraries cannot simultaneously read from the same TTY device.

### Current Development Plans

**Three Approaches Being Explored:**

1. **TTY Multiplexer (`tty/ttymux/`)**
   - Buffer-based multiplexer with `TTYMultiplexer`, `MultiplexedTTY`, `BufferedReader`
   - Single reader goroutine distributes input to multiple buffered channels
   - Ensures no input loss through comprehensive buffering
   - Status: Partially implemented

2. **TTY Hook System (`tty/ttyhook/`)**
   - Function hooking using `github.com/agiledragon/gohook`
   - Intercepts `os.Open("/dev/tty")` calls with monkey patching
   - Returns PTY slaves while termimg keeps real TTY access
   - Status: Experimental design phase

3. **PTY Mirror System**
   - Creates PTY pairs dynamically when `Mirror()` called
   - Fan-out approach: real TTY → copy goroutine → multiple PTY masters
   - Each library gets independent PTY slave for reading
   - Status: Planning phase

### Problem Being Solved

**Challenge**: TUI libraries (termui, bubbletea, tcell) and termimg both need to read from `/dev/tty` simultaneously for:

- Terminal capability queries (termimg)
- User input handling (TUI library)
- Resize event monitoring
- Mouse/keyboard event processing

**Current Limitation**: Only one process can effectively read from `/dev/tty` - others will miss input or block.

**Multiplexing Solution**: Create isolated input streams for each library while preserving all input data.

## Key Design Patterns

- **Strategy Pattern** - Multiple drawer implementations for different terminals
- **Factory Pattern** - Terminal creation with configurable providers
- **Registry Pattern** - Dynamic registration of implementations
- **Adapter Pattern** - Unified interface for different TTY types
- **Observer Pattern** - Terminal resize event handling
- **Command Pattern** - Terminal query/response system
- **Multiplexer Pattern** - TTY input fan-out to multiple consumers (in development)
