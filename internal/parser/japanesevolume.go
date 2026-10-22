package parser

import "regexp"

// Matches a name that is a volume and nothing else, `第N巻` or `N巻`, with the groups in brackets that a
// collection puts after it (a publisher, a year): "第09巻", "2巻 (講談社)", "第01巻 [2020]".
var japaneseVolumeAloneRe = regexp.MustCompile(widen(
	`^(?:第\s*)?` + chapterNumber + `\s*巻(?:\s*(?:\([^)]*\)|\[[^\]]*\]|（[^）]*）|【[^】]*】))*\s*$`,
))

// JapaneseVolumeParser recognizes a folder or a .cbz that is named for a volume and holds that volume's
// pages, as a collection scanned from the printed books has it: `(一般コミック) [奥浩哉] いぬやしき 第09巻`
// (see docs/adr/0024-a-volume-named-folder-is-a-volume-of-one-chapter.md). Mangabind binds chapters into
// volumes, so such a unit is a volume with one chapter, numbered 1 and without a title. It is read after a
// prefix as the other Japanese markers are. A range of volumes (`第1-3巻`) is not one volume and is not read.
type JapaneseVolumeParser struct{}

func (JapaneseVolumeParser) Name() string { return "japanese-volume" }

func (JapaneseVolumeParser) Parse(dirName string) (ParsedChapter, bool) {
	return afterPrefix(dirName, parseJapaneseVolumeAlone)
}

func parseJapaneseVolumeAlone(s string) (ParsedChapter, bool) {
	m := japaneseVolumeAloneRe.FindStringSubmatch(s)
	if m == nil {
		return ParsedChapter{}, false
	}
	vol, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}
	return ParsedChapter{Volume: &vol, Chapter: 1}, true
}
