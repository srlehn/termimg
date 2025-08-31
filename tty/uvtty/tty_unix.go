//go:build unix

package uvtty

import (
	"github.com/charmbracelet/x/termios"
	"github.com/srlehn/termimg/internal/errors"
)

// getWindowSize implements the Unix-specific version using termios
func (t *TTYUV) getWindowSize() (cw int, ch int, pw int, ph int, e error) {
	if t == nil || t.outFile == nil {
		return 0, 0, 0, 0, errors.NilReceiver()
	}

	winsize, err := termios.GetWinsize(int(t.outFile.Fd()))
	if err != nil {
		return 0, 0, 0, 0, errors.New(err)
	}

	return int(winsize.Col), int(winsize.Row), int(winsize.Xpixel), int(winsize.Ypixel), nil
}
