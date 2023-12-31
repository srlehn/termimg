// SPDX-License-Identifier: Unlicense OR BSD-3-Clause

package language

import (
	"os"
	"strings"
)

var canonMap = [256]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, '-', 0, 0,
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 0, 0, 0, 0, 0, 0,
	'-', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
	'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 0, 0, 0, 0, '-',
	0, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
	'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 0, 0, 0, 0, 0,
}

// Language store the canonicalized BCP 47 tag,
// which has the generic form <lang>-<country>-<other tags>...
type Language string

// NewLanguage canonicalizes the language input (as a BCP 47 language tag), by converting it to
// lowercase, mapping '_' to '-', and stripping all characters other
// than letters, numbers and '-'.
func NewLanguage(language string) Language {
	out := make([]byte, 0, len(language))
	for _, r := range language {
		if r >= 0xFF {
			continue
		}
		can := canonMap[r]
		if can != 0 {
			out = append(out, can)
		}
	}
	return Language(out)
}

// Primary returns the root language of l, that is
// the part before the first '-' separator
func (l Language) Primary() Language {
	if index := strings.IndexByte(string(l), '-'); index != -1 {
		l = l[:index]
	}
	return l
}

// SimpleInheritance returns the list of matching language, using simple truncation inheritance.
// The resulting slice starts with the given whole language.
// See http://www.unicode.org/reports/tr35/#Locale_Inheritance for more information.
func (l Language) SimpleInheritance() []Language {
	tags := strings.Split(string(l), "-")
	out := make([]Language, 0, len(tags))
	for len(tags) != 0 {
		out = append(out, Language(strings.Join(tags, "-")))
		tags = tags[:len(tags)-1]
	}
	return out
}

// IsDerivedFrom returns `true` if `l` has
// the `root` as primary language.
func (l Language) IsDerivedFrom(root Language) bool { return l.Primary() == root }

// IsUndetermined returns `true` if its primary language is "und".
// It is a shortcut for IsDerivedFrom("und").
func (l Language) IsUndetermined() bool { return l.IsDerivedFrom("und") }

// SplitExtensionTags splits the language at the extension and private-use subtags, which are
// marked by a "-<one char>-" pattern.
// It returns the language before the first pattern, and, if any, the private-use subtag.
//
// (l, "") is returned if the language has no extension or private-use tag.
func (l Language) SplitExtensionTags() (prefix, private Language) {
	if len(l) >= 2 && l[0] == 'x' && l[1] == '-' { // x-<....> 'fully' private
		return "", l
	}

	firstExtension := -1
	for i := 0; i+3 < len(l); i++ {
		if l[i] == '-' && l[i+2] == '-' {
			if firstExtension == -1 { // mark the end of the prefix
				firstExtension = i
			}

			if l[i+1] == 'x' { // private-use tag
				return l[:firstExtension], l[i+1:]
			}
			// else keep looking for private sub tags
		}
	}

	if firstExtension == -1 {
		return l, ""
	}
	return l[:firstExtension], ""
}

// LanguageComparison is a three state enum resulting from comparing two languages
type LanguageComparison uint8

const (
	LanguagesDiffer      LanguageComparison = iota // the two languages are totally differents
	LanguagesExactMatch                            // the two languages are exactly the same
	LanguagePrimaryMatch                           // the two languages have the same primary language, but differs.
)

// Compare compares `other` and `l`.
// Undetermined languages are only compared using the remaining tags,
// meaning that "und-fr" and "und-be" are compared as LanguagesDiffer, not
// LanguagePrimaryMatch.
func (l Language) Compare(other Language) LanguageComparison {
	if l == other {
		return LanguagesExactMatch
	}

	primary1, primary2 := l.Primary(), other.Primary()
	if primary1 != primary2 {
		return LanguagesDiffer
	}

	// check for the undetermined special case
	if primary1 == "und" {
		return LanguagesDiffer
	}
	return LanguagePrimaryMatch
}

func languageFromLocale(locale string) Language {
	if i := strings.IndexByte(locale, '.'); i >= 0 {
		locale = locale[:i]
	}
	return NewLanguage(locale)
}

// DefaultLanguage returns the language found in environment variables LC_ALL, LC_CTYPE or
// LANG (in that order), or the zero value if not found.
func DefaultLanguage() Language {
	p, ok := os.LookupEnv("LC_ALL")
	if ok {
		return languageFromLocale(p)
	}

	p, ok = os.LookupEnv("LC_CTYPE")
	if ok {
		return languageFromLocale(p)
	}

	p, ok = os.LookupEnv("LANG")
	if ok {
		return languageFromLocale(p)
	}

	return ""
}
