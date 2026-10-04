// Package scanner lists the chapter units and page files that make up a
// manga folder, without touching their content. A "chapter unit" is
// either a folder of loose image files or a .cbz file - both
// are treated as equally valid chapter sources.
package scanner

import (
	"archive/zip"
	"io/fs"
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

// Link is a symbolic link the scanner did not follow, with the reason. A link
// is followed only when it leads to a regular file inside the input folder:
// the folder is what the person chose to bind, and a link in it that leads
// elsewhere would otherwise put a file they never chose into a book, which can
// then be saved, shared or uploaded - see docs/adr/0014-links-stay-inside-the-input.md.
type Link struct {
	Path string // the link itself, joined to the folder it was found in
	Why  string // what is wrong with it, as a short phrase for a message
}

const (
	whyOutside  = "leads outside the input folder"
	whyNotAFile = "does not lead to a file"
	whyNoInput  = "could not be checked against the input folder"
)

// Scan lists the immediate contents of root, classifying each entry as a
// chapter folder, a .cbz chapter archive, or - for anything else other than
// a few known junk files (.DS_Store, Thumbs.db, desktop.ini) - an
// unsupported file reported in skipped rather than silently dropped. This
// is what lets the caller warn about unsupported .epub/.pdf document
// formats instead of producing no output at all. Results are sorted in
// natural order for stable, deterministic output; this is not yet
// volume/chapter order, which is the grouper package's job once names have
// been parsed. A .cbz that is a link is listed only when it leads to a file
// inside root; one that does not is returned in links instead.
func Scan(root string) (entries []ChapterEntry, skipped []string, links []Link, err error) {
	rawEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, e := range rawEntries {
		name := e.Name()

		if e.IsDir() {
			entries = append(entries, ChapterEntry{Name: name, Path: filepath.Join(root, name)})
			continue
		}

		if strings.EqualFold(filepath.Ext(name), ".cbz") {
			if why := whyLinkIsUnsafe(root, root, e); why != "" {
				links = append(links, Link{Path: filepath.Join(root, name), Why: why})
				continue
			}
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
	return entries, skipped, links, nil
}

func isIgnorableJunk(name string) bool {
	switch name {
	case ".DS_Store", "Thumbs.db", "desktop.ini":
		return true
	case "mangabind.json":
		// Mangabind's own optional local chapter->volume metadata file (see
		// internal/metadata) - it lives right next to the chapters it
		// describes, not off in some other directory, so it must never be
		// mistaken for an unrecognized chapter file.
		return true
	}
	return false
}

// Pages lists the page files directly inside a chapter folder, in natural
// order. Subdirectories are ignored. A page that is a link is listed only when
// it leads to a file inside root, the manga's own folder; one that does not is
// returned in links and left out of pages.
func Pages(root, chapterDir string) (pages []string, links []Link, err error) {
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		return nil, nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if why := whyLinkIsUnsafe(root, chapterDir, e); why != "" {
			links = append(links, Link{Path: filepath.Join(chapterDir, e.Name()), Why: why})
			continue
		}
		pages = append(pages, e.Name())
	}

	naturalsort.Strings(pages)
	return pages, links, nil
}

// whyLinkIsUnsafe looks at one entry of dir, a folder inside root. It is empty
// for anything that is not a symbolic link, and for a link that leads to a
// regular file inside root once every link on the way is followed. Otherwise
// it says why the link is not to be followed: it leads outside root, it leads
// to something that is not a file (a folder, or nothing at all), or root
// itself could not be resolved to compare with.
//
// It is checked once, as the folder is read; a link swapped for another between
// then and the copy is out of reach of anything but not sharing the folder
// with whoever can write to it.
func whyLinkIsUnsafe(root, dir string, entry fs.DirEntry) string {
	if entry.Type()&fs.ModeSymlink == 0 {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(dir, entry.Name()))
	if err != nil {
		return whyNotAFile
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return whyNoInput
	}
	if !inside(realRoot, resolved) {
		return whyOutside
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return whyNotAFile
	}
	return ""
}

// inside reports whether path is root or lies below it. Both are real paths,
// with their links already resolved; filepath.Rel compares them the way the
// operating system does (case-insensitively on Windows).
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
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
