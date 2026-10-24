package main

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/gustavommcv/mangabind/internal/grouper"
	"github.com/gustavommcv/mangabind/internal/metadata"
)

// versionGroupReport is one chapter that more than one file claims: the same chapter downloaded from two scan
// groups, in two languages, or twice. Without a volume to put them in they are not yet a conflict, and the
// report tells them all the same, so that a consumer can ask which to use before it has a mapping
// (docs/adr/0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md).
type versionGroupReport struct {
	Chapter float64  `json:"chapter"`
	Special string   `json:"special,omitempty"`
	Volume  *float64 `json:"volume,omitempty"`
	// Units are the absolute paths of the files that claim the chapter, in the order they were found, whether
	// or not they were chosen: each unit's own disposition says which were.
	Units []string `json:"units"`
	// Resolution is "chosen" when the mapping file chose among the files, and "unresolved" when it did not.
	Resolution string `json:"resolution"`
}

// versionGroup is the chapters that claim one chapter number in one volume, by their index in the chapters
// that were read.
type versionGroup struct {
	chapter float64
	special string
	volume  *float64
	members []int
}

// versionGroups finds the chapters that more than one file claims. They are the files with one chapter number
// and special suffix and one effective volume, which is nothing for the files that have none: with the volume
// the mapping file gives them, they would all be in it.
func versionGroups(chapters []grouper.Chapter) []versionGroup {
	type groupKey struct {
		chapter   float64
		special   string
		hasVolume bool
		volume    float64
	}
	index := map[groupKey]int{}
	var groups []versionGroup
	for i, c := range chapters {
		key := groupKey{chapter: c.Parsed.Chapter, special: c.Parsed.Special}
		if c.Parsed.Volume != nil {
			key.hasVolume, key.volume = true, *c.Parsed.Volume
		}
		at, seen := index[key]
		if !seen {
			at = len(groups)
			index[key] = at
			groups = append(groups, versionGroup{chapter: key.chapter, special: key.special, volume: copyFloat(c.Parsed.Volume)})
		}
		groups[at].members = append(groups[at].members, i)
	}
	claimed := groups[:0]
	for _, g := range groups {
		if len(g.members) > 1 {
			claimed = append(claimed, g)
		}
	}
	sort.SliceStable(claimed, func(i, j int) bool {
		left, right := claimed[i], claimed[j]
		switch {
		case (left.volume == nil) != (right.volume == nil):
			return left.volume == nil
		case left.volume != nil && *left.volume != *right.volume:
			return *left.volume < *right.volume
		case left.chapter != right.chapter:
			return left.chapter < right.chapter
		}
		return left.special < right.special
	})
	return claimed
}

// chooseVersions tells in the report the chapters that more than one file claims, and applies what the mapping
// file chose among them: the files it does not name are left out of the chapters that come back, with the
// disposition not_chosen, and the files it names are kept. More than one kept file is all of them kept as
// alternatives of the chapter, each told which it is of how many. A choice that names none of the files that
// claim the chapter is not used, and is told; the files are then a conflict, as they are with no choice.
func (m *mangaRun) chooseVersions(chapters []grouper.Chapter, metaMap *metadata.Map) []grouper.Chapter {
	groups := versionGroups(chapters)
	if len(groups) == 0 {
		return chapters
	}
	left := map[int]bool{}
	for _, g := range groups {
		claimants := make([]string, len(g.members))
		for i, member := range g.members {
			claimants[i] = absolutePath(chapters[member].Path)
		}
		resolution := "unresolved"
		if metaMap != nil {
			if use, listed := metaMap.Choice(g.chapter, g.special); listed {
				if m.applyChoice(chapters, g, use, left) {
					resolution = "chosen"
				} else {
					m.rep.warn("chapter_choice_unmatched", "metadata", unmatchedChoiceMessage(g), forChapter(g.volume, g.chapter, g.special), relatedTo(claimants))
				}
			}
		}
		m.report.ChapterVersions = append(m.report.ChapterVersions, versionGroupReport{
			Chapter: g.chapter, Special: g.special, Volume: g.volume, Units: claimants, Resolution: resolution,
		})
	}
	if len(left) == 0 {
		return chapters
	}
	kept := make([]grouper.Chapter, 0, len(chapters)-len(left))
	for i, c := range chapters {
		if !left[i] {
			kept = append(kept, c)
		}
	}
	return kept
}

// applyChoice keeps the files of the group that the choice names and puts the others in left, marking their
// units. It is false, and does nothing, when the choice names a file and none of them is one the group has.
func (m *mangaRun) applyChoice(chapters []grouper.Chapter, g versionGroup, use []string, left map[int]bool) bool {
	named := map[string]bool{}
	for _, file := range use {
		named[file] = true
	}
	var kept, others []int
	for _, member := range g.members {
		if named[m.pathBelowInput(chapters[member].Path)] {
			kept = append(kept, member)
		} else {
			others = append(others, member)
		}
	}
	if len(use) > 0 && len(kept) == 0 {
		return false
	}
	for _, member := range others {
		left[member] = true
		if unit := findUnit(m.report.Units, chapters[member].Path); unit != nil {
			unit.Disposition = "not_chosen"
		}
	}
	if len(kept) > 1 {
		for place, member := range kept {
			chapters[member].Version = &grouper.Version{Index: place + 1, Count: len(kept)}
		}
	}
	return true
}

// pathBelowInput is how the mapping file names a chapter's folder or archive: its path below the manga folder,
// written with "/".
func (m *mangaRun) pathBelowInput(path string) string {
	rel, err := filepath.Rel(m.input, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func unmatchedChoiceMessage(g versionGroup) string {
	return fmt.Sprintf("The mapping file chooses files for chapter %v%s that are not among the %d that claim it, so the choice was not used.", g.chapter, g.special, len(g.members))
}
