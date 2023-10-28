package iointernal_test

import (
	"io"
	"strings"
	"testing"

	"github.com/srlehn/termimg/internal/iointernal"
	"github.com/stretchr/testify/assert"
)

func TestRuneReader(t *testing.T) {
	s := []rune(`7ä⌘🤘🌵`)
	rdr := iointernal.NewRuneReader(strings.NewReader(string(s)))
	var repl []rune
	for {
		r, _, err := rdr.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		repl = append(repl, r)
	}
	assert.Equal(t, s, repl)
}
