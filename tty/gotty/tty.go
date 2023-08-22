// <Copyright> 2018,2019 Simon Robin Lehn. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// Package gotty provides an implementation of term.TTY via [github.com/mattn/go-tty].
//
// [github.com/mattn/go-tty]: https://pkg.go.dev/github.com/mattn/go-tty
package gotty

import (
	ttymattn "github.com/mattn/go-tty"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/term"
)

type ttyMattN struct {
	*ttymattn.TTY
	fileName string
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
