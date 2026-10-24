package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/gustavommcv/mangabind/internal/cbz"
	"github.com/gustavommcv/mangabind/internal/grouper"
)

// writeEvents tells the progress sink how the writing goes: which volume is being written, and how many of
// the pages of the whole manga are done.
type writeEvents struct {
	sink       progressSink
	manga      string
	volumes    int
	totalPages int
}

func (w writeEvents) emit(state string, index int, volume float64, completed int) {
	if w.sink == nil {
		return
	}
	w.sink(machineProgress{
		Stage: "write", State: state, Manga: w.manga,
		VolumeIndex: index + 1, VolumeCount: w.volumes, VolumeNumber: &volume,
		CompletedPages: &completed, TotalPages: w.totalPages,
	})
}

// write binds the volumes of the result: one .cbz each, or one for the whole manga with -combine. It
// returns how many volumes could not be written when a volume that fails does not stop the others.
func (m *mangaRun) write(ctx context.Context, result grouper.Result) (failedVolumes int, err error) {
	var totalPages int
	for _, volume := range result.Volumes {
		totalPages += len(volume.Pages)
	}
	events := writeEvents{sink: m.opts.progress, manga: m.name, volumes: len(result.Volumes), totalPages: totalPages}
	if m.opts.combine {
		return 0, m.writeCombined(ctx, result, events)
	}
	return m.writeVolumes(ctx, result, events)
}

// plannedVolume is the entry of a volume in the report before it is written: where it goes, how many pages
// it has and which chapters are in it.
func (m *mangaRun) plannedVolume(vol grouper.Volume, outPath string) volumeReport {
	entries := make([]chapterEntryReport, len(vol.Chapters))
	for i, chapter := range vol.Chapters {
		entries[i] = chapterEntryReport{Label: chapter.Dir, Path: absolutePath(chapter.Path), PageCount: chapter.PageCount}
	}
	return volumeReport{
		Number:         vol.Number,
		OutputPath:     absolutePath(outPath),
		PageCount:      len(vol.Pages),
		Chapters:       chapterNamesForVolume(m.report.Units, vol.Number),
		ChapterEntries: entries,
	}
}

// writeCombined writes the whole manga as one .cbz instead of one per volume - see
// docs/adr/0012-combine-series-into-one-volume.md. Each volume still gets its own report entry (chapters,
// page count, number), but its pages physically live inside the one combined file; grouper.Group is
// untouched, only how the result is written differs.
func (m *mangaRun) writeCombined(ctx context.Context, result grouper.Result, events writeEvents) error {
	outPath := filepath.Join(m.opts.output, cbz.SeriesFileName(m.name))
	m.report.CombinedOutputPath = absolutePath(outPath)
	for _, vol := range result.Volumes {
		m.report.Volumes = append(m.report.Volumes, m.plannedVolume(vol, outPath))
		m.summary.volumes++
		m.summary.pages += len(vol.Pages)
	}
	if m.opts.dryRun {
		if m.opts.human {
			printfTo(m.opts.stdout, "would write %s (%d pages, %d volumes)\n", outPath, m.summary.pages, m.summary.volumes)
		}
		return nil
	}

	combined := grouper.CombineVolumes(result.Volumes)
	if len(result.Volumes) > 0 {
		events.emit("started", 0, result.Volumes[0].Number, 0)
	}
	volumeIndex := 0
	volumeEnd := 0
	if len(result.Volumes) > 0 {
		volumeEnd = len(result.Volumes[0].Pages)
	}
	var onPage func(int)
	if m.opts.progress != nil {
		onPage = func(completed int) {
			for volumeIndex+1 < len(result.Volumes) && completed > volumeEnd {
				volumeIndex++
				volumeEnd += len(result.Volumes[volumeIndex].Pages)
			}
			if len(result.Volumes) > 0 {
				events.emit("advanced", volumeIndex, result.Volumes[volumeIndex].Number, completed)
			}
		}
	}
	if err := cbz.WriteWithProgress(ctx, outPath, combined, onPage); err != nil {
		// Nothing was written: the volumes were counted as the plan was
		// made, and the tally is of what is on disk.
		m.summary = mangaSummary{}
		if errors.Is(err, context.Canceled) {
			return m.rep.interrupted("write", "Interrupted while writing the combined series; the unfinished file was removed.", absolutePath(outPath), err)
		}
		wrapped := fmt.Errorf("writing combined series: %w", err)
		m.rep.fail("combined_write_failed", "write", "Couldn't write the combined series.", wrapped, atPath(absolutePath(outPath)))
		return wrapped
	}
	if len(result.Volumes) > 0 {
		last := len(result.Volumes) - 1
		events.emit("completed", last, result.Volumes[last].Number, events.totalPages)
	}
	if m.opts.human && !m.opts.quiet {
		printfTo(m.opts.stdout, "wrote %s (%d pages, %d volumes)\n", outPath, m.summary.pages, m.summary.volumes)
	}
	return nil
}

