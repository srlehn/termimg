package keyfile

import (
	"bytes"
)

// String returns the value for group 'g' and key 'k' as a string.
// String will return a blank string if the key doesn't exist; use
// GroupExists or KeyExists to if you need to treat a missing value
// differently then a blank value.
func (kf *KeyFile) String(g, k string) (string, error) {
	return unescapeString(kf.Value(g, k))
}

// StringList returns a slice of strings for group 'g' and key 'k'.
// StringList will return an empty slice if the key doesn't exist; use
// GroupExists or KeyExists to if you need to treat a missing value
// differently then a blank value.
func (kf *KeyFile) StringList(g, k string) ([]string, error) {
	vlist, err := kf.ValueList(g, k)
	if err != nil {
		return nil, err
	}

	slist := make([]string, len(vlist), len(vlist))
	for i, val := range vlist {
		slist[i], err = unescapeString(val)
		if err != nil {
			return nil, err
		}
	}

	return slist, nil
}

func unescapeString(s string) (string, error) {
	var buf bytes.Buffer
	var isEscaped bool
	var err error

	for _, r := range s {
		if isEscaped {
			switch r {
			case 's':
				_, err = buf.WriteRune(' ')
			case 'n':
				_, err = buf.WriteRune('\n')
			case 't':
				_, err = buf.WriteRune('\t')
			case 'r':
				_, err = buf.WriteRune('\r')
			case '\\':
				_, err = buf.WriteRune('\\')
			default:
				err = ErrBadEscapeSequence
			}

			if err != nil {
				return "", err
			}

			isEscaped = false
		} else {
			if r == '\\' {
				isEscaped = true
			} else {
				buf.WriteRune(r)
			}
		}
	}
	if isEscaped {
		return "", ErrUnexpectedEndOfString
	}
	return buf.String(), nil
}
