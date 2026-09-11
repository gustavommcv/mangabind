package parser

import (
	"regexp"
	"strconv"
)

// Matches chapter-only folder names with no volume information, e.g.
// "Chapter 5", "Ch. 5.5", "c005", optionally followed by "- Title" or
// ": Title".
var chapterOnlyRe = regexp.MustCompile(
	`(?i)^(?:chapter|ch\.?|c)\s*0*(\d+(?:\.\d+)?)([xyz]\d+)?(?:\s*[-:]\s*(.+))?$`,
)

// ChapterOnlyParser recognizes conventions that carry a chapter number but
// no volume number.
type ChapterOnlyParser struct{}

func (ChapterOnlyParser) Name() string { return "chapter-only" }

func (ChapterOnlyParser) Parse(dirName string) (ParsedChapter, bool) {
	m := chapterOnlyRe.FindStringSubmatch(dirName)
	if m == nil {
		return ParsedChapter{}, false
	}

	ch, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return ParsedChapter{}, false
	}

	return ParsedChapter{
		Chapter: ch,
		Special: m[2],
		Title:   m[3],
	}, true
}
