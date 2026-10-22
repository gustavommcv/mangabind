package parser

import "regexp"

// The number of a chapter in the way the patterns below read it, in either width.
const chapterNumber = `(\d+(?:[.．]\d+)?)`

// Matches the Japanese counter of a chapter, `話`, after its number: "第1話", "第001話 始まり", "1話",
// "第10.5話 - 幕間". The title after it, if any, may follow a space, a dash or a colon, or nothing.
var japaneseCounterRe = regexp.MustCompile(widen(
	`^(?:第\s*)?` + chapterNumber + `\s*話(?:\s*[-:：]\s*|\s+)?(.*?)\s*$`,
))

// Matches "#12" and "#12 - Title": the chapter number after a hash, followed by nothing or by a space, a
// dash or a colon, so that "#12abc" is not a chapter.
var hashNumberRe = regexp.MustCompile(widen(
	`^#\s*` + chapterNumber + `(?:(?:\s*[-:：]\s*|\s+)(.+?))?\s*$`,
))

// JapaneseChapterParser recognizes the conventions of Japanese manga that carry a chapter number and no
// volume: `第N話`, `N話` and `#N`, each with an optional title (see
// docs/adr/0021-japanese-chapter-markers.md). A range (`第1-3話`) is not read: it would put one folder in
// the place of several chapters. `第N話` and `N話` may come after a prefix, an author or a title
// (docs/adr/0023-japanese-markers-after-a-prefix.md); `#N` has to begin the name, since a hash after a
// title is as often an issue of something else.
type JapaneseChapterParser struct{}

func (JapaneseChapterParser) Name() string { return "japanese-chapter" }

func (JapaneseChapterParser) Parse(dirName string) (ParsedChapter, bool) {
	if parsed, ok := parseJapaneseChapter(dirName); ok {
		return parsed, true
	}
	return afterPrefix(dirName, parseJapaneseCounter)
}

// parseJapaneseChapter reads one of the shapes above from the whole of s. It is shared with
// JapaneseVolumeChapterParser, which reads the same chapter after a volume.
func parseJapaneseChapter(s string) (ParsedChapter, bool) {
	if parsed, ok := parseJapaneseCounter(s); ok {
		return parsed, true
	}
	return parseChapterNumberWith(hashNumberRe, s)
}

// parseJapaneseCounter reads `第N話` and `N話` from the whole of s.
func parseJapaneseCounter(s string) (ParsedChapter, bool) {
	return parseChapterNumberWith(japaneseCounterRe, s)
}

func parseChapterNumberWith(re *regexp.Regexp, s string) (ParsedChapter, bool) {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ParsedChapter{}, false
	}
	ch, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}
	return ParsedChapter{Chapter: ch, Title: m[2]}, true
}
