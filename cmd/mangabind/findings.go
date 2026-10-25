package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

// reportFindings says what the grouping found that could not be used: that no volume came out of it, the
// chapters that were not read or held no pages, the chapters with no volume, the gaps in a volume and the
// chapters that two sources claim.
func (m *mangaRun) reportFindings(result grouper.Result) {
	if len(result.Volumes) == 0 {
		if m.opts.human {
			printlnTo(m.opts.stdout, "mangabind: no volumes produced - see warnings below")
		}
		m.rep.record("warning", "no_volumes_produced", "group", "No volumes could be produced from the inspected chapters.")
	}

	// Two kinds of unit were left out before they could be grouped: the ones no parser read, and the ones that
	// were read and held no pages (an empty folder, one with only junk, a .cbz whose only entries were too
	// large). Saying "could not parse" of the second is wrong, and since pages are now images only it is no
	// longer a rare case.
	var notParsed, noPages []string
	for _, unit := range m.report.Units {
		switch unit.Disposition {
		case "unparsed":
			notParsed = append(notParsed, unit.listed)
		case "empty":
			noPages = append(noPages, unit.listed)
		}
	}
	for _, line := range summarizeNames("could not parse chapter name, skipped", notParsed, "") {
		m.rep.sayWarning(line)
	}
	for _, line := range summarizeNames("chapter has no pages, skipped", noPages, "") {
		m.rep.sayWarning(line)
	}
	for _, unit := range m.report.Units {
		switch unit.Disposition {
		case "empty":
			m.rep.record("warning", "empty_chapter", "inspect", fmt.Sprintf("Chapter %q contains no pages; it was skipped.", unit.listed), atPath(unit.Path))
		case "unparsed":
			m.rep.record("warning", "unparsed_chapter", "parse", fmt.Sprintf("Couldn't parse chapter name %q; it was skipped.", unit.listed), atPath(unit.Path))
		}
	}
	unassignedNames := make([]string, len(result.Unassigned))
	for i, c := range result.Unassigned {
		unassignedNames[i] = m.shown(c.Path)
	}
	for _, line := range summarizeNames("chapter has no volume number, skipped", unassignedNames, "see -metadata-file") {
		m.rep.sayWarning(line)
	}
	for _, c := range result.Unassigned {
		chapter := c.Parsed.Chapter
		m.rep.record("warning", "unassigned_chapter", "group",
			fmt.Sprintf("Chapter %v%s has no volume assignment and was skipped.", chapter, c.Parsed.Special),
			forChapter(nil, chapter, c.Parsed.Special), atPath(absolutePath(c.Path)))
	}
	for _, g := range result.Gaps {
		if m.opts.noVolumes {
			m.rep.sayf("warning: the book is missing chapter(s) between %v and %v", g.After, g.Before)
			m.rep.record("warning", "chapter_gap", "group",
				fmt.Sprintf("The book is missing chapters between %v and %v.", g.After, g.Before))
			continue
		}
		m.rep.sayf("warning: volume %v is missing chapter(s) between %v and %v", g.Volume, g.After, g.Before)
		m.rep.record("warning", "chapter_gap", "group",
			fmt.Sprintf("Volume %v is missing chapters between %v and %v.", g.Volume, g.After, g.Before),
			forVolume(&g.Volume))
	}
	for _, c := range result.Conflicts {
		if m.opts.noVolumes {
			m.reportConflictOfBook(c)
			continue
		}
		m.rep.sayf("warning: volume %v chapter %v%s has %d conflicting sources, all skipped:", c.Volume, c.Chapter, c.Special, len(c.Sources))
		related := make([]string, len(c.Sources))
		for i, source := range c.Sources {
			m.rep.sayf("  - %q", m.shown(source))
			related[i] = absolutePath(source)
		}
		m.rep.record("warning", "chapter_conflict", "group",
			fmt.Sprintf("Volume %v chapter %v%s has %d conflicting sources; all were skipped.", c.Volume, c.Chapter, c.Special, len(c.Sources)),
			forChapter(&c.Volume, c.Chapter, c.Special), relatedTo(related))
	}
}

// reportConflictOfBook tells of a chapter that two sources claim in a book with no volumes: there is no volume to
// name, and the chapters of two volumes that are numbered again from 1 are the ones that meet here.
func (m *mangaRun) reportConflictOfBook(c grouper.Conflict) {
	m.rep.sayf("warning: chapter %v%s has %d conflicting sources, all skipped:", c.Chapter, c.Special, len(c.Sources))
	related := make([]string, len(c.Sources))
	for i, source := range c.Sources {
		m.rep.sayf("  - %q", m.shown(source))
		related[i] = absolutePath(source)
	}
	m.rep.record("warning", "chapter_conflict", "group",
		fmt.Sprintf("Chapter %v%s has %d conflicting sources; all were skipped.", c.Chapter, c.Special, len(c.Sources)),
		forChapter(nil, c.Chapter, c.Special), relatedTo(related))
}

// shown names the folder or archive of a chapter in a message: its path below the manga folder, which for a
// chapter directly in it is its name.
func (m *mangaRun) shown(path string) string {
	rel, err := filepath.Rel(m.input, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.Base(path)
	}
	return rel
}
