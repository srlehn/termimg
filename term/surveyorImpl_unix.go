//go:build unix

package term

import (
	"os"
	"os/signal"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"golang.org/x/sys/unix"
)

// TODO TIOCGWINSZ

func watchWINCH(tty TTY) (_ <-chan Resolution, closeFunc func() error, _ error) {
	if tty == nil {
		return nil, nil, errors.New(`nil tty`)
	}
	if !internal.IsDefaultTTY(tty.TTYDevName()) {
		// SIGWINCH is only sent to foreground processes of the terminal
		return nil, nil, errors.New(`not a foreground process of the tty`)
	}
	// ripped from github.com/mattn/go-tty/tty_unix.go (MIT license)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGWINCH)
	closeFunc = func() error {
		signal.Stop(sigs)
		close(sigs)
		return nil
	}
	ws := make(chan Resolution)
	go func() {
		defer close(ws)
		for {
			sig, ok := <-sigs
			if !ok {
				return
			}
			if sig != unix.SIGWINCH {
				continue
			}
			// don't block
			select {
			case ws <- Resolution{}:
			default:
			}
		}
	}()
	return ws, closeFunc, nil
}

func (s *SurveyorDefault) ResizeEvents(tty TTY) (_ <-chan Resolution, closeFunc func() error, _ error) {
	return watchWINCH(tty)
}
func (s *SurveyorNoANSI) ResizeEvents(tty TTY) (_ <-chan Resolution, closeFunc func() error, _ error) {
	return watchWINCH(tty)
}
func (s *SurveyorNoTIOCGWINSZ) ResizeEvents(tty TTY) (_ <-chan Resolution, closeFunc func() error, _ error) {
	return watchWINCH(tty)
}
