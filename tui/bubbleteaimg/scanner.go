//go:build dev

package bubbleteaimg

import (
	"image"
	"slices"
	"strings"
	"time"

	"github.com/muesli/reflow/ansi"
	"github.com/srlehn/termimg/internal/util"
	"github.com/srlehn/termimg/tty/bubbleteatty"
)

var (
	csi     = "\033["
	csiTerm = `y`
)

var _ bubbleteatty.Scanner = (*scanner)(nil)

type scanner struct {
	markers        map[string][2]int
	rects          map[string]image.Rectangle
	postWriteFuncs []struct { // ordered list of image widget draw funcs, etc
		id string
		f  func(bounds image.Rectangle)
	}
}

func newScanner() *scanner {
	return &scanner{
		markers: make(map[string][2]int),
		rects:   make(map[string]image.Rectangle),
	}
}

func (c *scanner) Scan(b []byte) []byte {
	if c == nil {
		return b
	}
	s := util.BytesToString(b)
	var replArgs []string
	for i := 0; i != -1; i++ {
		time.Sleep(100 * time.Millisecond)
		i2 := strings.Index(s[i:], csi)
		if i2 == -1 {
			break
		}
		i += i2
		j := strings.Index(s[i:], csiTerm)
		if j == -1 {
			break
		}
		j += i
		id := s[i+len(csi) : j+1-len(csiTerm)]
		replArgs = append(replArgs, s[i:j+1], ``)
		var v [2]int
		v, ok := c.markers[id]
		if !ok {
			v[0] = i
		} else {
			v[1] = i
		}
		c.markers[id] = v
	}
	c.getRects(s)
	s = strings.NewReplacer(replArgs...).Replace(s)
	return util.StringToBytes(s)
}

func (c *scanner) getRects(s string) {
	if c == nil || c.markers == nil {
		return
	}
	if c.rects == nil {
		c.rects = make(map[string]image.Rectangle)
	}
outer:
	for id, p := range c.markers {
		var x, y [2]int
		for i := 0; i < 2; i++ {
			if p[i] < 0 || p[i] > len(s) {
				continue outer
			}
			l := strings.LastIndex(s[:p[i]], "\n")
			if l == -1 {
				l = 0
			}
			x[i] = ansi.PrintableRuneWidth(s[l+1 : p[i]])
			y[i] = strings.Count(s[:p[i]], "\n") + 1
		}
		c.rects[id] = image.Rect(x[0], y[0], x[1], y[1])
	}
}

func (s *scanner) PostWrite() {
	if s == nil {
		return
	}
	defer func() { s.rects = nil; s.postWriteFuncs = nil }()
	if s.rects == nil {
		return
	}
	for _, pwf := range s.postWriteFuncs {
		bounds, ok := s.rects[pwf.id]
		if !ok || pwf.f == nil {
			continue
		}
		pwf.f(bounds)
	}
}

// setAfterWriteFunc sets a temporary functions that will be called once after the next Write.
func (s *scanner) setAfterWriteFunc(id string, f func(bounds image.Rectangle)) {
	if s == nil {
		return
	}
	s.postWriteFuncs = append(
		// remove previously planned draws for this widget id
		slices.DeleteFunc(s.postWriteFuncs, func(r struct {
			id string
			f  func(bounds image.Rectangle)
		}) bool {
			return r.id == id
		}),
		struct {
			id string
			f  func(bounds image.Rectangle)
		}{id: id, f: f},
	)
}
