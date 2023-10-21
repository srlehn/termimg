package term

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/srlehn/termimg/internal/consts"
	"github.com/srlehn/termimg/internal/environ"
	"github.com/srlehn/termimg/internal/errors"
	"github.com/srlehn/termimg/internal/parser"
	"github.com/srlehn/termimg/internal/propkeys"
	"github.com/srlehn/termimg/internal/queries"
)

// Querier sends escapes sequences to the terminal and returns the answer.
type Querier interface {
	Query(string, TTY, Parser) (string, error)
}

////////////////////////////////////////////////////////////////////////////////

// CachedQuerier ...
type CachedQuerier interface {
	CachedQuery(string, TTY, Parser, environ.Properties) (string, error)
}

func CachedQuery(qu Querier, qs string, tty TTY, p Parser, prIn, prOut environ.Properties) (string, error) {
	// TODO bug same value for different parsers
	if prIn == nil {
		return ``, errors.New(consts.ErrNilParam)
	}
	if prOut == nil {
		prOut = environ.NewProperties()
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

func (q *queryCacher) CachedQuery(qs string, tty TTY, p Parser, pr environ.Properties) (string, error) {
	if q == nil {
		return ``, errors.New(consts.ErrNilReceiver)
	}
	if q.Querier == nil {
		return ``, errors.New(`nil querier`)
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
	return ``, errors.New(consts.ErrPlatformNotSupported)
}

var _ TTY = (*ttyDummy)(nil)

type ttyDummy struct {
	filename string
}

func (t *ttyDummy) Write(p []byte) (n int, err error) {
	return 0, errors.New(consts.ErrPlatformNotSupported)
}
func (t *ttyDummy) ReadRune() (r rune, size int, err error) {
	return 0, 0, errors.New(consts.ErrPlatformNotSupported)
}
func (t *ttyDummy) Close() error { return errors.New(consts.ErrPlatformNotSupported) }
func (t *ttyDummy) TTYDevName() string {
	if t == nil {
		return ``
	}
	return t.filename
}

////////////////////////////////////////////////////////////////////////////////

// QueryDeviceAttributes should only be used for external TermCheckers
func QueryDeviceAttributes(qu Querier, tty TTY, prIn, prOut environ.Properties) error {
	// TODO add mux.Wrap()
	if qu == nil || tty == nil || prIn == nil {
		return errors.New(consts.ErrNilParam)
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

	var errs []error
	for _, da := range []struct {
		qs string
		b  bool
		p  Parser
	}{
		{queries.DA1, avoidDA1, parser.StopOnAlpha},
		{queries.DA2, avoidDA2, parser.StopOnAlpha},
		{queries.DA3, avoidDA3, parser.StopOnBackSlash},
	} {
		if da.b {
			continue
		}
		repl, err := cqu.CachedQuery(da.qs, tty, da.p, prIn)
		if err != nil {
			errs = append(errs, err)
		}
		switch da.qs {
		case queries.DA1:
			// DA1 - Primary Device Attributes
			// https://vt100.net/docs/vt510-rm/DA1.html
			// remove CSI+"?" (control sequence introducer)
			repl, found := strings.CutPrefix(repl, queries.CSI+`?`)
			if !found {
				goto skipExtraction
			}
			// remove CSI terminator
			repl, found = strings.CutSuffix(repl, `c`)
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
		case queries.DA2: // Model & Version
			// https://vt100.net/docs/vt510-rm/DA2.html
			// https://terminalguide.namepad.de/seq/csi_sc__q/
			// remove CSI+">" (control sequence introducer)
			repl, found := strings.CutPrefix(repl, queries.CSI+`>`)
			if !found {
				goto skipExtraction
			}
			// remove CSI terminator
			repl, found = strings.CutSuffix(repl, `c`)
			if !found {
				goto skipExtraction
			}
			attrs := strings.Split(repl, `;`)
			if len(attrs) != 3 {
				goto skipExtraction
			}
			prOut.SetProperty(propkeys.DA2Model, attrs[0])
			prOut.SetProperty(propkeys.DA2Version, attrs[1])
			prOut.SetProperty(propkeys.DA2Keyboard, attrs[2])
			model, err := strconv.Atoi(attrs[0])
			if err == nil && model >= 0x20 && model <= 0x7E {
				// store if printable
				prOut.SetProperty(propkeys.DA2ModelLetter, string(rune(model)))
			}
		case queries.DA3:
			// DA3 - Tertiary Device Attributes
			// https://vt100.net/docs/vt510-rm/DA3.html
			// DECRPTUI - Report Terminal Unit ID
			// https://www.vt100.net/docs/vt510-rm/DECRPTUI.html
			var replDec []byte
			// remove DCS+"!|" (device control string)
			repl, found := strings.CutPrefix(repl, "\033P!|")
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
			prOut.SetProperty(propkeys.DA3IDHex, repl)
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
		return errors.New(err)
	}
	return nil
}

var (
	xtGetTCapSpecialStrs = []string{`TN`, `Co`, `RGB`}
)

func xtGetTCap(tcap string, qu Querier, tty TTY, prIn, prOut environ.Properties) (string, error) {
	// TODO multiple tcaps
	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html /XTGETTCAP
	// https://github.com/dankamongmen/notcurses/blob/master/TERMINALS.md#queries
	// TODO darktile crashes on query
	tcapHex := strings.ToUpper(hex.EncodeToString([]byte(tcap)))

	_, invalid := prIn.Property(propkeys.XTGETTCAPInvalidPrefix + tcapHex)
	if invalid {
		return ``, errors.New(consts.ErrXTGetTCapInvalidRequest)
	}
	repl, exists := prIn.Property(propkeys.XTGETTCAPKeyNamePrefix + tcapHex)
	if exists {
		return repl, nil
	}

	// add DA1 so that we don't have to wait for a timeout
	qsXTGETTCAP := queries.DCS + `+q` + tcapHex + queries.ST + queries.DA1

	var encounteredST bool
	var prs ParserFunc = func(r rune) bool {
		if r == '[' {
			// CSI not DSC, so no reply to XTGETTCAP query
			encounteredST = true
			return false
		}
		if encounteredST {
			if parser.StopOnAlpha(r) {
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
		prOut = environ.NewProperties()
	}
	setProps := func(pr environ.Properties, repl string) {
		prOut.SetProperty(propkeys.XTGETTCAPKeyNamePrefix+tcapHex, repl)
		switch tcap {
		case `TN`, `Co`, `RGB`:
			prOut.SetProperty(propkeys.XTGETTCAPSpecialPrefix+tcap, repl)
		}
	}
	repl, err := CachedQuery(qu, qsXTGETTCAP, tty, prs, prIn, prOut)
	if err != nil {
		if err.Error() == consts.ErrTimeoutInterval.Error() {
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
			return ``, errors.New(consts.ErrXTGetTCapInvalidRequest)
		case '1': //valid
		default:
			return ``, errors.New(errStrUknownFormat)
		}
		repl = strings.TrimPrefix(repl, `1+r`+tcapHex+`=`)
		replDec, err := hex.DecodeString(repl)
		if err != nil {
			return ``, errors.New(err)
		}
		repl = string(replDec)
	}

end:
	setProps(prOut, repl)
	return repl, nil
}

func xtVersion(qu Querier, tty TTY, prIn, prOut environ.Properties) (string, error) {
	xtVer, okXTVer := prIn.Property(propkeys.XTVERSION)
	if !okXTVer {
		repl, err := CachedQuery(qu, queries.XTVERSION+queries.DA1, tty, parser.NewParser(false, true), prIn, prOut)
		if err != nil {
			return ``, err
		}
		replParts := strings.Split(repl, queries.ST)
		if len(replParts) != 2 {
			return ``, errors.New(`no reply to XTVERSION query`)
		}
		xtVerPrefix := queries.DCS + `>|`
		repl, hasXTVerReplPrefix := strings.CutPrefix(replParts[0], xtVerPrefix)
		if !hasXTVerReplPrefix {
			return ``, errors.New(`invalid reply to XTVERSION query`)
		}
		xtVer = repl
	}
	prOut.SetProperty(propkeys.XTVERSION, xtVer)
	return xtVer, nil
}
