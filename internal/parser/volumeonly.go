package parser

import "regexp"

// Matches a name that is a volume and nothing else, in the English way, with the groups in brackets that a
// collection puts after it (a year, a publisher, a scanner): "v01", "Volume 01", "Vol.3.5 (2019) (Digital)".
var volumeOnlyRe = regexp.MustCompile(widen(
	`(?i)^(?:vol(?:ume)?\.?|v)\s*` + chapterNumber + `(?:\s*(?:\([^)]*\)|\[[^\]]*\]))*\s*$`,
))

// VolumeOnlyParser recognizes a folder or an archive that is named for a volume alone, as Kavita documents
// the name of a volume file (`Series Name v01.cbz`) and as the collections in its tests are named
// (`Love Hina - Volume 01 [Scans]`, `Battle Royale, v01 (2000) [TokyoPop]`). See
// docs/adr/0026-an-english-volume-named-unit-is-a-volume-of-one-chapter.md. Like JapaneseVolumeParser, it
// reads such a unit as a volume with one chapter numbered 1, and it reads it after a prefix, since the name
// of the series comes first. A volume with a title after it (`Vol. 01 - The threads of Time`) is not read.
type VolumeOnlyParser struct{}

func (VolumeOnlyParser) Name() string { return "volume-only" }

func (VolumeOnlyParser) Parse(dirName string) (ParsedChapter, bool) {
	return afterPrefix(dirName, parseVolumeOnly)
}

func parseVolumeOnly(s string) (ParsedChapter, bool) {
	m := volumeOnlyRe.FindStringSubmatch(s)
	if m == nil {
		return ParsedChapter{}, false
	}
	vol, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}
	return ParsedChapter{Volume: &vol, Chapter: 1}, true
}
