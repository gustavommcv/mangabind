package main

import (
	"sort"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

func copyFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// findUnit finds the unit of a chapter folder or archive by its path. The name would not do: two chapters in
// different volume folders may have one, and the name of an archive has no extension.
func findUnit(units []unitReport, path string) *unitReport {
	path = absolutePath(path)
	for i := range units {
		if units[i].Path == path {
			return &units[i]
		}
	}
	return nil
}

func markUnitDispositions(units []unitReport, result grouper.Result) {
	for i := range units {
		if units[i].Disposition == "candidate" {
			units[i].Disposition = "included"
		}
	}
	for _, chapter := range result.Unassigned {
		if unit := findUnit(units, chapter.Path); unit != nil {
			unit.Disposition = "unassigned"
		}
	}
	for _, conflict := range result.Conflicts {
		for _, source := range conflict.Sources {
			if unit := findUnit(units, source); unit != nil {
				unit.Disposition = "conflict"
			}
		}
	}
}

func chapterNamesForVolume(units []unitReport, volume float64) []string {
	var chapters []unitReport
	for _, unit := range units {
		if unit.Disposition == "included" && unit.EffectiveVolume != nil && *unit.EffectiveVolume == volume {
			chapters = append(chapters, unit)
		}
	}
	sort.Slice(chapters, func(i, j int) bool {
		left, right := chapters[i].Parser, chapters[j].Parser
		if left.Chapter != nil && right.Chapter != nil && *left.Chapter != *right.Chapter {
			return *left.Chapter < *right.Chapter
		}
		return left.Special < right.Special
	})
	names := make([]string, len(chapters))
	for i, chapter := range chapters {
		names[i] = chapter.Name
	}
	return names
}
