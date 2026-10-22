package parser

import (
	"regexp"
	"strings"
	"unicode"
)

// The characters that close what is written before a marker: a bracket around an author or a group, in
// either width or in the corner forms Japanese uses.
const closingBrackets = "]】)）」』］〕》"

// Matches a volume or a chapter written in a prefix, in the English way (`Vol.01`, `Volume 1`, `v01`,
// `Ch.5`, `Chapter 5`) or the Japanese (`第1巻`, `2巻`, `第1-3巻`, `第5話`).
var markerInPrefixRe = regexp.MustCompile(widen(`(?i)\b(?:vol(?:ume)?\.?|v|ch(?:apter)?\.?)\s*\d|\d\s*[巻話]`))

// afterPrefix reads a name that may begin with an author or a title before the marker that parse looks
// for: `[奥浩哉] いぬやしき 第09巻`, `[ハレム]ナナとカオル　第10話`, `Raw_第1話 始まり`. parse is tried on the
// whole name and then on each part of it that begins where a prefix may end: after a space of either width,
// a closing bracket or an underscore (what Mihon puts between the scanlator and the name), and before any
// `第`, since that character begins a marker and nothing else. The first part that parses is the answer, so
// the leftmost marker is read and what follows it is the title.
//
// A marker stuck to the letter before it (`タイトル12話`) is not looked for: a number in a title is not a
// chapter. Nor is one after a prefix that holds a volume or a chapter (`Vol.01 第1話`, `第1-3巻 第2話`,
// `第1-3話 第5話`): what the prefix says would be dropped without a word, where a name that is not read is at
// least reported.
func afterPrefix(name string, parse func(string) (ParsedChapter, bool)) (ParsedChapter, bool) {
	var prev rune
	for i, r := range name {
		if i == 0 || r == '第' || endsPrefix(prev) {
			if i > 0 && markerInPrefixRe.MatchString(name[:i]) {
				return ParsedChapter{}, false
			}
			if parsed, ok := parse(name[i:]); ok {
				return parsed, true
			}
		}
		prev = r
	}
	return ParsedChapter{}, false
}

func endsPrefix(r rune) bool {
	return unicode.IsSpace(r) || r == '_' || strings.ContainsRune(closingBrackets, r)
}
