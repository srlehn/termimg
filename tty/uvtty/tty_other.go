//go:build !unix

package uvtty

import "github.com/srlehn/termimg/internal/errors"

// getWindowSize implements the non-Unix fallback version
func (t *TTYUV) getWindowSize() (cw int, ch int, pw int, ph int, e error) {
	if t == nil || t.UVTerminal == nil {
		return 0, 0, 0, 0, errors.NilReceiver()
	}

	// Fallback to UV's GetSize for cell dimensions only
	if currentW, currentH, err := t.UVTerminal.GetSize(); err == nil {
		t.mu.Lock()
		t.cellW, t.cellH = currentW, currentH
		t.mu.Unlock()

		t.mu.RLock()
		cw, ch = t.cellW, t.cellH
		pw, ph = t.pixelW, t.pixelH
		t.mu.RUnlock()

		return cw, ch, pw, ph, nil
	}

	return 0, 0, 0, 0, errors.New("unable to get terminal size")
}
