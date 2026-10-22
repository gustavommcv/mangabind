package parser

import "regexp"

// Matches a chapter folder that's nothing but a number, e.g. "001", "12.5" -
// a minimalist convention some scan groups/repackagers use, with no
// "Chapter"/"Ch" word at all. No volume or title is ever present in a name
// this bare, so this parser never sets them.
var bareNumberRe = regexp.MustCompile(widen(`^[0０]*(\d+(?:[.．]\d+)?)$`))

// BareNumberParser recognizes chapter folders that are just a number, with
// no other text.
type BareNumberParser struct{}

func (BareNumberParser) Name() string { return "bare-number" }

func (BareNumberParser) Parse(dirName string) (ParsedChapter, bool) {
	m := bareNumberRe.FindStringSubmatch(dirName)
	if m == nil {
		return ParsedChapter{}, false
	}

	ch, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}

	return ParsedChapter{Chapter: ch}, true
}
