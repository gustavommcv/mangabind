// Package scanner lists the chapter units and page files that make up a
// downloaded manga, without touching their content. A "chapter unit" is
// either a folder of loose image files (HakuNeko's default download
// format) or a .cbz file (HakuNeko's "Comic Book Archive" format) - both
// are treated as equally valid chapter sources.
package scanner

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gustavommcv/mangabind/internal/naturalsort"
)

// ChapterEntry is one chapter unit found directly under a manga's root
// directory: either a folder of images, or a .cbz archive.
type ChapterEntry struct {
	Name      string // name to parse for volume/chapter info - a .cbz's extension is stripped
	Path      string // absolute path to the folder or .cbz file
	IsArchive bool   // true when Path is a .cbz file rather than a folder
}

// Scan lists the immediate contents of root, classifying each entry as a
// chapter folder, a .cbz chapter archive, or - for anything else other than
// a few known junk files (.DS_Store, Thumbs.db, desktop.ini) - an
// unsupported file reported in skipped rather than silently dropped. This
// is what lets the caller warn about e.g. HakuNeko's .epub/.pdf download
// formats instead of producing no output at all. Results are sorted in
// natural order for stable, deterministic output; this is not yet
// volume/chapter order, which is the grouper package's job once names have
// been parsed.
func Scan(root string) (entries []ChapterEntry, skipped []string, err error) {
	rawEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}

	for _, e := range rawEntries {
		name := e.Name()

		if e.IsDir() {
			entries = append(entries, ChapterEntry{Name: name, Path: filepath.Join(root, name)})
			continue
		}

		if strings.EqualFold(filepath.Ext(name), ".cbz") {
			entries = append(entries, ChapterEntry{
				Name:      strings.TrimSuffix(name, filepath.Ext(name)),
				Path:      filepath.Join(root, name),
				IsArchive: true,
			})
			continue
		}

		if isIgnorableJunk(name) {
			continue
		}

		skipped = append(skipped, name)
	}

	sort.Slice(entries, func(i, j int) bool {
		return naturalsort.Less(entries[i].Name, entries[j].Name)
	})
	return entries, skipped, nil
}

func isIgnorableJunk(name string) bool {
	switch name {
	case ".DS_Store", "Thumbs.db", "desktop.ini":
		return true
	}
	return false
}

// Pages lists the page files directly inside a chapter folder, in natural
// order. Subdirectories are ignored.
func Pages(chapterDir string) ([]string, error) {
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		return nil, err
	}

	var pages []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		pages = append(pages, e.Name())
	}

	naturalsort.Strings(pages)
	return pages, nil
}

// PagesInArchive lists the page entries inside a .cbz chapter archive, in
// natural order. Directory entries within the archive are ignored.
func PagesInArchive(cbzPath string) ([]string, error) {
	zr, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	var pages []string
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		pages = append(pages, f.Name)
	}

	naturalsort.Strings(pages)
	return pages, nil
}
