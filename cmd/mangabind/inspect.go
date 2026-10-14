package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gustavommcv/mangabind/internal/grouper"
	"github.com/gustavommcv/mangabind/internal/metadata"
	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

// inspect reads what is in the folder of the manga and groups it into volumes, filling in the units of the
// report on the way. found is false, with no error, when the folder has no chapters at all: there is
// nothing more to do, and the report says so.
func (m *mangaRun) inspect(ctx context.Context) (result grouper.Result, found bool, err error) {
	metaMap, err := m.loadMetadata()
	if err != nil {
		return result, false, err
	}
	entries, err := m.scanInput()
	if err != nil {
		return result, false, err
	}
	if len(entries) == 0 {
		// A warning, so -quiet does not hide that nothing was done.
		m.rep.sayWarning("no chapter folders or .cbz files found in " + m.input)
		m.rep.record("warning", "no_chapters_found", "inspect", "No chapter folders or CBZ files were found.", atPath(absolutePath(m.input)))
		return result, false, nil
	}
	chapters, unparsed, err := m.inspectChapters(ctx, entries, metaMap)
	if err != nil {
		return result, false, err
	}
	result = grouper.Group(chapters, unparsed)
	markUnitDispositions(m.report.Units, result)
	m.progress(machineProgress{Stage: "inspect", State: "completed", Manga: m.name})
	return result, true, nil
}

// loadMetadata loads the metadata file that applies to the manga, if one does; with none, the map is nil.
func (m *mangaRun) loadMetadata() (*metadata.Map, error) {
	mf := resolveMetadataFile(m.input, m.opts.metadataFile)
	if mf == "" {
		return nil, nil
	}
	m.report.MetadataFile = absolutePath(mf)
	loaded, err := metadata.Load(mf)
	if err != nil {
		wrapped := fmt.Errorf("loading metadata file %s: %w", mf, err)
		m.rep.fail("metadata_load_failed", "metadata", "Couldn't load the chapter-to-volume metadata file.", wrapped, atPath(absolutePath(mf)))
		return nil, wrapped
	}
	if m.opts.human && !m.opts.quiet {
		printfTo(m.opts.stderr, "mangabind: using metadata file %s\n", mf)
	}
	if duplicates := loaded.Duplicates(); len(duplicates) > 0 {
		m.rep.warn("metadata_duplicate_chapter", "metadata", duplicateChaptersMessage(duplicates), atPath(absolutePath(mf)))
	}
	return loaded, nil
}

// scanInput lists the chapters in the folder, and tells of the links and the files that were left out.
func (m *mangaRun) scanInput() ([]scanner.ChapterEntry, error) {
	entries, skipped, links, err := scanner.Scan(m.input)
	if err != nil {
		wrapped := fmt.Errorf("scanning %s: %w", m.input, err)
		m.rep.fail("input_scan_failed", "inspect", "Couldn't inspect the manga folder.", wrapped, atPath(absolutePath(m.input)))
		return nil, wrapped
	}
	m.rep.links(links, nil)
	for _, name := range skipped {
		m.rep.warn("unsupported_input_file", "inspect", unsupportedFileMessage(name), atPath(absolutePath(filepath.Join(m.input, name))))
	}
	return entries, nil
}

// inspectChapters turns each entry into a unit of the report, and into a chapter to group when it can be.
// unparsed has the names of the entries that cannot: those no parser read, and those that held no pages.
func (m *mangaRun) inspectChapters(ctx context.Context, entries []scanner.ChapterEntry, metaMap *metadata.Map) (chapters []grouper.Chapter, unparsed []string, err error) {
	reg := parser.DefaultRegistry()
	for _, e := range entries {
		if cause := ctx.Err(); cause != nil {
			return nil, nil, m.rep.interrupted("inspect", "Interrupted while inspecting the chapters; nothing was written.", "", cause)
		}
		chapter, ok, err := m.inspectEntry(reg, e, metaMap)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			unparsed = append(unparsed, e.Name)
			continue
		}
		chapters = append(chapters, chapter)
	}
	return chapters, unparsed, nil
}

