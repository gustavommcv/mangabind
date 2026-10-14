package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

// reporter is how one manga tells what happened to it: to its report, for software, and, when the run is
// for people, as lines on stderr. An issue goes into the report through it, so that it names its manga, and
// an issue that is also said to the terminal is said by the same call, which is how the two stay alike.
type reporter struct {
	manga  string
	report *mangaReport
	human  bool
	stderr io.Writer
}

// issueField sets a field that only some issues have.
type issueField func(*machineIssue)

func atPath(path string) issueField {
	return func(value *machineIssue) { value.Path = path }
}

func forVolume(volume *float64) issueField {
	return func(value *machineIssue) { value.Volume = copyFloat(volume) }
}

// forChapter names the chapter an issue is about, and its volume when it has one.
func forChapter(volume *float64, chapter float64, special string) issueField {
	return func(value *machineIssue) {
		value.Volume = copyFloat(volume)
		value.Chapter = copyFloat(&chapter)
		value.Special = special
	}
}

func relatedTo(paths []string) issueField {
	return func(value *machineIssue) { value.RelatedPaths = paths }
}

func withDiagnostic(text string) issueField {
	return func(value *machineIssue) { value.Diagnostic = text }
}

func recoverable(value *machineIssue) { value.Recoverable = true }

// say prints a line for people on stderr; a run that answers with a report prints nothing.
func (r reporter) say(line string) {
	if r.human {
		printlnTo(r.stderr, line)
	}
}

func (r reporter) sayf(format string, args ...any) {
	if r.human {
		printlnTo(r.stderr, fmt.Sprintf(format, args...))
	}
}

func (r reporter) sayWarning(line string) {
	r.say("warning: " + line)
}

// record puts an issue of this manga into its report, and says nothing.
func (r reporter) record(severity, code, stage, message string, fields ...issueField) {
	value := issue(severity, code, stage, message)
	value.Manga = r.manga
	for _, field := range fields {
		field(&value)
	}
	r.report.addIssue(value)
}

// warn records a warning, and says it to people in the same words.
func (r reporter) warn(code, stage, message string, fields ...issueField) {
	r.sayWarning(message)
	r.record("warning", code, stage, message, fields...)
}

// fail records the error that ended the work of the manga, with its text as the diagnostic. It says nothing:
// the error goes up to runCLIContext, which says it.
func (r reporter) fail(code, stage, message string, cause error, fields ...issueField) {
	r.record("error", code, stage, message, append(fields, withDiagnostic(cause.Error()), recoverable)...)
}

// interrupted records that the run was stopped by a signal: as an issue in the report, and, for the
// terminal, as a line of its own. It returns the error that carries the interruption up to runCLIContext,
// which exits with 130. The caller finalizes the report and returns.
func (r reporter) interrupted(stage, message, path string, cause error) error {
	r.record("error", "interrupted", stage, message, atPath(path), recoverable)
	r.say("mangabind: " + message)
	return fmt.Errorf("interrupted: %w", cause)
}

// links tells of the links the scanner did not follow: a warning for
// each, naming the link and what is wrong with it, and nothing of it goes into
// a volume (docs/adr/0014-links-stay-inside-the-input.md). chapter is the
// chapter a page link was found in, and nil for a link found beside the
// chapters; the issue names its volume when it has one, so that a consumer
// can tell which volume lacks the page.
func (r reporter) links(links []scanner.Link, chapter *parser.ParsedChapter) {
	for _, link := range links {
		fields := []issueField{atPath(absolutePath(link.Path))}
		if chapter != nil {
			fields = append(fields, forChapter(chapter.Volume, chapter.Chapter, chapter.Special))
		}
		r.warn("link_skipped", "inspect", libraryLinkMessage(link), fields...)
	}
}

// skippedPages tells of the files of one chapter that were left out
// because they are not images: one warning for the chapter, naming a few of
// them, since a downloader that adds a credits page to every chapter would
// otherwise make a hundred lines of the same complaint. For a folder the issue
// lists every file in related_paths; for a .cbz chapter, where there is no path
// to give, the entries are in the diagnostic.
func (r reporter) skippedPages(chapter scanner.ChapterEntry, parsed *parser.ParsedChapter, skipped []string) {
	if len(skipped) == 0 {
		return
	}
	fields := []issueField{
		forChapter(parsed.Volume, parsed.Chapter, parsed.Special),
		atPath(absolutePath(chapter.Path)),
	}
	if chapter.IsArchive {
		fields = append(fields, withDiagnostic("entries: "+strings.Join(skipped, ", ")))
	} else {
		related := make([]string, len(skipped))
		for i, name := range skipped {
			related[i] = absolutePath(filepath.Join(chapter.Path, name))
		}
		fields = append(fields, relatedTo(related))
	}
	r.warn("unsupported_page_files", "inspect", skippedPagesMessage(chapter.Name, skipped), fields...)
}

// oversizedPages tells of the entries of one .cbz chapter that were left
// out because they declare they would expand to more than a page can be (see
// scanner.tooLarge): one warning for the chapter, naming a few and giving their
// sizes. The sizes are what makes the warning believable, so they are in the
// message as well as in the diagnostic.
func (r reporter) oversizedPages(chapter scanner.ChapterEntry, parsed *parser.ParsedChapter, oversized []scanner.Oversized) {
	if len(oversized) == 0 {
		return
	}
	details := make([]string, len(oversized))
	for i, entry := range oversized {
		details[i] = fmt.Sprintf("%s (%d bytes from %d)", entry.Name, entry.Size, entry.Compressed)
	}
	r.warn("page_entries_too_large", "inspect", oversizedPagesMessage(chapter.Name, oversized),
		forChapter(parsed.Volume, parsed.Chapter, parsed.Special),
		atPath(absolutePath(chapter.Path)),
		withDiagnostic("entries: "+strings.Join(details, ", ")),
	)
}
