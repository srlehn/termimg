package qdefault_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/srlehn/termimg/internal/dummytty"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/queries"
	"github.com/srlehn/termimg/query/qdefault"
)

func TestQDefaultQuery(t *testing.T) {
	query := queries.CSI + `0c`
	replDummy := queries.CSI + `?65;1;9c`
	tty, err := dummytty.New(replDummy)
	if err != nil {
		t.Fatal(err)
	}
	qu := qdefault.NewQuerier()
	repl, err := qu.Query(query, tty, parser.StopOnC)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, replDummy, repl)
}
