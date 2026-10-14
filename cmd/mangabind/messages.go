package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gustavommcv/mangabind/internal/metadata"
	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

// maxIndividualWarnings caps how many names of the same warning category get
// printed one by one before summarizeNames collapses the rest into a single
// line - without it, a manga with hundreds of chapters and no volume
// information anywhere would print one warning per chapter.
const maxIndividualWarnings = 10

// summarizeNames turns a list of same-category names into warning lines:
// each one individually if there are few enough, otherwise the first
// maxIndividualWarnings plus one line summarizing the rest (with an
// optional hint pointing at the fix).
func summarizeNames(label string, names []string, hint string) []string {
	if len(names) == 0 {
		return nil
	}
	if len(names) <= maxIndividualWarnings {
		lines := make([]string, len(names))
		for i, name := range names {
			lines[i] = fmt.Sprintf("%s: %q", label, name)
		}
		return lines
	}

	lines := make([]string, 0, maxIndividualWarnings+1)
	for _, name := range names[:maxIndividualWarnings] {
		lines = append(lines, fmt.Sprintf("%s: %q", label, name))
	}
	rest := fmt.Sprintf("... and %d more", len(names)-maxIndividualWarnings)
	if hint != "" {
		rest += " (" + hint + ")"
	}
	return append(lines, rest)
}

// libraryLinkMessage says of a link found in a library folder that it was not
// followed, in the words reportLinks uses for the links inside a manga.
func libraryLinkMessage(link scanner.Link) string {
	return fmt.Sprintf("found a link %q that %s, skipped", filepath.Base(link.Path), link.Why)
}

// duplicateChaptersMessage says that the metadata file lists chapters under
// more than one volume, naming the first as an example: one warning for the
// file, since overlapping ranges make a dozen of them at once.
func duplicateChaptersMessage(duplicates []metadata.Duplicate) string {
	first := duplicates[0]
	chapter := fmt.Sprintf("%v%s", first.Chapter, first.Special)
	if len(duplicates) == 1 {
		return fmt.Sprintf("the metadata file lists chapter %s under volumes %v and %v; the last one, volume %v, is used", chapter, first.Earlier, first.Later, first.Later)
	}
	return fmt.Sprintf("the metadata file lists %d chapters under more than one volume (the first is chapter %s, under volumes %v and %v); the last listing of each is used", len(duplicates), chapter, first.Earlier, first.Later)
}

// reportLinks tells of the links the scanner did not follow: a warning for
// each, naming the link and what is wrong with it, and nothing of it goes into
// a volume (docs/adr/0014-links-stay-inside-the-input.md). chapter is the
// chapter a page link was found in, and nil for a link found beside the
// chapters; the issue names its volume when it has one, so that a consumer
// can tell which volume lacks the page.
func reportLinks(report *mangaReport, mangaName string, links []scanner.Link, chapter *parser.ParsedChapter, human bool, stderr io.Writer) {
	for _, link := range links {
		message := fmt.Sprintf("found a link %q that %s, skipped", filepath.Base(link.Path), link.Why)
		if human {
			printlnTo(stderr, "warning:", message)
		}
		value := issue("warning", "link_skipped", "inspect", message)
		value.Manga = mangaName
		value.Path = absolutePath(link.Path)
		if chapter != nil {
			value.Volume = copyFloat(chapter.Volume)
			value.Chapter = copyFloat(&chapter.Chapter)
			value.Special = chapter.Special
		}
		report.addIssue(value)
	}
}

// reportSkippedPages tells of the files of one chapter that were left out
// because they are not images: one warning for the chapter, naming a few of
// them, since a downloader that adds a credits page to every chapter would
// otherwise make a hundred lines of the same complaint. For a folder the issue
// lists every file in related_paths; for a .cbz chapter, where there is no path
// to give, the entries are in the diagnostic.
func reportSkippedPages(report *mangaReport, mangaName string, chapter scanner.ChapterEntry, parsed *parser.ParsedChapter, skipped []string, human bool, stderr io.Writer) {
	if len(skipped) == 0 {
		return
	}
	message := skippedPagesMessage(chapter.Name, skipped)
	if human {
		printlnTo(stderr, "warning:", message)
	}
	value := issue("warning", "unsupported_page_files", "inspect", message)
	value.Manga = mangaName
	value.Volume = copyFloat(parsed.Volume)
	value.Chapter = copyFloat(&parsed.Chapter)
	value.Special = parsed.Special
	value.Path = absolutePath(chapter.Path)
	if chapter.IsArchive {
		value.Diagnostic = "entries: " + strings.Join(skipped, ", ")
	} else {
		for _, name := range skipped {
			value.RelatedPaths = append(value.RelatedPaths, absolutePath(filepath.Join(chapter.Path, name)))
		}
	}
	report.addIssue(value)
}