// writeVolumes writes one .cbz for each volume. A volume that cannot be written is reported and the next
// one is bound: what can be bound is bound. An interruption ends the run.
func (m *mangaRun) writeVolumes(ctx context.Context, result grouper.Result, events writeEvents) (failedVolumes int, err error) {
	completedPages := 0
	for index, vol := range result.Volumes {
		outPath := filepath.Join(m.opts.output, cbz.VolumeFileName(m.name, vol.Number))
		if cause := ctx.Err(); cause != nil {
			return failedVolumes, m.stopAt(result.Volumes[index:], fmt.Sprintf("Interrupted before volume %v was written.", vol.Number), "", cause)
		}
		volume := m.plannedVolume(vol, outPath)
		if m.opts.dryRun {
			if m.opts.human {
				printfTo(m.opts.stdout, "would write %s (%d pages)\n", outPath, len(vol.Pages))
			}
		} else {
			events.emit("started", index, vol.Number, completedPages)
			copied := 0
			var onPage func(int)
			if m.opts.progress != nil {
				onPage = func(completed int) {
					copied = completed
					events.emit("advanced", index, vol.Number, completedPages+completed)
				}
			}
			if err := cbz.WriteWithProgress(ctx, outPath, vol.Pages, onPage); err != nil {
				if errors.Is(err, context.Canceled) {
					return failedVolumes, m.stopAt(result.Volumes[index:], fmt.Sprintf("Interrupted while writing volume %v; the unfinished file was removed.", vol.Number), absolutePath(outPath), err)
				}
				// Report it and bind the next volume: what can be bound is
				// bound, and the error returned at the end says that
				// something was not. The pages that were copied stay in the
				// progress count, which never goes back.
				wrapped := fmt.Errorf("writing volume %v: %w", vol.Number, err)
				failedVolumes++
				completedPages += copied
				m.report.Volumes = append(m.report.Volumes, volume)
				m.rep.fail("volume_write_failed", "write", fmt.Sprintf("Couldn't write volume %v.", vol.Number), wrapped, forVolume(&vol.Number), atPath(absolutePath(outPath)))
				m.rep.say("mangabind: " + wrapped.Error())
				continue
			}
			completedPages += len(vol.Pages)
			events.emit("completed", index, vol.Number, completedPages)
			volume.Written = true
			if m.opts.human && !m.opts.quiet {
				printfTo(m.opts.stdout, "wrote %s (%d pages)\n", outPath, len(vol.Pages))
			}
		}
		m.report.Volumes = append(m.report.Volumes, volume)
		m.summary.volumes++
		m.summary.pages += len(vol.Pages)
	}
	return failedVolumes, nil
}

// stopAt ends the run at the first of the volumes that are left: those volumes are in the report as planned
// and not written, as they would be after a failure, and the report says why.
func (m *mangaRun) stopAt(left []grouper.Volume, message, path string, cause error) error {
	for _, vol := range left {
		outPath := filepath.Join(m.opts.output, cbz.VolumeFileName(m.name, vol.Number))
		m.report.Volumes = append(m.report.Volumes, m.plannedVolume(vol, outPath))
	}
	return m.rep.interrupted("write", message, path, cause)
}
