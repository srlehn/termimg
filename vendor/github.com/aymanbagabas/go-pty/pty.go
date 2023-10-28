package pty

import (
	"context"
	"errors"
	"io"
	"os"
)

var (
	// ErrInvalidCommand is returned when the command is invalid.
	ErrInvalidCommand = errors.New("pty: invalid command")

	// ErrUnsupported is returned when the platform is unsupported.
	ErrUnsupported = errors.New("pty: unsupported platform")
)

// New returns a new pseudo-terminal.
func New() (Pty, error) {
	return newPty()
}

// Pty is a pseudo-terminal interface.
type Pty interface {
	io.ReadWriteCloser

	// Name returns the name of the pseudo-terminal.
	// On Windows, this will always be "windows-pty".
	// On Unix, this will return the name of the slave end of the
	// pseudo-terminal TTY.
	Name() string

	// Command returns a command that can be used to start a process
	// attached to the pseudo-terminal.
	Command(name string, args ...string) *Cmd

	// CommandContext returns a command that can be used to start a process
	// attached to the pseudo-terminal.
	CommandContext(ctx context.Context, name string, args ...string) *Cmd

	// Resize resizes the pseudo-terminal.
	Resize(width int, height int) error

	// Fd returns the file descriptor of the pseudo-terminal.
	// On Unix, this will return the file descriptor of the master end.
	// On Windows, this will return the handle of the console.
	Fd() uintptr
}

// UnixPty is a Unix pseudo-terminal interface.
type UnixPty interface {
	Pty

	// Master returns the pseudo-terminal master end (pty).
	Master() *os.File

	// Slave returns the pseudo-terminal slave end (tty).
	Slave() *os.File

	// Control calls f on the pseudo-terminal master end (pty).
	Control(f func(fd uintptr)) error

	// SetWinsize sets the pseudo-terminal window size.
	SetWinsize(ws *Winsize) error
}

// ConPty is a Windows ConPTY interface.
type ConPty interface {
	Pty

	// InputPipe returns the ConPty input pipe.
	InputPipe() *os.File

	// OutputPipe returns the ConPty output pipe.
	OutputPipe() *os.File
}
