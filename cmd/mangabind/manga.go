package main

import (
	"context"
	"fmt"
)

// mangaSummary is what processManga reports back so runBatch can print a
// final tally without processManga needing to know it's running in a batch.
type mangaSummary struct {
	volumes int
	pages   int
}

// mangaRun is one manga being processed: what it was asked, the report it is filling in, and what it has
// written so far. The work is done in three stages, one file each: inspect (inspect.go) reads the chapters
// and groups them into volumes, write (write.go) binds the volumes, and reportFindings (findings.go) says
// what the grouping found that could not be used.
type mangaRun struct {
	input   string
	opts    runOptions
	name    string
	report  mangaReport
	rep     reporter
	summary mangaSummary
}

func newMangaRun(input string, opts runOptions) *mangaRun {
	name := folderName(input)
	m := &mangaRun{input: input, opts: opts, name: name, report: newMangaReport(name, absolutePath(input))}
	m.rep = reporter{manga: name, report: &m.report, human: opts.human, stderr: opts.stderr}
	return m
}

// processManga reads the chapters in input and writes its volumes as the options say. It gives the tally
// of what was written, the report of the manga, and the error that ended the run, if one did.
func processManga(ctx context.Context, input string, opts runOptions) (mangaSummary, mangaReport, error) {
	return newMangaRun(input, opts).run(ctx)
}

func (m *mangaRun) run(ctx context.Context) (mangaSummary, mangaReport, error) {
	m.progress(machineProgress{Stage: "inspect", State: "started", Manga: m.name})
	result, found, err := m.inspect(ctx)
	if err != nil || !found {
		return m.finish(err)
	}
	failedVolumes, err := m.write(ctx, result)
	if err != nil {
		return m.finish(err)
	}
	m.reportFindings(result)
	if failedVolumes > 0 {
		return m.finish(fmt.Errorf("%d of %d volume(s) could not be written", failedVolumes, len(result.Volumes)))
	}
	return m.finish(nil)
}

// finish closes the report with what was written and hands back what the run ends with.
func (m *mangaRun) finish(err error) (mangaSummary, mangaReport, error) {
	finalizeMangaReport(&m.report, m.summary)
	return m.summary, m.report, err
}

func (m *mangaRun) progress(event machineProgress) {
	if m.opts.progress != nil {
		m.opts.progress(event)
	}
}
