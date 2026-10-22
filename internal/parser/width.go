package parser

import (
	"strconv"
	"strings"
)

// Japanese input methods type digits, the full stop and the space in their full-width forms (１２．５,
// ．, U+3000), and a folder named in one of them is the same chapter as the one named in ASCII
// (docs/adr/0020-full-width-digits-and-space-in-chapter-names.md). The parsers accept those forms where
// they accept the ASCII ones, in the parts of a name that say a number or separate words; the text of a
// title, a group or a language is returned as it was written.

// widener widens the two escapes a pattern is written with: \d is a digit of either width and \s a space
// of either width. A pattern written with them is read as before for a name in ASCII.
var widener = strings.NewReplacer(`\d`, `[0-9０-９]`, `\s`, `[\s　]`)

// widen returns pattern, written with \d and \s, as one that also accepts the full-width digits and the
// ideographic space. It does not touch a class that holds \d or \s (none of the patterns has one).
func widen(pattern string) string {
	return widener.Replace(pattern)
}

// foldWidth returns s with its full-width digits and full stops as ASCII ones, which is the form
// strconv reads. Everything else is left as it is.
func foldWidth(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '０' && r <= '９':
			return '0' + (r - '０')
		case r == '．':
			return '.'
		}
		return r
	}, s)
}

// parseNumber reads a chapter or volume number the patterns matched, in either width.
func parseNumber(s string) (float64, error) {
	return strconv.ParseFloat(foldWidth(s), 64)
}
