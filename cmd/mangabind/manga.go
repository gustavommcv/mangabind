package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/gustavommcv/mangabind/internal/cbz"
	"github.com/gustavommcv/mangabind/internal/grouper"
	"github.com/gustavommcv/mangabind/internal/metadata"
	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

// mangaSummary is what processManga reports back so runBatch can print a
// final tally without processManga needing to know it's running in a batch.
type mangaSummary struct {
	volumes int
	pages   int
}

// processManga reads the chapters in input and writes its volumes as the options say. It gives the tally
// of what was written, the report of the manga, and the error that ended the run, if one did.
func processManga(ctx context.Context, input string, opts runOptions) (mangaSummary, mangaReport, error) {
	output, quiet, dryRun, combine, human := opts.output, opts.quiet, opts.dryRun, opts.combine, opts.human
	stdout, stderr, progress := opts.stdout, opts.stderr, opts.progress
	var summary mangaSummary
	mangaName := folderName(input)
	report := newMangaReport(mangaName, absolutePath(input))
	r := reporter{manga: mangaName, report: &report, human: human, stderr: stderr}
	if progress != nil {
		progress(machineProgress{Stage: "inspect", State: "started", Manga: mangaName})
	}

	var metaMap *metadata.Map
	if mf := resolveMetadataFile(input, opts.metadataFile); mf != "" {
		report.MetadataFile = absolutePath(mf)
		m, err := metadata.Load(mf)
		if err != nil {
			wrapped := fmt.Errorf("loading metadata file %s: %w", mf, err)
			r.fail("metadata_load_failed", "metadata", "Couldn't load the chapter-to-volume metadata file.", wrapped, atPath(absolutePath(mf)))
			finalizeMangaReport(&report, summary)
			return summary, report, wrapped
		}
		metaMap = m
		if human && !quiet {
			printfTo(stderr, "mangabind: using metadata file %s\n", mf)
		}
		if duplicates := m.Duplicates(); len(duplicates) > 0 {
			r.warn("metadata_duplicate_chapter", "metadata", duplicateChaptersMessage(duplicates), atPath(absolutePath(mf)))
		}
	}

	entries, skipped, links, err := scanner.Scan(input)
	if err != nil {
		wrapped := fmt.Errorf("scanning %s: %w", input, err)
		r.fail("input_scan_failed", "inspect", "Couldn't inspect the manga folder.", wrapped, atPath(absolutePath(input)))
		finalizeMangaReport(&report, summary)
		return summary, report, wrapped
	}
	r.links(links, nil)
	for _, name := range skipped {
		r.warn("unsupported_input_file", "inspect", unsupportedFileMessage(name), atPath(absolutePath(filepath.Join(input, name))))
	}
	if len(entries) == 0 {
		// A warning, so -quiet does not hide that nothing was done.
		r.sayWarning("no chapter folders or .cbz files found in " + input)
		r.record("warning", "no_chapters_found", "inspect", "No chapter folders or CBZ files were found.", atPath(absolutePath(input)))
		finalizeMangaReport(&report, summary)
		return summary, report, nil
	}

	reg := parser.DefaultRegistry()
	var chapters []grouper.Chapter
	var unparsed []string

	for _, e := range entries {
		if cause := ctx.Err(); cause != nil {
			err := r.interrupted("inspect", "Interrupted while inspecting the chapters; nothing was written.", "", cause)
			finalizeMangaReport(&report, summary)
			return summary, report, err
		}
		parsed, parserName, ok := reg.Parse(e.Name)
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
		if !ok {
			unit.Parser = parserReport{Matched: false}
			unit.MetadataAssignment.Status = "not_applicable"
			unit.Disposition = "unparsed"
			report.Units = append(report.Units, unit)
			unparsed = append(unparsed, e.Name)
			continue
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
		if metaMap != nil {
			unit.MetadataAssignment.Status = "not_found"
		}

		if metaMap != nil {
			if metaVol, found := metaMap.Lookup(parsed.Chapter, parsed.Special); found {
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
					r.sayf("warning: chapter %v%s: name says volume %v, metadata file says volume %v - keeping the name",
						parsed.Chapter, parsed.Special, *parsed.Volume, metaVol)
					r.record("warning", "metadata_volume_conflict", "metadata",
						fmt.Sprintf("Chapter %v%s keeps volume %v from its name instead of metadata volume %v.", parsed.Chapter, parsed.Special, *parsed.Volume, metaVol),
						forChapter(parsed.Volume, parsed.Chapter, parsed.Special), atPath(absolutePath(e.Path)))
				}
			}
		}
		unit.EffectiveVolume = copyFloat(parsed.Volume)

		var pages, notPages []string
		var tooLarge []scanner.Oversized
		if e.IsArchive {
			pages, notPages, tooLarge, err = scanner.PagesInArchive(e.Path)
		} else {
			var pageLinks []scanner.Link
			pages, notPages, pageLinks, err = scanner.Pages(input, e.Path)
			r.links(pageLinks, &parsed)
		}
		if err != nil {
			wrapped := fmt.Errorf("listing pages in %s: %w", e.Path, err)
			unit.Disposition = "failed"
			report.Units = append(report.Units, unit)
			r.fail("page_listing_failed", "inspect", fmt.Sprintf("Couldn't list pages for %q.", e.Name), wrapped, atPath(absolutePath(e.Path)))
			finalizeMangaReport(&report, summary)
			return summary, report, wrapped
		}
		unit.PageCount = len(pages)
		r.skippedPages(e, &parsed, notPages)
		r.oversizedPages(e, &parsed, tooLarge)
		if len(pages) == 0 {
			unit.Disposition = "empty"
			report.Units = append(report.Units, unit)
			unparsed = append(unparsed, e.Name)
			continue
		}

		report.Units = append(report.Units, unit)
		chapters = append(chapters, grouper.Chapter{Parsed: parsed, Path: e.Path, Pages: pages, IsArchive: e.IsArchive})
	}

	result := grouper.Group(chapters, unparsed)
	markUnitDispositions(report.Units, result)
	if progress != nil {
		progress(machineProgress{Stage: "inspect", State: "completed", Manga: mangaName})
	}
	var totalPages int
	for _, volume := range result.Volumes {
		totalPages += len(volume.Pages)
	}
	completedPages := 0
	emitWrite := func(state string, index int, volume float64, completed int) {
		if progress != nil {
			progress(machineProgress{
				Stage: "write", State: state, Manga: mangaName,
				VolumeIndex: index + 1, VolumeCount: len(result.Volumes), VolumeNumber: &volume,
				CompletedPages: &completed, TotalPages: totalPages,
			})
		}
	}

	// Volumes that could not be written, in the per-volume mode: see below.
	failedVolumes := 0
	if combine {
		// One .cbz for the whole manga instead of one per volume - see
		// docs/adr/0012-combine-series-into-one-volume.md. Each volume still
		// gets its own report entry (chapters, page count, number), but its
		// pages physically live inside the one combined file; grouper.Group
		// above is untouched, only how the result is written differs.
		outPath := filepath.Join(output, cbz.SeriesFileName(mangaName))
		report.CombinedOutputPath = absolutePath(outPath)
		for _, vol := range result.Volumes {
			report.Volumes = append(report.Volumes, volumeReport{
				Number:     vol.Number,
				OutputPath: absolutePath(outPath),
				PageCount:  len(vol.Pages),
				Chapters:   chapterNamesForVolume(report.Units, vol.Number),
				Written:    false,
			})
			summary.volumes++
			summary.pages += len(vol.Pages)
		}
		if dryRun {
			if human {
				printfTo(stdout, "would write %s (%d pages, %d volumes)\n", outPath, summary.pages, summary.volumes)
			}
		} else {
			combined := grouper.CombineVolumes(result.Volumes)
			if len(result.Volumes) > 0 {
				emitWrite("started", 0, result.Volumes[0].Number, 0)
			}
			volumeIndex := 0
			volumeEnd := 0
			if len(result.Volumes) > 0 {
				volumeEnd = len(result.Volumes[0].Pages)
			}
			var onPage func(int)
			if progress != nil {
				onPage = func(completed int) {
					for volumeIndex+1 < len(result.Volumes) && completed > volumeEnd {
						volumeIndex++
						volumeEnd += len(result.Volumes[volumeIndex].Pages)
					}
					if len(result.Volumes) > 0 {
						emitWrite("advanced", volumeIndex, result.Volumes[volumeIndex].Number, completed)
					}
				}
			}
			if err := cbz.WriteWithProgress(ctx, outPath, combined, onPage); err != nil {
				// Nothing was written: the volumes were counted as the plan was
				// made, and the tally is of what is on disk.
				summary = mangaSummary{}
				if errors.Is(err, context.Canceled) {
					err = r.interrupted("write", "Interrupted while writing the combined series; the unfinished file was removed.", absolutePath(outPath), err)
					finalizeMangaReport(&report, summary)
					return summary, report, err
				}
				wrapped := fmt.Errorf("writing combined series: %w", err)
				r.fail("combined_write_failed", "write", "Couldn't write the combined series.", wrapped, atPath(absolutePath(outPath)))
				finalizeMangaReport(&report, summary)
				return summary, report, wrapped
			}
			if len(result.Volumes) > 0 {
				last := len(result.Volumes) - 1
				emitWrite("completed", last, result.Volumes[last].Number, totalPages)
			}
			if human && !quiet {
				printfTo(stdout, "wrote %s (%d pages, %d volumes)\n", outPath, summary.pages, summary.volumes)
			}
		}
	} else {
		// stopAt ends the run at result.Volumes[index]: that volume and the ones
		// after it are in the report as planned and not written, as they would
		// be after a failure, and the report says why.
		stopAt := func(index int, message, path string, cause error) (mangaSummary, mangaReport, error) {
			for _, vol := range result.Volumes[index:] {
				report.Volumes = append(report.Volumes, volumeReport{
					Number:     vol.Number,
					OutputPath: absolutePath(filepath.Join(output, cbz.VolumeFileName(mangaName, vol.Number))),
					PageCount:  len(vol.Pages),
					Chapters:   chapterNamesForVolume(report.Units, vol.Number),
				})
			}
			err := r.interrupted("write", message, path, cause)
			finalizeMangaReport(&report, summary)
			return summary, report, err
		}
		for index, vol := range result.Volumes {
			outPath := filepath.Join(output, cbz.VolumeFileName(mangaName, vol.Number))
			if cause := ctx.Err(); cause != nil {
				return stopAt(index, fmt.Sprintf("Interrupted before volume %v was written.", vol.Number), "", cause)
			}
			volume := volumeReport{
				Number:     vol.Number,
				OutputPath: absolutePath(outPath),
				PageCount:  len(vol.Pages),
				Chapters:   chapterNamesForVolume(report.Units, vol.Number),
				Written:    false,
			}
			if dryRun {
				if human {
					printfTo(stdout, "would write %s (%d pages)\n", outPath, len(vol.Pages))
				}
			} else {
				emitWrite("started", index, vol.Number, completedPages)
				copied := 0
				var onPage func(int)
				if progress != nil {
					onPage = func(completed int) {
						copied = completed
						emitWrite("advanced", index, vol.Number, completedPages+completed)
					}
				}
				if err := cbz.WriteWithProgress(ctx, outPath, vol.Pages, onPage); err != nil {
					if errors.Is(err, context.Canceled) {
						return stopAt(index, fmt.Sprintf("Interrupted while writing volume %v; the unfinished file was removed.", vol.Number), absolutePath(outPath), err)
					}
					// Report it and bind the next volume: what can be bound is
					// bound, and the error returned at the end says that
					// something was not. The pages that were copied stay in the
					// progress count, which never goes back.
					wrapped := fmt.Errorf("writing volume %v: %w", vol.Number, err)
					failedVolumes++
					completedPages += copied
					report.Volumes = append(report.Volumes, volume)
					r.fail("volume_write_failed", "write", fmt.Sprintf("Couldn't write volume %v.", vol.Number), wrapped, forVolume(&vol.Number), atPath(absolutePath(outPath)))
					r.say("mangabind: " + wrapped.Error())
					continue
				}
				completedPages += len(vol.Pages)
				emitWrite("completed", index, vol.Number, completedPages)
				volume.Written = true
				if human && !quiet {
					printfTo(stdout, "wrote %s (%d pages)\n", outPath, len(vol.Pages))
				}
			}
			report.Volumes = append(report.Volumes, volume)
			summary.volumes++
			summary.pages += len(vol.Pages)
		}
	}
	if len(result.Volumes) == 0 {
		if human {
			printlnTo(stdout, "mangabind: no volumes produced - see warnings below")
		}
		r.record("warning", "no_volumes_produced", "group", "No volumes could be produced from the inspected chapters.")
	}

	// result.Unparsed holds two kinds of chapter: the ones no parser read, and
	// the ones that were read and held no pages (an empty folder, one with only
	// junk, a .cbz whose only entries were too large). Saying "could not parse"
	// of the second is wrong, and since pages are now images only it is no
	// longer a rare case.
	var notParsed, noPages []string
	for _, name := range result.Unparsed {
		if unit := findUnit(report.Units, name); unit != nil && unit.Parser.Matched {
			noPages = append(noPages, name)
		} else {
			notParsed = append(notParsed, name)
		}
	}
	for _, line := range summarizeNames("could not parse chapter name, skipped", notParsed, "") {
		r.sayWarning(line)
	}
	for _, line := range summarizeNames("chapter has no pages, skipped", noPages, "") {
		r.sayWarning(line)
	}
	for _, name := range result.Unparsed {
		if unit := findUnit(report.Units, name); unit != nil && unit.Parser.Matched {
			r.record("warning", "empty_chapter", "inspect", fmt.Sprintf("Chapter %q contains no pages; it was skipped.", name), atPath(unit.Path))
			continue
		}
		r.record("warning", "unparsed_chapter", "parse", fmt.Sprintf("Couldn't parse chapter name %q; it was skipped.", name), atPath(absolutePath(filepath.Join(input, name))))
	}
	unassignedNames := make([]string, len(result.Unassigned))
	for i, c := range result.Unassigned {
		unassignedNames[i] = filepath.Base(c.Path)
	}
	for _, line := range summarizeNames("chapter has no volume number, skipped", unassignedNames, "see -metadata-file") {
		r.sayWarning(line)
	}
	for _, c := range result.Unassigned {
		chapter := c.Parsed.Chapter
		r.record("warning", "unassigned_chapter", "group",
			fmt.Sprintf("Chapter %v%s has no volume assignment and was skipped.", chapter, c.Parsed.Special),
			forChapter(nil, chapter, c.Parsed.Special), atPath(absolutePath(c.Path)))
	}
	for _, g := range result.Gaps {
		r.sayf("warning: volume %v is missing chapter(s) between %v and %v", g.Volume, g.After, g.Before)
		r.record("warning", "chapter_gap", "group",
			fmt.Sprintf("Volume %v is missing chapters between %v and %v.", g.Volume, g.After, g.Before),
			forVolume(&g.Volume))
	}
	for _, c := range result.Conflicts {
		r.sayf("warning: volume %v chapter %v%s has %d conflicting sources, all skipped:", c.Volume, c.Chapter, c.Special, len(c.Sources))
		related := make([]string, len(c.Sources))
		for i, source := range c.Sources {
			r.sayf("  - %q", filepath.Base(source))
			related[i] = absolutePath(source)
		}
		r.record("warning", "chapter_conflict", "group",
			fmt.Sprintf("Volume %v chapter %v%s has %d conflicting sources; all were skipped.", c.Volume, c.Chapter, c.Special, len(c.Sources)),
			forChapter(&c.Volume, c.Chapter, c.Special), relatedTo(related))
	}

	finalizeMangaReport(&report, summary)
	if failedVolumes > 0 {
		return summary, report, fmt.Errorf("%d of %d volume(s) could not be written", failedVolumes, len(result.Volumes))
	}
	return summary, report, nil
}
