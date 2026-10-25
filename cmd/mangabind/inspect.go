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
	chapters = m.chooseVersions(chapters, metaMap)
	result = grouper.GroupLabeled(chapters, unparsed, m.opts.labels)
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

// inspection is one chapter folder or archive to read, with the volume of the folder it was found in when it
// was found in a volume folder.
type inspection struct {
	entry        scanner.ChapterEntry
	folderVolume *float64
}

// inspectChapters turns each entry into a unit of the report, and into a chapter to group when it can be.
// unparsed has the names of the entries that cannot: those no parser read, and those that held no pages.
func (m *mangaRun) inspectChapters(ctx context.Context, entries []scanner.ChapterEntry, metaMap *metadata.Map) (chapters []grouper.Chapter, unparsed []string, err error) {
	reg := parser.DefaultRegistry()
	for _, in := range m.expandVolumeFolders(reg, entries) {
		if cause := ctx.Err(); cause != nil {
			return nil, nil, m.rep.interrupted("inspect", "Interrupted while inspecting the chapters; nothing was written.", "", cause)
		}
		chapter, ok, err := m.inspectEntry(reg, in.entry, in.folderVolume, metaMap)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			unparsed = append(unparsed, listedName(in.entry))
			continue
		}
		chapters = append(chapters, chapter)
	}
	return chapters, unparsed, nil
}

// expandVolumeFolders puts the chapters found in a volume folder in the place of the folder: a folder named
// for a volume alone that has no pages of its own and holds chapter folders or archives
// (docs/adr/0027-volume-folders-hold-chapters.md). Any other entry stays as it is.
func (m *mangaRun) expandVolumeFolders(reg *parser.Registry, entries []scanner.ChapterEntry) []inspection {
	var expanded []inspection
	for _, e := range entries {
		if volume, inside, ok := m.chaptersOfVolumeFolder(reg, e); ok {
			for _, chapter := range inside {
				expanded = append(expanded, inspection{entry: chapter, folderVolume: &volume})
			}
			continue
		}
		expanded = append(expanded, inspection{entry: e})
	}
	return expanded
}

// chaptersOfVolumeFolder lists the chapters in e when e is a volume folder. A folder with pages is the
// volume itself (ADR 0024), and one with nothing in it that can be a chapter is read as any other folder, so
// that it is reported as having no pages.
func (m *mangaRun) chaptersOfVolumeFolder(reg *parser.Registry, e scanner.ChapterEntry) (volume float64, chapters []scanner.ChapterEntry, ok bool) {
	if e.IsArchive {
		return 0, nil, false
	}
	parsed, _, matched := reg.Parse(e.Name)
	if !matched || !parsed.VolumeUnit || parsed.Volume == nil {
		return 0, nil, false
	}
	pages, _, pageLinks, err := scanner.Pages(m.input, e.Path)
	if err != nil || len(pages) > 0 {
		return 0, nil, false
	}
	inside, skipped, links, err := scanner.ScanIn(m.input, e.Path)
	if err != nil || len(inside) == 0 {
		return 0, nil, false
	}
	m.rep.links(pageLinks, nil)
	m.rep.links(links, nil)
	for _, name := range skipped {
		m.rep.warn("unsupported_input_file", "inspect", unsupportedFileMessage(name), atPath(absolutePath(filepath.Join(e.Path, name))))
	}
	return *parsed.Volume, inside, true
}

// listedName is how an entry is named in a message: its name, or its path below the manga folder when it is
// in a volume folder.
func listedName(e scanner.ChapterEntry) string {
	if e.Parent == "" {
		return e.Name
	}
	return filepath.Join(e.Parent, e.Name)
}

// inspectEntry reads one chapter folder or archive: its name, the volume the metadata file gives it, its
// pages. Its unit goes into the report whatever becomes of it; ok is whether it is a chapter that can be
// grouped.
func (m *mangaRun) inspectEntry(reg *parser.Registry, e scanner.ChapterEntry, folderVolume *float64, metaMap *metadata.Map) (chapter grouper.Chapter, ok bool, err error) {
	parsed, parserName, matched := reg.Parse(e.Name)
	unit := unitReport{
		Name:               e.Name,
		Path:               absolutePath(e.Path),
		Kind:               "folder",
		Disposition:        "candidate",
		MetadataAssignment: metadataAssignmentReport{Status: "not_requested"},
		listed:             listedName(e),
	}
	if e.IsArchive {
		unit.Kind = "cbz"
	}
	if e.Parent != "" {
		unit.VolumeFolder = absolutePath(filepath.Join(m.input, e.Parent))
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
	if m.opts.noVolumes {
		m.leaveVolumes(&unit, &parsed)
	} else {
		if folderVolume != nil {
			m.takeFolderVolume(&parsed, *folderVolume, e)
		}
		m.assignVolume(&unit, &parsed, e, metaMap)
	}
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
	if len(pages) > 0 {
		unit.FirstPage = pages[0]
	}
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

// leaveVolumes puts a chapter in the one volume of a book with no volumes, whatever its name, its folder or the
// mapping file say: they are what the report keeps of it, and none of them is a conflict, since none is used.
func (m *mangaRun) leaveVolumes(unit *unitReport, parsed *parser.ParsedChapter) {
	unit.MetadataAssignment.Status = "not_applicable"
	one := grouper.WholeBookVolume
	parsed.Volume = &one
}

// takeFolderVolume gives a chapter found in a volume folder that carries no volume in its name the volume of
// the folder. A volume in the name wins, as it does over the metadata file, and a name that says another
// volume than its folder is a warning.
func (m *mangaRun) takeFolderVolume(parsed *parser.ParsedChapter, folderVolume float64, e scanner.ChapterEntry) {
	switch {
	case parsed.Volume == nil:
		parsed.Volume = &folderVolume
	case *parsed.Volume != folderVolume:
		m.rep.sayf("warning: chapter %v%s is in the folder of volume %v, but its name says volume %v - keeping the name",
			parsed.Chapter, parsed.Special, folderVolume, *parsed.Volume)
		m.rep.record("warning", "volume_folder_conflict", "inspect",
			fmt.Sprintf("Chapter %v%s is in the folder of volume %v but its name says volume %v; it keeps the volume from its name.", parsed.Chapter, parsed.Special, folderVolume, *parsed.Volume),
			forChapter(parsed.Volume, parsed.Chapter, parsed.Special), atPath(absolutePath(e.Path)))
	}
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
