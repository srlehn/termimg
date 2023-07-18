// Package keyfile implements the ini file format that is used in many
// of the xdg specs.
//
// WARNING: This package is meant for internal use and the API may
// change without warning.
package keyfile

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// KeyFile is an implementation of the keyfile format used by several
// FreeDesktop.org (xdg) specs. The key values without a header section
// can be accessed using an empty string as the group.
type KeyFile struct {
	m map[string]map[string]string
}

// New creates a new KeyFile and returns it.
func New(r io.Reader) (*KeyFile, error) {
	kf := new(KeyFile)
	kf.m = make(map[string]map[string]string)
	hdr := ""
	kf.m[hdr] = make(map[string]string)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case len(line) == 0:
			// Empty line.
		case line[0] == '#':
			// Comment.
		case line[0] == '[' && line[len(line)-1] == ']':
			// Group header.
			hdr = line[1 : len(line)-1]
			kf.m[hdr] = make(map[string]string)
		case strings.Contains(line, "="):
			// Entry.
			p := strings.SplitN(line, "=", 2)
			p[0] = strings.TrimSpace(p[0])
			p[1] = strings.TrimSpace(p[1])
			kf.m[hdr][p[0]] = p[1]
		default:
			return nil, ErrInvalid
		}
	}
	return kf, nil
}

// Groups returns a slice of groups that exist for the KeyFile.
func (kf *KeyFile) Groups() []string {
	groups := make([]string, 0, len(kf.m))
	for k := range kf.m {
		groups = append(groups, k)
	}
	return groups
}

// GroupExists returns a bool indicating whether the given group 'g'
// exists.
func (kf *KeyFile) GroupExists(g string) bool {
	_, exists := kf.m[g]
	return exists
}

// Keys returns a slice of keys that exist for the given group 'g'.
func (kf *KeyFile) Keys(g string) []string {
	keys := make([]string, 0, len(kf.m[g]))
	for k := range kf.m[g] {
		keys = append(keys, k)
	}
	return keys
}

// KeyExists returns a bool indicating whether the given group 'g' and
// key 'k' exists.
func (kf *KeyFile) KeyExists(g, k string) bool {
	_, exists := kf.m[g][k]
	return exists
}

// Value returns the raw string for group 'g' and key 'k'. Value will
// return a blank string if the key doesn't exist; use GroupExists or
// KeyExists to if you need to treat a missing value differently then a
// blank value.
func (kf *KeyFile) Value(g, k string) string {
	return kf.m[g][k]
}

// ValueList returns a slice of raw strings for group 'g' and key 'k'.
// ValueList will return an empty slice if the key doesn't exist; use
// GroupExists or KeyExists to if you need to treat a missing value
// differently then a blank value.
func (kf *KeyFile) ValueList(g, k string) ([]string, error) {
	var buf bytes.Buffer
	var isEscaped bool
	var list []string

	for _, r := range kf.Value(g, k) {
		if isEscaped {
			if r == ';' {
				buf.WriteRune(';')
			} else {
				// The escape sequence isn't '\;', so we
				// want to copy it over as is.
				buf.WriteRune('\\')
				buf.WriteRune(r)
			}
			isEscaped = false
		} else {
			switch r {
			case '\\':
				isEscaped = true
			case ';':
				list = append(list, buf.String())
				buf.Reset()
			default:
				buf.WriteRune(r)
			}
		}
	}
	if isEscaped {
		return nil, ErrUnexpectedEndOfString
	}

	last := buf.String()
	if last != "" {
		list = append(list, last)
	}

	return list, nil
}