// inspectEntry reads one chapter folder or archive: its name, the volume the metadata file gives it, its
// pages. Its unit goes into the report whatever becomes of it; ok is whether it is a chapter that can be
// grouped.
func (m *mangaRun) inspectEntry(reg *parser.Registry, e scanner.ChapterEntry, metaMap *metadata.Map) (chapter grouper.Chapter, ok bool, err error) {
	parsed, parserName, matched := reg.Parse(e.Name)
	unit := unitReport{
		Name:               e.Name,
		Path:               absolutePath(e.Path),
		Kind:               "folder",
		Disposition:        "candidate",
		MetadataAssignment: metadataAssignmentReport{Status: "not_requested"},
	}
	if e.IsArchive {
		unit.Kind = "cbz"
	}
	if !matched {
		unit.Parser = parserReport{Matched: false}
		unit.MetadataAssignment.Status = "not_applicable"
		unit.Disposition = "unparsed"
		m.report.Units = append(m.report.Units, unit)
		return chapter, false, nil
	}

	chapterNumber := parsed.Chapter
	unit.Parser = parserReport{
		Matched:  true,
		Name:     parserName,
		Volume:   copyFloat(parsed.Volume),
		Chapter:  &chapterNumber,
		Special:  parsed.Special,
		Title:    parsed.Title,
		Group:    parsed.Group,
		Language: parsed.Lang,
	}
	m.assignVolume(&unit, &parsed, e, metaMap)
	unit.EffectiveVolume = copyFloat(parsed.Volume)

	pages, notPages, tooLarge, err := m.listPages(e, &parsed)
	if err != nil {
		wrapped := fmt.Errorf("listing pages in %s: %w", e.Path, err)
		unit.Disposition = "failed"
		m.report.Units = append(m.report.Units, unit)
		m.rep.fail("page_listing_failed", "inspect", fmt.Sprintf("Couldn't list pages for %q.", e.Name), wrapped, atPath(absolutePath(e.Path)))
		return chapter, false, wrapped
	}
	unit.PageCount = len(pages)
	m.rep.skippedPages(e, &parsed, notPages)
	m.rep.oversizedPages(e, &parsed, tooLarge)
	if len(pages) == 0 {
		unit.Disposition = "empty"
		m.report.Units = append(m.report.Units, unit)
		return chapter, false, nil
	}

	m.report.Units = append(m.report.Units, unit)
	return grouper.Chapter{Parsed: parsed, Path: e.Path, Pages: pages, IsArchive: e.IsArchive}, true, nil
}

// assignVolume gives a chapter that carries no volume in its name the volume the metadata file lists it
// under. A volume in the name always wins: the unit records whether the file agreed, and a disagreement is
// a warning.
func (m *mangaRun) assignVolume(unit *unitReport, parsed *parser.ParsedChapter, e scanner.ChapterEntry, metaMap *metadata.Map) {
	if metaMap == nil {
		return
	}
	unit.MetadataAssignment.Status = "not_found"
	metaVol, found := metaMap.Lookup(parsed.Chapter, parsed.Special)
	if !found {
		return
	}
	metadataVolume := metaVol
	unit.MetadataAssignment.MetadataVolume = &metadataVolume
	switch {
	case parsed.Volume == nil:
		v := metaVol
		parsed.Volume = &v
		unit.MetadataAssignment.Status = "applied"
	case *parsed.Volume == metaVol:
		unit.MetadataAssignment.Status = "confirmed"
	case *parsed.Volume != metaVol:
		unit.MetadataAssignment.Status = "ignored_conflict"
		m.rep.sayf("warning: chapter %v%s: name says volume %v, metadata file says volume %v - keeping the name",
			parsed.Chapter, parsed.Special, *parsed.Volume, metaVol)
		m.rep.record("warning", "metadata_volume_conflict", "metadata",
			fmt.Sprintf("Chapter %v%s keeps volume %v from its name instead of metadata volume %v.", parsed.Chapter, parsed.Special, *parsed.Volume, metaVol),
			forChapter(parsed.Volume, parsed.Chapter, parsed.Special), atPath(absolutePath(e.Path)))
	}
}

// listPages lists the pages of a chapter folder or archive, and tells of the links in a folder that were
// not followed. The entries of an archive that are too large are the only ones that come back as such.
func (m *mangaRun) listPages(e scanner.ChapterEntry, parsed *parser.ParsedChapter) (pages, notPages []string, tooLarge []scanner.Oversized, err error) {
	if e.IsArchive {
		return scanner.PagesInArchive(e.Path)
	}
	pages, notPages, pageLinks, err := scanner.Pages(m.input, e.Path)
	m.rep.links(pageLinks, parsed)
	return pages, notPages, nil, err
}
