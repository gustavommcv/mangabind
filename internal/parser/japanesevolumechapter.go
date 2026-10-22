package parser

import "regexp"

// Matches the Japanese counter of a volume, `巻`, after its number and before whatever follows it, which is
// read as a chapter: "第1巻 第2話", "1巻　#3", "第01巻 - 第12話 始まり". The volume may have a decimal part, as
// in the other parsers. A range of volumes ("第1-3巻") has no `巻` right after its first number, so it is not
// matched.
var japaneseVolumeRe = regexp.MustCompile(widen(
	`^(?:第\s*)?` + chapterNumber + `\s*巻(?:\s*[-:：]\s*|\s*)(.+?)\s*$`,
))

// JapaneseVolumeChapterParser recognizes a chapter whose name begins with its volume in the Japanese way,
// `第N巻` or `N巻`, followed by a chapter as JapaneseChapterParser reads it (see
// docs/adr/0022-japanese-volume-markers.md). A volume with nothing after it is not a chapter and is not
// read.
type JapaneseVolumeChapterParser struct{}

func (JapaneseVolumeChapterParser) Name() string { return "japanese-volume-chapter" }

func (JapaneseVolumeChapterParser) Parse(dirName string) (ParsedChapter, bool) {
	m := japaneseVolumeRe.FindStringSubmatch(dirName)
	if m == nil {
		return ParsedChapter{}, false
	}
	vol, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}
	parsed, ok := parseJapaneseChapter(m[2])
	if !ok {
		return ParsedChapter{}, false
	}
	parsed.Volume = &vol
	return parsed, true
}
