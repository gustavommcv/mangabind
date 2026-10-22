package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gustavommcv/mangabind/internal/metadata"
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

// libraryLinkMessage says of a link that it was not followed: one found in a
// library folder, or one inside a manga.
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
		return fmt.Sprintf("found %q - .epub chapters aren't supported (only folders of images, .cbz or .zip), skipped", name)
	case ".pdf":
		return fmt.Sprintf("found %q - .pdf chapters aren't supported (only folders of images, .cbz or .zip), skipped", name)
	case ".rar", ".cbr":
		return fmt.Sprintf("found %q - .rar archives aren't supported (only folders of images, .cbz or .zip; extract it or save it as .zip), skipped", name)
	default:
		return fmt.Sprintf("found unrecognized file, skipped: %q", name)
	}
}
