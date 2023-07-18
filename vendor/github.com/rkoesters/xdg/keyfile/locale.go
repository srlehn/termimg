package keyfile

import (
	"bytes"
	"errors"
	"os"
)

// Locale represents a locale for use in parsing translatable strings.
type Locale struct {
	lang     string
	country  string
	encoding string
	modifier string
}

var defaultLocale *Locale

// DefaultLocale returns the locale specified by the environment.
func DefaultLocale() Locale {
	if defaultLocale == nil {
		val := os.Getenv("LANGUAGE")
		if val == "" {
			val = os.Getenv("LC_ALL")
			if val == "" {
				val = os.Getenv("LC_MESSAGES")
				if val == "" {
					val = os.Getenv("LANG")
				}
			}
		}

		l, err := ParseLocale(val)
		if err == nil {
			defaultLocale = &l
		} else {
			defaultLocale = &Locale{}
		}
	}
	return *defaultLocale
}

// ErrBadLocaleFormat is returned by ParseLocale when the given string
// is not formatted properly (for example, missing the lang component).
var ErrBadLocaleFormat = errors.New("bad locale format")

// ParseLocale parses a locale in the format
//
// 	lang_COUNTRY.ENCODING@MODIFIER
//
// where "_COUNTRY", ".ENCODING", and "@MODIFIER" can be omitted. A
// blank string, "C", and "POSIX" are special cases that evaluate to a
// blank Locale.
func ParseLocale(s string) (Locale, error) {
	// A blank string, "C", and "POSIX" are valid locales, they
	// evaluate to a blank Locale.
	if s == "" || s == "C" || s == "POSIX" {
		return Locale{}, nil
	}

	var buf bytes.Buffer
	var l Locale

	i := 0

	// lang
	for i < len(s) && s[i] != '_' && s[i] != '.' && s[i] != '@' {
		buf.WriteByte(s[i])
		i++
	}
	l.lang = buf.String()
	buf.Reset()

	// lang is required.
	if l.lang == "" {
		return Locale{}, ErrBadLocaleFormat
	}

	// COUNTRY
	if i < len(s) && s[i] == '_' {
		i++
		for i < len(s) && s[i] != '.' && s[i] != '@' {
			buf.WriteByte(s[i])
			i++
		}
		l.country = buf.String()
		buf.Reset()
	}

	// ENCODING
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] != '@' {
			buf.WriteByte(s[i])
			i++
		}
		l.encoding = buf.String()
		buf.Reset()
	}

	// MODIFIER
	if i < len(s) && s[i] == '@' {
		i++
		for i < len(s) {
			buf.WriteByte(s[i])
			i++
		}
		l.modifier = buf.String()
	}

	return l, nil
}

// String returns the given locale as a formatted string. The returned
// string is in the same format expected by ParseLocale.
func (l Locale) String() string {
	var buf bytes.Buffer

	buf.WriteString(l.lang)

	if l.country != "" {
		buf.WriteRune('_')
		buf.WriteString(l.country)
	}

	if l.encoding != "" {
		buf.WriteRune('.')
		buf.WriteString(l.encoding)
	}

	if l.modifier != "" {
		buf.WriteRune('@')
		buf.WriteString(l.modifier)
	}

	return buf.String()
}

// Variants returns a sorted slice of Locales that should be checked for
// when resolving a localestring.
func (l Locale) Variants() []Locale {
	variants := make([]Locale, 0, 4)

	hasLang := l.lang != ""
	hasCountry := l.country != ""
	hasModifier := l.modifier != ""

	if hasLang && hasCountry && hasModifier {
		variants = append(variants, Locale{
			lang:     l.lang,
			country:  l.country,
			modifier: l.modifier,
		})
	}

	if hasLang && hasCountry {
		variants = append(variants, Locale{
			lang:    l.lang,
			country: l.country,
		})
	}

	if hasLang && hasModifier {
		variants = append(variants, Locale{
			lang:     l.lang,
			modifier: l.modifier,
		})
	}

	if hasLang {
		variants = append(variants, Locale{
			lang: l.lang,
		})
	}

	return variants
}
