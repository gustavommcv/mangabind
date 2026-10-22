package parser

import "regexp"

// Matches chapter-only folder names with no volume information, e.g.
// "Chapter 5", "Ch. 5.5", "c005", optionally followed by "- Title" or
// ": Title".
var chapterOnlyRe = regexp.MustCompile(widen(
	`(?i)^(?:chapter|ch\.?|c)\s*[0０]*(\d+(?:[.．]\d+)?)([xyz]\d+)?(?:\s*[-:]\s*(.+))?$`,
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
	}, true
}
