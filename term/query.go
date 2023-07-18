package term

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	errorsGo "github.com/go-errors/errors"

	"github.com/srlehn/termimg/internal"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/propkeys"
)

// Querier sends escapes sequences to the terminal and returns the answer.
type Querier interface {
	Query(string, TTY, Parser) (string, error)
}

////////////////////////////////////////////////////////////////////////////////

// CachedQuerier ...
type CachedQuerier interface {
	CachedQuery(string, TTY, Parser, environ.Proprietor) (string, error)
}

func CachedQuery(qu Querier, qs string, tty TTY, p Parser, prIn, prOut environ.Proprietor) (string, error) {
	// TODO bug same value for different parsers
	if prIn == nil {
		return ``, errorsGo.New(internal.ErrNilParam)
	}
	if prOut == nil {
		prOut = environ.NewProprietor()
	}
	qsEnc := base64.StdEncoding.EncodeToString([]byte(qs))
	propKey := propkeys.QueryCachePrefix + qsEnc
	repl, ok := prIn.Property(propKey)
	if ok {
		prOut.SetProperty(propKey, repl)
		return repl, nil
	}
	repl, err := qu.Query(qs, tty, p)
	if err != nil {
		return ``, err
	}
	prOut.SetProperty(propKey, repl)
	return repl, nil
}

type queryCacher struct{ Querier }

func NewCachedQuerier(qu Querier) CachedQuerier { return &queryCacher{Querier: qu} }

func (q *queryCacher) CachedQuery(qs string, tty TTY, p Parser, pr environ.Proprietor) (string, error) {
	if q == nil {
		return ``, errorsGo.New(internal.ErrNilReceiver)
	}
	if q.Querier == nil {
		return ``, errorsGo.New(`nil querier`)
	}
	return CachedQuery(q, qs, tty, p, pr, pr)
}

////////////////////////////////////////////////////////////////////////////////

// Parser.Parse returns true when end of terminal reply is reached.
type Parser interface {
	Parse(rune) bool
}

// ParserFunc returns true when end of terminal reply is reached.
type ParserFunc func(rune) bool

// Parse returns true when end of terminal reply is reached.
func (f ParserFunc) Parse(r rune) bool {
	return f(r)
}

////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////

var _ Querier = (*querierDummy)(nil)

type querierDummy struct{}

func (q *querierDummy) Query(string, TTY, Parser) (string, error) {
	return ``, errorsGo.New(internal.ErrPlatformNotSupported)
}

var _ TTY = (*ttyDummy)(nil)

type ttyDummy struct {
	filename string
}

func (t *ttyDummy) Write(p []byte) (n int, err error) {
	return 0, errorsGo.New(internal.ErrPlatformNotSupported)
}
func (t *ttyDummy) ReadRune() (r rune, size int, err error) {
	return 0, 0, errorsGo.New(internal.ErrPlatformNotSupported)
}
func (t *ttyDummy) Close() error { return errorsGo.New(internal.ErrPlatformNotSupported) }
func (t *ttyDummy) TTYDevName() string {
	if t == nil {
		return ``
	}
	return t.filename
}

////////////////////////////////////////////////////////////////////////////////

const (
	QueryStringDA1 = "\033[0c"  // https://terminalguide.namepad.de/seq/csi_sc/
	QueryStringDA2 = "\033[>0c" // https://terminalguide.namepad.de/seq/csi_sc__q/
	QueryStringDA3 = "\033[=0c" // https://terminalguide.namepad.de/seq/csi_sc__r/
)

func QueryDeviceAttributes(qu Querier, tty TTY, prIn, prOut environ.Proprietor) error {
	// TODO add mux.Wrap()
	if qu == nil || tty == nil || prIn == nil {
		return errorsGo.New(internal.ErrNilParam)
	}

	// only run once
	if deviceAttributesWereQueried, ok := prIn.Property(propkeys.DeviceAttributesWereQueried); ok && deviceAttributesWereQueried == `true` {
		return nil
	}
	prOut.SetProperty(propkeys.DeviceAttributesWereQueried, `true`)

	var cqu CachedQuerier
	if cq, okCQu := qu.(CachedQuerier); okCQu {
		cqu = cq
	} else {
		cqu = NewCachedQuerier(qu)
	}

	_, avoidDA1 := prIn.Property(propkeys.AvoidDA1)
	_, avoidDA2 := prIn.Property(propkeys.AvoidDA2)
	_, avoidDA3 := prIn.Property(propkeys.AvoidDA3)

	var stopOnBackSlash ParserFunc = func(r rune) bool { return r == '\\' }
	var errs []error
	for _, da := range []struct {
		qs string
		b  bool
		p  Parser
	}{
		{QueryStringDA1, avoidDA1, StopOnAlpha},
		{QueryStringDA2, avoidDA2, StopOnAlpha},
		{QueryStringDA3, avoidDA3, stopOnBackSlash},
	} {
		if da.b {
			continue
		}
		repl, err := cqu.CachedQuery(da.qs, tty, da.p, prIn)
		if err != nil {
			errs = append(errs, err)
		}
		switch da.qs {
		case QueryStringDA1:
			// DA1 - Primary Device Attributes
			// https://vt100.net/docs/vt510-rm/DA1.html
			var found bool
			// var replDec []byte
			// remove CSI+"?" (control sequence introducer)
			repl, found = strings.CutPrefix(repl, "\033[?")
			if !found {
				goto skipExtraction
			}
			// remove ST (string terminator)
			repl, found = strings.CutSuffix(repl, "c")
			if !found {
				goto skipExtraction
			}
			attrs := strings.Split(repl, `;`)
			prOut.SetProperty(propkeys.DeviceClass, attrs[0])
			if len(attrs) == 1 {
				goto skipExtraction
			}
			attrs = attrs[1:]
			prOut.SetProperty(propkeys.DeviceAttributes, strings.Join(attrs, `;`))
			for _, attr := range attrs {
				switch attr {
				case `3`: // https://terminalguide.namepad.de/seq/csi_sc/
					prOut.SetProperty(propkeys.ReGISCapable, `true`)
				case `4`:
					prOut.SetProperty(propkeys.SixelCapable, `true`)
				case `18`:
					prOut.SetProperty(propkeys.WindowingCapable, `true`)
				}
			}
		case QueryStringDA2: // Version
		case QueryStringDA3:
			// DA3 - Tertiary Device Attributes
			// https://vt100.net/docs/vt510-rm/DA3.html
			// DECRPTUI - Report Terminal Unit ID
			// https://www.vt100.net/docs/vt510-rm/DECRPTUI.html
			var found bool
			var replDec []byte
			// remove DCS+"!|" (device control string)
			repl, found = strings.CutPrefix(repl, "\033P!|")
			if !found {
				goto skipExtraction
			}
			// remove ST (string terminator)
			repl, found = strings.CutSuffix(repl, "\033\\")
			if !found {
				goto skipExtraction
			}
			replDec, err = hex.DecodeString(repl)
			if err != nil {
				goto skipExtraction
			}
			repl = string(replDec)
			prOut.SetProperty(propkeys.DA3ID, repl)
		}
	skipExtraction:
		if err != nil {
			errs = append(errs, err)
		}
	}
	err := errors.Join(errs...)
	if err != nil {
		return errorsGo.New(err)
	}
	return nil
}

