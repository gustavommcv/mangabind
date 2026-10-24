package parser

import "regexp"

// Matches a name that says an episode where another says a chapter: `Episode 5`, `Ep. 5`, `ep05`, as a
// webcomic, or any series published by episodes, numbers its chapters. The title, the language and the group
// may follow it as they follow a chapter (`Ep.07 - Title (en) [Group]`).
var episodeRe = regexp.MustCompile(widen(
	`(?i)^(?:episode|ep)\.?\s*[0０]*(\d+(?:[.．]\d+)?)([xyz]\d+)?(?:\s*[-:]\s*(.+?))?(?:\s*\((` + languageTagPattern + `)\))?(?:\s*\[(.+?)\])?\s*$`,
))

// EpisodeParser recognizes names that number an episode, which is a chapter that has no volume, and may
// follow the name of the series (`peppercarrot_episode01`, `Pepper Carrot Episode 7`). See
// docs/adr/0030-episodes-are-chapters.md.
type EpisodeParser struct{}

func (EpisodeParser) Name() string { return "episode" }

func (EpisodeParser) Parse(dirName string) (ParsedChapter, bool) {
	return afterPrefix(dirName, parseEpisode)
}

func parseEpisode(name string) (ParsedChapter, bool) {
	m := episodeRe.FindStringSubmatch(name)
	if m == nil {
		return ParsedChapter{}, false
	}
	chapter, err := parseNumber(m[1])
	if err != nil {
		return ParsedChapter{}, false
	}
	return ParsedChapter{
		Chapter: chapter,
		Special: foldWidth(m[2]),
		Title:   m[3],
		Lang:    m[4],
		Group:   m[5],
	}, true
}
