package parser

import "regexp"

// A language tag as a download tool writes it between brackets at the end of a name: two or three letters
// and, after a hyphen, a region or a script (`en`, `pt-br`, `zh-hk`). It is not any word in brackets, so that
// the end of a title (`(Part 2)`, `(Final)`, `(+2 short Extras)`) stays in the title.
const languageTagPattern = `[a-z]{2,3}(?:-[a-z0-9]{2,4})*`

// Matches chapter-only folder names with no volume information, e.g.
// "Chapter 5", "Ch. 5.5", "c005", optionally followed by "- Title" or
// ": Title", and, as the MangaDex downloaders write it, a language and a
// group at the end: "Ch.0001 (en)", "Ch.0011 - Title (en) [Group]".
var chapterOnlyRe = regexp.MustCompile(widen(
	`(?i)^(?:chapter|ch\.?|c)\s*[0０]*(\d+(?:[.．]\d+)?)([xyz]\d+)?(?:\s*[-:]\s*(.+?))?(?:\s*\((` + languageTagPattern + `)\))?(?:\s*\[(.+?)\])?\s*$`,
))

// ChapterOnlyParser recognizes conventions that carry a chapter number but
// no volume number.
type ChapterOnlyParser struct{}

func (ChapterOnlyParser) Name() string { return "chapter-only" }

func (ChapterOnlyParser) Parse(dirName string) (ParsedChapter, bool) {
	m := chapterOnlyRe.FindStringSubmatch(dirName)
	if m == nil {
		return ParsedChapter{}, false
	}

	ch, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}

	return ParsedChapter{
		Chapter: ch,
		Special: foldWidth(m[2]),
		Title:   m[3],
		Lang:    m[4],
		Group:   m[5],
	}, true
}
