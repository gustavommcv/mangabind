// Package metadata loads an optional local chapter-to-volume mapping,
// used to fill in the volume number for chapters whose folder/file name
// doesn't carry one (see docs/adr/0010-local-metadata-file.md). It never
// talks to the network - producing this file from an external source
// (an external API or anything else) is deliberately someone else's job.
package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// File is the on-disk schema of a metadata file.
type File struct {
	SchemaVersion int           `json:"schema_version"`
	Manga         *Manga        `json:"manga,omitempty"`
	Volumes       []VolumeEntry `json:"volumes"`
	Source        *Source       `json:"source,omitempty"`
}

// Manga is purely informational - never read by the matching logic.
type Manga struct {
	Title string `json:"title,omitempty"`
}

// VolumeEntry lists the chapters belonging to one volume. Each entry in
// Chapters is either a single chapter token ("8", "8.5", "21x1") or an
// inclusive integer range ("1-7"); ranges can't carry a special suffix -
// list those individually.
type VolumeEntry struct {
	Number   string   `json:"number"`
	Chapters []string `json:"chapters"`
}

// Source is purely informational - records where the data came from so a
// human (or a future tool) can refresh it later. Never read by Map/Lookup.
type Source struct {
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
}

const supportedSchemaVersion = 1

type chapterKey struct {
	chapter float64
	special string
}

// Map resolves a (chapter, special) pair to its volume number, loaded from
// a metadata File.
type Map struct {
	byChapter map[chapterKey]float64
}

// Load reads and parses the metadata file at path.
func Load(path string) (*Map, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if f.SchemaVersion != supportedSchemaVersion {
		return nil, fmt.Errorf("%s: unsupported schema_version %d (expected %d)",
			path, f.SchemaVersion, supportedSchemaVersion)
	}

	m := &Map{byChapter: map[chapterKey]float64{}}
	for _, vol := range f.Volumes {
		volNum, err := strconv.ParseFloat(vol.Number, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: volume %q: invalid number: %w", path, vol.Number, err)
		}
		for _, token := range vol.Chapters {
			keys, err := expandChapterToken(token)
			if err != nil {
				return nil, fmt.Errorf("%s: volume %q: %w", path, vol.Number, err)
			}
			for _, k := range keys {
				m.byChapter[k] = volNum
			}
		}
	}
	return m, nil
}

// Lookup returns the volume number for the given chapter/special
// combination, if the metadata file covers it.
func (m *Map) Lookup(chapter float64, special string) (volume float64, ok bool) {
	v, ok := m.byChapter[chapterKey{chapter, special}]
	return v, ok
}

var (
	chapterTokenRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)([xyz]\d+)?$`)
	rangeRe        = regexp.MustCompile(`^(\d+)-(\d+)$`)
)

// expandChapterToken parses one "chapters" array element into the one or
// more chapterKeys it denotes.
func expandChapterToken(token string) ([]chapterKey, error) {
	if lo, hi, ok := parseRange(token); ok {
		keys := make([]chapterKey, 0, hi-lo+1)
		for n := lo; n <= hi; n++ {
			keys = append(keys, chapterKey{float64(n), ""})
		}
		return keys, nil
	}

	m := chapterTokenRe.FindStringSubmatch(token)
	if m == nil {
		return nil, fmt.Errorf("invalid chapter token %q", token)
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid chapter token %q: %w", token, err)
	}
	return []chapterKey{{n, m[2]}}, nil
}

func parseRange(token string) (lo, hi int, ok bool) {
	m := rangeRe.FindStringSubmatch(token)
	if m == nil {
		return 0, 0, false
	}
	lo, errLo := strconv.Atoi(m[1])
	hi, errHi := strconv.Atoi(m[2])
	if errLo != nil || errHi != nil || lo > hi {
		return 0, 0, false
	}
	return lo, hi, true
}