var (
	xtGetTCapSpecialStrs = []string{`TN`, `Co`, `RGB`}
)

func XTGetTCap(tcap string, qu Querier, tty TTY, prIn, prOut environ.Proprietor) (string, error) {
	// TODO multiple tcaps
	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html /XTGETTCAP
	// https://github.com/dankamongmen/notcurses/blob/master/TERMINALS.md#queries
	tcapHex := strings.ToUpper(hex.EncodeToString([]byte(tcap)))

	_, invalid := prIn.Property(propkeys.XTGETTCAPInvalidPrefix + tcapHex)
	if invalid {
		return ``, errorsGo.New(internal.ErrXTGetTCapInvalidRequest)
	}
	repl, exists := prIn.Property(propkeys.XTGETTCAPKeyNamePrefix + tcapHex)
	if exists {
		return repl, nil
	}

	// add DA1 so that we don't have to wait for a timeout
	qsXTGETTCAP := "\033P+q" + tcapHex + "\033\\" + QueryStringDA1

	var encounteredST bool
	var prs ParserFunc = func(r rune) bool {
		if r == '[' {
			// CSI not DSC, so no reply to XTGETTCAP query
			encounteredST = true
			return false
		}
		if encounteredST {
			if StopOnAlpha(r) {
				return true
			} else {
				return false
			}
		}
		if r == '\\' || r == '\a' {
			encounteredST = true
		}
		return false
	}

	if prOut == nil {
		prOut = environ.NewProprietor()
	}
	setProps := func(pr environ.Proprietor, repl string) {
		prOut.SetProperty(propkeys.XTGETTCAPKeyNamePrefix+tcapHex, repl)
		switch tcap {
		case `TN`, `Co`, `RGB`:
			prOut.SetProperty(propkeys.XTGETTCAPSpecialPrefix+tcap, repl)
		}
	}
	repl, err := CachedQuery(qu, qsXTGETTCAP, tty, prs, prIn, prOut)
	if err != nil {
		if err.Error() == internal.ErrTimeoutInterval.Error() {
			setProps(prOut, ``)
			return ``, nil
		} else {
			return ``, err
		}
	}
	replParts := strings.SplitN(repl, `[`, 2) // split at CSI
	if len(replParts) < 2 {
		return ``, errors.New(`found no CSI in reply`)
	}
	errStrUknownFormat := `unknown reply format`
	repl = strings.TrimSuffix(replParts[0], "\033") // separate, because of xterm
	if len(repl) == 0 {
		goto end // found only reply to DA1 not to XTGETTCAP
	}
	{
		var found bool
		repl, found = strings.CutSuffix(repl, "\033\\") // strip ST
		if !found {
			return ``, errors.New(errStrUknownFormat)
		}
		// strip DCS
		repl = strings.TrimPrefix(repl, "\033")    // TODO might be missing, query issue?
		repl, found = strings.CutPrefix(repl, "P") // strip DCS
		if !found || len(repl) == 0 {
			return ``, errors.New(errStrUknownFormat)
		}
		switch repl[0] {
		case '0':
			setProps(prOut, ``)
			prOut.SetProperty(propkeys.XTGETTCAPInvalidPrefix+tcapHex, `true`)
			return ``, errorsGo.New(internal.ErrXTGetTCapInvalidRequest)
		case '1': //valid
		default:
			return ``, errors.New(errStrUknownFormat)
		}
		repl = strings.TrimPrefix(repl, `1+r`+tcapHex+`=`)
		replDec, err := hex.DecodeString(repl)
		if err != nil {
			return ``, errorsGo.New(err)
		}
		repl = string(replDec)
	}

end:
	setProps(prOut, repl)
	return repl, nil
}

////////////////////////////////////////////////////////////////////////////////

// StopOnAlpha ...
var StopOnAlpha ParserFunc = func(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
