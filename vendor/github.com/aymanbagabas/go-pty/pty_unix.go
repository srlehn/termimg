//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package pty

import (
	"context"
	"errors"
	"os"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// unixPty is a POSIX compliant Unix pseudo-terminal.
// See: https://pubs.opengroup.org/onlinepubs/9699919799/
type unixPty struct {
	master, slave *os.File
	closed        bool
}

var _ Pty = &unixPty{}

// Close implements Pty.
func (p *unixPty) Close() error {
	if p.closed {
		return nil
	}
	defer func() {
		p.closed = true
	}()
	return errors.Join(p.master.Close(), p.slave.Close())
}

// Command implements Pty.
func (p *unixPty) Command(name string, args ...string) *Cmd {
	c := &Cmd{
		pty:  p,
		Path: name,
		Args: append([]string{name}, args...),
	}
	return c
}

// CommandContext implements Pty.
func (p *unixPty) CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	c := p.Command(name, args...)
	c.ctx = ctx
	return c
}

// Name implements Pty.
func (p *unixPty) Name() string {
	return p.slave.Name()
}

// Read implements Pty.
func (p *unixPty) Read(b []byte) (n int, err error) {
	return p.master.Read(b)
}

// Control implements UnixPty.
func (p *unixPty) Control(f func(fd uintptr)) error {
	return p.control(f)
}

func (p *unixPty) control(f func(fd uintptr)) error {
	conn, err := p.master.SyscallConn()
	if err != nil {
		return err
	}
	return conn.Control(f)
}

// Master implements UnixPty.
func (p *unixPty) Master() *os.File {
	return p.master
}

// Slave implements UnixPty.
func (p *unixPty) Slave() *os.File {
	return p.slave
}

// Winsize represents the terminal window size.
type Winsize = unix.Winsize

// SetWinsize implements UnixPty.
func (p *unixPty) SetWinsize(ws *Winsize) error {
	var ctrlErr error
	if err := p.control(func(fd uintptr) {
		ctrlErr = unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, ws)
	}); err != nil {
		return err
	}

	return ctrlErr
}

// Resize implements Pty.
func (p *unixPty) Resize(width int, height int) error {
	return p.SetWinsize(&Winsize{
		Row: uint16(height),
		Col: uint16(width),
	})
}

// Write implements Pty.
func (p *unixPty) Write(b []byte) (n int, err error) {
	return p.master.Write(b)
}

// Fd implements Pty.
func (p *unixPty) Fd() uintptr {
	return p.master.Fd()
}

func newPty() (UnixPty, error) {
	master, slave, err := pty.Open()
	if err != nil {
		return nil, err
	}

	return &unixPty{
		master: master,
		slave:  slave,
	}, nil
}