// reportOversizedPages tells of the entries of one .cbz chapter that were left
// out because they declare they would expand to more than a page can be (see
// scanner.tooLarge): one warning for the chapter, naming a few and giving their
// sizes. The sizes are what makes the warning believable, so they are in the
// message as well as in the diagnostic.
func reportOversizedPages(report *mangaReport, mangaName string, chapter scanner.ChapterEntry, parsed *parser.ParsedChapter, oversized []scanner.Oversized, human bool, stderr io.Writer) {
	if len(oversized) == 0 {
		return
	}
	message := oversizedPagesMessage(chapter.Name, oversized)
	if human {
		printlnTo(stderr, "warning:", message)
	}
	value := issue("warning", "page_entries_too_large", "inspect", message)
	value.Manga = mangaName
	value.Volume = copyFloat(parsed.Volume)
	value.Chapter = copyFloat(&parsed.Chapter)
	value.Special = parsed.Special
	value.Path = absolutePath(chapter.Path)
	details := make([]string, len(oversized))
	for i, entry := range oversized {
		details[i] = fmt.Sprintf("%s (%d bytes from %d)", entry.Name, entry.Size, entry.Compressed)
	}
	value.Diagnostic = "entries: " + strings.Join(details, ", ")
	report.addIssue(value)
}

func oversizedPagesMessage(chapter string, oversized []scanner.Oversized) string {
	named := make([]string, 0, maxNamedSkippedPages)
	for _, entry := range oversized[:min(len(oversized), maxNamedSkippedPages)] {
		named = append(named, fmt.Sprintf("%q (%s, from %s)", entry.Name, formatSize(entry.Size), formatSize(entry.Compressed)))
	}
	list := strings.Join(named, ", ")
	if more := len(oversized) - len(named); more > 0 {
		list += fmt.Sprintf(" and %d more", more)
	}
	if len(oversized) == 1 {
		return fmt.Sprintf("chapter %q: skipped 1 entry that expands to far more than a page can be: %s", chapter, list)
	}
	return fmt.Sprintf("chapter %q: skipped %d entries that expand to far more than a page can be: %s", chapter, len(oversized), list)
}

// formatSize writes a number of bytes with the unit a person reads it in.
func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value, suffix := float64(bytes), "B"
	for _, next := range []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"} {
		if value < unit {
			break
		}
		value /= unit
		suffix = next
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

// maxNamedSkippedPages is how many skipped files a message names before it says
// how many more there are.
const maxNamedSkippedPages = 3

func skippedPagesMessage(chapter string, skipped []string) string {
	named := make([]string, 0, maxNamedSkippedPages)
	for _, name := range skipped[:min(len(skipped), maxNamedSkippedPages)] {
		named = append(named, fmt.Sprintf("%q", name))
	}
	list := strings.Join(named, ", ")
	if more := len(skipped) - len(named); more > 0 {
		list += fmt.Sprintf(" and %d more", more)
	}
	if len(skipped) == 1 {
		return fmt.Sprintf("chapter %q: skipped 1 file that is not an image: %s", chapter, list)
	}
	return fmt.Sprintf("chapter %q: skipped %d files that are not images: %s", chapter, len(skipped), list)
}

// unsupportedFileMessage explains why a file found alongside chapter
// folders/.cbz files wasn't treated as a chapter - naming the two
// download formats Mangabind doesn't support explicitly, so the user knows
// this isn't a bug, rather than getting no feedback at all.
func unsupportedFileMessage(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".epub":
		return fmt.Sprintf("found %q - .epub chapters aren't supported (only folders of images or .cbz), skipped", name)
	case ".pdf":
		return fmt.Sprintf("found %q - .pdf chapters aren't supported (only folders of images or .cbz), skipped", name)
	default:
		return fmt.Sprintf("found unrecognized file, skipped: %q", name)
	}
}
