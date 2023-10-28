package tcelltty

import (
	"github.com/gdamore/tcell/v2"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

// TODO set ONLCR so that no \r is needed at line end

type ttyTCell struct {
	tcell.Tty
	screen   tcell.Screen
	fileName string
}

var _ term.TTY = (*ttyTCell)(nil)
var _ term.TTYProvider = New

func New(ttyFile string) (term.TTY, error) {
	ttyTC, err := tcell.NewDevTtyFromDev(ttyFile)
	if err != nil {
		return nil, err
	}
	if err := ttyTC.Start(); err != nil {
		return nil, err
	}
	tty := &ttyTCell{
		Tty:      ttyTC,
		fileName: ttyFile,
	}
	return tty, nil
}

func (t *ttyTCell) TCellScreen() (tcell.Screen, error) {
	if t == nil {
		return nil, errors.NilReceiver()
	}
	if t.screen != nil {
		return t.screen, nil
	}
	scr, err := tcell.NewTerminfoScreenFromTty(t.Tty)
	if err != nil {
		return nil, err
	}
	t.screen = scr

	return scr, nil
}

func (t *ttyTCell) Write(b []byte) (n int, err error) {
	if t == nil || t.Tty == nil {
		return 0, errors.NilReceiver()
	}
	return t.Tty.Write(b)
}

func (t *ttyTCell) Read(p []byte) (n int, err error) {
	if t == nil || t.Tty == nil {
		return 0, errors.NilReceiver()
	}
	// TODO read key events instead?
	return t.Tty.Read(p)
}

func (t *ttyTCell) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

func (t *ttyTCell) SizePixel() (cw int, ch int, pw int, ph int, e error) {
	if t == nil || t.Tty == nil {
		return 0, 0, 0, 0, errors.NilReceiver()
	}
	if t.screen == nil {
		if _, err := t.TCellScreen(); err != nil {
			return 0, 0, 0, 0, err
		}
	}
	cw, ch = t.screen.Size()
	return
}

// Close ...
func (t *ttyTCell) Close() error {
	if t == nil || t.Tty == nil {
		return nil
	}
	defer func() { t.Tty = nil }()
	errStop := t.Tty.Stop()
	errClose := t.Tty.Close()
	err := errors.Join(errStop, errClose)
	if err != nil {
		return errors.New(err)
	}
	return nil
}
