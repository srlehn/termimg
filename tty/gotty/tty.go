// <Copyright> 2018,2019 Simon Robin Lehn. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// Package gotty provides an implementation of term.TTY via [github.com/mattn/go-tty].
//
// [github.com/mattn/go-tty]: https://pkg.go.dev/github.com/mattn/go-tty
package gotty

import (
	"sync"

	ttymattn "github.com/mattn/go-tty"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyMattN struct {
	*ttymattn.TTY
	fileName       string
	winch          chan term.Resolution
	watchWINCHOnce sync.Once
	buf            []byte
}

var _ term.TTY = (*ttyMattN)(nil)

func New(ttyFile string) (term.TTY, error) {
	t, err := ttymattn.OpenDevice(ttyFile)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New(`nil tty`)
	}
	return &ttyMattN{TTY: t, fileName: ttyFile}, nil
}

func (t *ttyMattN) Write(b []byte) (n int, err error) {
	if t == nil {
		return 0, errors.New(consts.ErrNilReceiver)
	}
	if t.TTY == nil {
		return 0, errors.New(`nil tty`)
	}
	f := t.Output()
	if f == nil {
		return 0, errors.New(`nil file`)
	}
	return f.Write(b)
}

func (t *ttyMattN) Read(p []byte) (n int, err error) {
	if t == nil || t.TTY == nil {
		return 0, errors.New(consts.ErrNilReceiver)
	}
	p = p[:0]
	i := len(t.buf)
	if i > 0 {
		if i >= cap(p) {
			copy(p, t.buf[:cap(p)-1])
			t.buf = t.buf[cap(p)-1:]
			return len(p), nil
		}
		t.buf = nil
		copy(p, t.buf)
	}
	for ; i < cap(p); i++ {
		r, err := t.TTY.ReadRune()
		if err != nil {
			return len(p), errors.New(err)
		}
		b := []byte(string(r))
		l := min(cap(p)-len(p), len(b))
		b1, b2 := b[:l], b[l:]
		if len(b2) > 0 {
			t.buf = b2
		}
		p = append(p, b1...)
		if len(p) == cap(p) {
			break
		}
	}

	return len(p), nil
}

func (t *ttyMattN) ReadRune() (r rune, size int, err error) {
	r = '\uFFFD'
	if t == nil {
		return r, len(string(r)), errors.New(consts.ErrNilReceiver)
	}
	if t.TTY == nil {
		return r, len(string(r)), errors.New(`nil tty`)
	}
	r, err = t.TTY.ReadRune()
	if err != nil {
		r = '\uFFFD'
	}
	return r, len(string(r)), err
}

func (t *ttyMattN) TTYDevName() string {
	if t == nil {
		return internal.DefaultTTYDevice()
	}
	return t.fileName
}

// Close ...
func (t *ttyMattN) Close() error {
	if t == nil || t.TTY == nil {
		return nil
	}
	defer func() { t.TTY = nil }()
	return t.TTY.Close()
}

// ResizeEvents ...
func (t *ttyMattN) ResizeEvents() (_ <-chan term.Resolution, closeFunc func() error, _ error) {
	if t == nil || t.TTY == nil {
		return nil, nil, errors.New(consts.ErrNilReceiver)
	}
	if t.winch != nil {
		return t.winch, nil, nil
	}
	var errRet error
	t.watchWINCHOnce.Do(func() {
		winchMattN := t.TTY.SIGWINCH()
		if winchMattN == nil {
			errRet = errors.New(`unable to receive resize events`)
			return
		}
		t.winch = make(chan term.Resolution)
		closeOnce := sync.Once{}
		closeFunc = func() error {
			closeOnce.Do(func() { close(t.winch) })
			return nil
		}
		go func() {
			defer closeFunc()
			for {
				winchEv, ok := <-winchMattN
				if !ok {
					break
				}
				// don't block
				select {
				case t.winch <- term.Resolution{TermInCellsW: uint(winchEv.W), TermInCellsH: uint(winchEv.H)}:
				default:
				}
			}
		}()
	})
	if errRet == nil && t.winch == nil {
		errRet = errors.New(`unable to receive resize events`)
	}
	return t.winch, closeFunc, errRet
}
