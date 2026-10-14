package main

import (
	"path/filepath"
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

func findUnit(units []unitReport, name string) *unitReport {
	for i := range units {
		if units[i].Name == name {
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
		if unit := findUnit(units, filepath.Base(chapter.Path)); unit != nil {
			unit.Disposition = "unassigned"
		}
	}
	for _, conflict := range result.Conflicts {
		for _, source := range conflict.Sources {
			if unit := findUnit(units, filepath.Base(source)); unit != nil {
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
