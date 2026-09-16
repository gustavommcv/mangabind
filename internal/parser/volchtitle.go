package parser

import (
	"regexp"
	"strconv"
)

// Matches "Vol.01 Ch.0001 - Title (en) [Group]" and close variants: an
// optional special-chapter suffix (x1/y2/z3), and an optional language
// and/or group tag.
var volChTitleRe = regexp.MustCompile(
	`(?i)^Vol\.(\d+(?:\.\d+)?)\s+Ch\.(\d+(?:\.\d+)?)([xyz]\d+)?\s*-\s*(.+?)(?:\s*\((\w[\w-]*)\))?(?:\s*\[(.+?)\])?\s*$`,
)

// VolChTitleParser recognizes the "Vol.NN Ch.NNNN - Title (lang) [Group]"
// convention, commonly used by scan releases.
type VolChTitleParser struct{}

func (VolChTitleParser) Name() string { return "vol-ch-title" }

func (VolChTitleParser) Parse(dirName string) (ParsedChapter, bool) {
	m := volChTitleRe.FindStringSubmatch(dirName)
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
