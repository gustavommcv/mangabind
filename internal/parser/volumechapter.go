package parser

import (
	"regexp"
	"strconv"
)

// Matches spelled-out "Volume N Chapter M" / "Vol 1 - Ch. 005" and close
// variants - less strict about dots/dashes than VolChTitleParser, which only
// matches the abbreviated "Vol.NN Ch.NNNN" form. Users have
// reported exactly this shape ("Vol 1 Chapter 5") mixed in with other
// conventions for the same manga (cited in
// docs/adr/0003-pluggable-chapter-parsing.md). This pattern is a superset of
// VolChTitleParser's - registering VolChTitleParser first keeps it the
// authority for its own exact syntax, but the overlap is harmless since both
// extract the same fields from a matching name.
var volumeChapterRe = regexp.MustCompile(
	`(?i)^Vol(?:ume)?\.?\s*(\d+(?:\.\d+)?)\s*[-:]?\s*Ch(?:apter)?\.?\s*(\d+(?:\.\d+)?)([xyz]\d+)?(?:\s*[-:]\s*(.+?))?(?:\s*\((\w[\w-]*)\))?(?:\s*\[(.+?)\])?\s*$`,
)

// VolumeChapterParser recognizes spelled-out "Volume N Chapter M" naming.
type VolumeChapterParser struct{}

func (VolumeChapterParser) Name() string { return "volume-chapter" }

func (VolumeChapterParser) Parse(dirName string) (ParsedChapter, bool) {
	m := volumeChapterRe.FindStringSubmatch(dirName)
	if m == nil {
		return ParsedChapter{}, false
	}

	vol, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return ParsedChapter{}, false
	}
	ch, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return ParsedChapter{}, false
	}

	return ParsedChapter{
		Volume:  &vol,
		Chapter: ch,
		Special: m[3],
		Title:   m[4],
		Lang:    m[5],
		Group:   m[6],
	}, true
}
