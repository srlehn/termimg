package iointernal

import (
	"io"
	"unicode/utf8"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/errors"
)

type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

func NewRuneReader(rdr io.Reader) RuneReader {
	if rdr == nil {
		return nil
	}
	var readRuneFunc func() (rn rune, size int, err error)
	if rnRdr, ok := rdr.(RuneReader); ok {
		readRuneFunc = rnRdr.ReadRune
	}
	return &runeReader{
		reader:       rdr,
		readRuneFunc: readRuneFunc,
	}
}

var _ RuneReader = (*runeReader)(nil)

type runeReader struct {
	reader       io.Reader
	buf          []byte
	readRuneFunc func() (rn rune, size int, err error)
}

func (r *runeReader) ReadRune() (rn rune, size int, err error) {
	rn = utf8.RuneError // '\uFFFD'
	defer func() {
	}()
	if r == nil {
		return rn, len(string(rn)), errors.New(consts.ErrNilReceiver)
	}
	if r.readRuneFunc != nil {
		return r.readRuneFunc()
	}
	if r.reader == nil {
		return rn, len(string(rn)), errors.New(`nil tty`)
	}
	rb := make([]byte, 4) // 4 is the max bytes for UTF-8 character code points
	l := min(len(r.buf), cap(rb))
	copy(rb, r.buf[:l])
	i := 0
	for ; i < cap(rb); i++ {
		if i >= len(r.buf) {
			b := make([]byte, 1)
			_, err := r.reader.Read(b)
			if err != nil {
				if i > 0 {
					r.buf = rb[1:i]
				}
				return rn, i + 1, errors.New(err)
			}
			rb[i] = b[0]
		}
		if utf8.Valid(rb[:i+1]) {
			rb = rb[:i+1]
			r.buf = nil
			break
		} else if i == cap(rb)-1 {
			r.buf = rb[1:]
			return rn, i + 1, errors.New(err)
		}
	}
	rn, _ = utf8.DecodeRune(rb)
	if rn == utf8.RuneError {
		rn = utf8.RuneError
		return rn, i + 1, errors.New(err)
	}
	return rn, i + 1, nil
}
