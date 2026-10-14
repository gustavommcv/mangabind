package main

import (
	"fmt"
	"path/filepath"

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

	// result.Unparsed holds two kinds of chapter: the ones no parser read, and
	// the ones that were read and held no pages (an empty folder, one with only
	// junk, a .cbz whose only entries were too large). Saying "could not parse"
	// of the second is wrong, and since pages are now images only it is no
	// longer a rare case.
	var notParsed, noPages []string
	for _, name := range result.Unparsed {
		if unit := findUnit(m.report.Units, name); unit != nil && unit.Parser.Matched {
			noPages = append(noPages, name)
		} else {
			notParsed = append(notParsed, name)
		}
	}
	for _, line := range summarizeNames("could not parse chapter name, skipped", notParsed, "") {
		m.rep.sayWarning(line)
	}
	for _, line := range summarizeNames("chapter has no pages, skipped", noPages, "") {
		m.rep.sayWarning(line)
	}
	for _, name := range result.Unparsed {
		if unit := findUnit(m.report.Units, name); unit != nil && unit.Parser.Matched {
			m.rep.record("warning", "empty_chapter", "inspect", fmt.Sprintf("Chapter %q contains no pages; it was skipped.", name), atPath(unit.Path))
			continue
		}
		m.rep.record("warning", "unparsed_chapter", "parse", fmt.Sprintf("Couldn't parse chapter name %q; it was skipped.", name), atPath(absolutePath(filepath.Join(m.input, name))))
	}
	unassignedNames := make([]string, len(result.Unassigned))
	for i, c := range result.Unassigned {
		unassignedNames[i] = filepath.Base(c.Path)
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
		m.rep.sayf("warning: volume %v is missing chapter(s) between %v and %v", g.Volume, g.After, g.Before)
		m.rep.record("warning", "chapter_gap", "group",
			fmt.Sprintf("Volume %v is missing chapters between %v and %v.", g.Volume, g.After, g.Before),
			forVolume(&g.Volume))
	}
	for _, c := range result.Conflicts {
		m.rep.sayf("warning: volume %v chapter %v%s has %d conflicting sources, all skipped:", c.Volume, c.Chapter, c.Special, len(c.Sources))
		related := make([]string, len(c.Sources))
		for i, source := range c.Sources {
			m.rep.sayf("  - %q", filepath.Base(source))
			related[i] = absolutePath(source)
		}
		m.rep.record("warning", "chapter_conflict", "group",
			fmt.Sprintf("Volume %v chapter %v%s has %d conflicting sources; all were skipped.", c.Volume, c.Chapter, c.Special, len(c.Sources)),
			forChapter(&c.Volume, c.Chapter, c.Special), relatedTo(related))
	}
}
