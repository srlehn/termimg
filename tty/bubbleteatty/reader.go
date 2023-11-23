//go:build dev

package bubbleteatty

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/srlehn/termimg/internal/errors"
)

const (
	bufMaxSize = 10 * 1 << 20
	ioTimeOut  = 100 * time.Millisecond
)

func newTeaReader(r io.Reader) *teaReader {
	if r == nil {
		return nil
	}
	t := &teaReader{
		r:        r,
		requChan: make(chan uint),
	}
	go t.handleRequests()
	return t
}

type teaReader struct {
	r        io.Reader
	subs     []*subReader
	wChans   []chan<- []byte
	requChan chan uint
}

func (t *teaReader) read(l uint) error {
	if t == nil || t.r == nil {
		return errors.NilReceiver()
	}
	wg := &sync.WaitGroup{}
	b := make([]byte, l)
	n, err := t.r.Read(b)
	if n > 0 {
		for _, wChan := range t.wChans {
			if wChan == nil {
				continue
			}
			wg.Add(1)
			go func(wChan chan<- []byte) {
				defer wg.Done()
				select {
				case wChan <- b[:n]:
				case <-time.After(ioTimeOut):
				}
			}(wChan)
		}
	}
	wg.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (t *teaReader) handleRequests() {
	if t == nil {
		return
	}
	for {
		l, ok := <-t.requChan
		if !ok {
			return
		}
		_ = t.read(l)
	}
}

type subReader struct {
	buf         *bytes.Buffer
	rChan       <-chan []byte
	requChan    chan<- uint
	newReadChan chan struct{}
}

func (t *teaReader) NewSubReader() io.Reader {
	if t == nil {
		return nil
	}
	rwChan := make(chan []byte, 1)
	s := &subReader{
		buf:         &bytes.Buffer{},
		rChan:       rwChan,
		requChan:    t.requChan,
		newReadChan: make(chan struct{}),
	}
	t.subs = append(t.subs, s)
	t.wChans = append(t.wChans, rwChan)
	go s.receiveInput()
	return s
}

func (s *subReader) receiveInput() {
	if s == nil || s.buf == nil || s.rChan == nil || s.newReadChan == nil {
		return
	}
	for {
		if s.buf.Cap() > bufMaxSize {
			s.buf = &bytes.Buffer{}
		}
		b, ok := <-s.rChan
		if !ok {
			return
		}
		if _, err := s.buf.Write(b); err != nil {
			s.buf.Reset()
			s.buf.Write(b)
		}
		select {
		case s.newReadChan <- struct{}{}:
		default:
		}
	}
}

func (s *subReader) Read(p []byte) (n int, err error) {
	n, err = s.buf.Read(p)
	if err == nil && n == len(p) {
		return
	}
	select {
	case s.requChan <- uint(len(p) - n):
	default:
	}
	select {
	case <-s.newReadChan:
	case <-time.After(ioTimeOut):
	}
	n2, err := s.buf.Read(p[n:])
	if err == nil && n+n2 == len(p) {
		return n + n2, nil
	}
	return n + n2, err
}

/*
// TODO
termenv.WithUnsafe() termenv.OutputOption
lipgloss.NewRenderer(w io.Writer, opts ...termenv.OutputOption) *lipgloss.Renderer
(s lipgloss.Style) Renderer(r *lipgloss.Renderer) lipgloss.Style

https://pkg.go.dev/github.com/muesli/termenv#Environ
type Environ interface {
	Environ() []string
	Getenv(string) string
}
add Getenv() to type Enver interface {Environ() []string, ...}
add (*Terminal).Env() Enver // filtered properties field

termenvOpts := []termenv.OutputOption{
	termenv.WithProfile(termenv.EnvColorProfile()),
	termenv.WithUnsafe(),
	termenv.WithEnvironment((*term.Terminal).Env()),
	termenv.WithColorCache(true),
)
renderer := NewRenderer(w, ...termenvOpts)
style.Copy()
*/
