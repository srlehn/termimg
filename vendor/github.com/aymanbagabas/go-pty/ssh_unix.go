//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package pty

import (
	"fmt"

	"github.com/u-root/u-root/pkg/termios"
	"golang.org/x/crypto/ssh"
)

func applyTerminalModesToFd(fd int, width int, height int, modes ssh.TerminalModes) error {
	// Get the current TTY configuration.
	tios, err := termios.GTTY(int(fd))
	if err != nil {
		return fmt.Errorf("GTTY: %w", err)
	}

	// Apply the modes from the SSH request.
	tios.Row = height
	tios.Col = width

	for c, v := range modes {
		if c == ssh.TTY_OP_ISPEED {
			tios.Ispeed = int(v)
			continue
		}
		if c == ssh.TTY_OP_OSPEED {
			tios.Ospeed = int(v)
			continue
		}
		k, ok := terminalModeFlagNames[c]
		if !ok {
			continue
		}
		if _, ok := tios.CC[k]; ok {
			tios.CC[k] = uint8(v)
			continue
		}
		if _, ok := tios.Opts[k]; ok {
			tios.Opts[k] = v > 0
			continue
		}
	}

	// Save the new TTY configuration.
	if _, err := tios.STTY(int(fd)); err != nil {
		return fmt.Errorf("STTY: %w", err)
	}

	return nil
}
