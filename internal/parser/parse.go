package parser

var StopOnAlpha ParserFunc = func(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

var StopOnBackSlash ParserFunc = func(r rune) bool { return r == '\\' }

var StopOnR ParserFunc = func(r rune) bool { return r == 'R' }

var StopOnC ParserFunc = func(r rune) bool { return r == 'c' }

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

func DA1Wrap(p Parser) Parser {
	var encounteredCSI bool
	var lastChar rune
	return ParserFunc(func(r rune) bool {
		var ret bool
		if encounteredCSI && StopOnAlpha(r) {
			ret = true
			goto end
		}
		if r == '[' && lastChar == '\033' {
			encounteredCSI = true
			goto end
		}
		ret = p.Parse(r)
	end:
		lastChar = r
		return ret
	})
}

func NParser(p Parser, n uint) Parser {
	var seqCnt uint
	return ParserFunc(func(r rune) bool {
		if p == nil {
			return true // don't block
		}
		if p.Parse(r) {
			seqCnt++
			if seqCnt == n {
				return true
			}
		}
		return false
	})
}

type parser struct {
	last        rune
	state       string
	terminology bool
	stopOnCSI   bool
}

func NewParser(terminology bool, stopOnCSI bool) Parser {
	return &parser{
		terminology: terminology,
		stopOnCSI:   stopOnCSI,
	}
}

func (p *parser) Parse(r rune) bool {
	if p == nil {
		return true // don't hang
	}
	const (
		nF          = `nF`
		csi         = `CSI`
		stTerm      = `ST-terminated`
		terminology = `Terminology`
		_Fp         = `Fp`
		_Fe         = `Fe`
		_Fs         = `Fs`
	)
	var ret bool
	var seqType string
	switch p.state {
	case nF: // [0x20—0x2F]+[0x30-0x7E]
		switch {
		case r >= 0x20 && r <= 0x2F:
		case r >= 0x30 && r <= 0x7E && (p.last >= 0x20 && p.last <= 0x2F):
			p.state = ``
			seqType = nF
			ret = true
		default:
			// fail
			p.state = ``
			ret = true
		}
	case csi: // [0x40-0x7E]
		if r >= 0x40 && r <= 0x7E {
			p.state = ``
			seqType = csi
			ret = true
		}
	case stTerm:
		if r == '\a' || (r == '\\' && p.last == '\033') {
			p.state = ``
			seqType = stTerm
			ret = true
		}
	case terminology:
		if r == '\x00' {
			p.state = ``
			seqType = terminology
			ret = true
		}
	case ``:
		if p.last == '\033' {
			switch {
			// nF escape sequences [0x20—0x2F]+[0x30-0x7E]
			case r >= 0x20 && r <= 0x2F:
				p.state = nF

			// Fp escape sequences [0x30—0x3F]
			case r >= 0x30 && r <= 0x3F:
				seqType = _Fp
				ret = true

			// Fe escape sequences [0x40-0x5F]
			// CSI - terminated by [0x40-0x7E]
			case r == '[':
				p.state = csi
			// DCS, OSC, SOS, PM, APC - ST terminated
			case r == 'P' || r == ']' || r == 'X' || r == '^' || r == '_':
				p.state = stTerm
			case r >= 0x40 && r <= 0x5F:
				seqType = _Fe
				ret = true

			// Terminology  \x00 terminated - conflics with Fs escape sequences
			// https://github.com/borisfaure/terminology/tree/master#extended-escapes-for-terminology-only
			case p.terminology && r == '}':
				p.state = terminology
			// Fs escape sequences [0x60—0x7E]
			case r >= 0x60 && r <= 0x7E:
				seqType = _Fs
				ret = true
			}
		}
	}
	p.last = r
	if p.stopOnCSI {
		return ret && seqType == csi
	}
	return ret
}

// for queries.ITerm2PropVersion+queries.DA1 queries
type ITerm2DA1Parser struct {
	lastRune                     rune
	insideITerm2Reply, insideCSI bool
}

func (p *ITerm2DA1Parser) Parse(r rune) bool {
	var ret bool
	if r == '\033' {
		goto end
	}
	if r == '[' && p.lastRune == '\033' {
		p.insideCSI = true
		goto end
	}
	if p.insideCSI && p.lastRune == '[' && r != '?' {
		// not a DA1 reply
		p.insideCSI = false
		p.insideITerm2Reply = true
	}
	if p.insideITerm2Reply && r == 'n' {
		p.insideITerm2Reply = false
	}
	if p.insideCSI && r == 'c' {
		p.insideCSI = false
		ret = true
	}
end:
	p.lastRune = r
	return ret
}
