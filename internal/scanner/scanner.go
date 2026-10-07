// Package scanner lists the chapter units and page files that make up a
// manga folder, without touching their content. A "chapter unit" is
// either a folder of loose image files or a .cbz file - both
// are treated as equally valid chapter sources.
package scanner

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path"
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
	whyOutside     = "leads outside the input folder"
	whyNotAFile    = "does not lead to a file"
	whyNoInput     = "could not be checked against the input folder"
	whyLibraryLink = "leads to a folder, and the manga folders of a library are not followed through links"
	whyNowhere     = "leads nowhere"
)

// Library lists the manga folders directly inside a library folder, which is
// what -batch processes, in the order os.ReadDir gives (by name). Only a real
// folder is a manga.
//
// A symbolic link is not followed here, for the reason ADR 0014 gives for the
// pages of one manga: a library that someone else made could otherwise put any
// folder of the reader's disk into a volume. A link that leads to a folder, or
// leads nowhere, is returned in links so that it can be reported instead of
// vanishing from the run without a word; a link to a file is as a file is: not
// a manga, and not worth a word.
func Library(root string) (manga []string, links []Link, err error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink != 0 {
			path := filepath.Join(root, e.Name())
			info, statErr := os.Stat(path)
			switch {
			case statErr != nil:
				links = append(links, Link{Path: path, Why: whyNowhere})
			case info.IsDir():
				links = append(links, Link{Path: path, Why: whyLibraryLink})
			}
			continue
		}
		if e.IsDir() {
			manga = append(manga, e.Name())
		}
	}
	return manga, links, nil
}

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
//
// Only images are pages (see isPage): a file that is not one is left out too,
// and named in skipped, except the junk that operating systems and download
// tools leave in every folder (.DS_Store, Thumbs.db, ComicInfo.xml and the
// like), which is left out without a word.
func Pages(root, chapterDir string) (pages, skipped []string, links []Link, err error) {
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if why := whyLinkIsUnsafe(root, chapterDir, e); why != "" {
			links = append(links, Link{Path: filepath.Join(chapterDir, e.Name()), Why: why})
			continue
		}
		switch classifyPage(e.Name()) {
		case pageImage:
			pages = append(pages, e.Name())
		case pageOther:
			skipped = append(skipped, e.Name())
		}
	}

	naturalsort.Strings(pages)
	naturalsort.Strings(skipped)
	return pages, skipped, links, nil
}

// pageClass is what a file found where pages are expected turns out to be.
type pageClass int

const (
	// pageImage is a file a comic reader opens as a page.
	pageImage pageClass = iota
	// pageJunk is a file that is never a page and never worth a word: the
	// leftovers of a file manager, an archiver or a downloader.
	pageJunk
	// pageOther is any other file: not a page, and something the person may
	// want to know was left out.
	pageOther
)

// imageExtensions are the extensions of the files that are pages, compared
// without regard to case. mangabind goes by the name and never opens a file, so
// this is the whole definition of a page; it is deliberately wider than what
// mangapress reads (jpg, jpeg, png, gif, bmp, webp), since a .cbz is for any
// comic reader, and a reader that opens avif, jxl or tiff should find them.
var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".bmp": true, ".avif": true, ".jxl": true, ".tif": true, ".tiff": true,
}

// classifyPage says what a file is, by its name alone. name is a file name, or
// for an archive entry its whole path inside the archive, written with either
// separator: archives made on Windows often use backslashes.
func classifyPage(name string) pageClass {
	parts := strings.Split(strings.ReplaceAll(name, "\\", "/"), "/")
	base := parts[len(parts)-1]
	for _, dir := range parts[:len(parts)-1] {
		if dir == "__MACOSX" {
			return pageJunk // macOS's parallel tree of resource-fork sidecars
		}
	}
	if strings.HasPrefix(base, "._") {
		return pageJunk // a sidecar, named after the file it belongs to, extension and all
	}
	switch strings.ToLower(base) {
	case ".ds_store", "thumbs.db", "desktop.ini", "comicinfo.xml":
		return pageJunk
	}
	if imageExtensions[strings.ToLower(path.Ext(base))] {
		return pageImage
	}
	return pageOther
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

// Oversized is an image entry of an archive that was left out for what it
// declares it would expand to: see tooLarge.
type Oversized struct {
	Name       string
	Size       uint64 // uncompressed
	Compressed uint64
}

const (
	// maxEntrySize is the most an archive entry may expand to. A page of a
	// comic is a few megabytes, and a scan at print resolution a few tens of
	// them; nothing is a quarter of a gigabyte.
	maxEntrySize = 256 << 20

	// maxRatio is the most an entry may expand from what it is stored as, once
	// it is bigger than ratioFloor. Real images do not compress a thousand to
	// one (the best a deflate stream can do is a little over that, which is what
	// an archive made to fill a disk uses); a small entry that does, a blank
	// bitmap, is below the floor and harmless.
	maxRatio   = 1000
	ratioFloor = 16 << 20
)

// tooLarge says whether an entry is bigger than a page can be, going by the
// sizes its own header declares. mangabind copies pages without looking at
// them, and a volume is stored, not compressed, so an entry that expands to
// gigabytes from a few hundred kilobytes would be written out in full (a 400 KB
// archive became a 419 MB volume). The declared size is the real one as far as
// it matters: archive/zip refuses to read more from an entry than it declares.
func tooLarge(f *zip.File) bool {
	size, packed := f.UncompressedSize64, f.CompressedSize64
	if size > maxEntrySize {
		return true
	}
	return size > ratioFloor && size/max(packed, 1) > maxRatio
}

// IsPageEntry reports whether PagesInArchive lists f as a page: an image by its
// name and not too large. The writer asks the same question, so that when an
// archive has two entries of one name it reads the one that was listed.
func IsPageEntry(f *zip.File) bool {
	return !f.FileInfo().IsDir() && classifyPage(f.Name) == pageImage && !tooLarge(f)
}

// PagesInArchive lists the page entries inside a .cbz chapter archive, in
// natural order. Directory entries within the archive are ignored. As in
// Pages, only images are pages: the other entries are named in skipped, apart
// from the same junk (macOS adds a whole __MACOSX folder of it to the archives
// it makes). An image entry that declares it would expand to more than a page
// can be is neither: it is returned in oversized, and not copied.
func PagesInArchive(cbzPath string) (pages, skipped []string, oversized []Oversized, err error) {
	zr, err := zip.OpenReader(cbzPath)
	// A name that is not local ("../a.jpg") makes OpenReader report
	// ErrInsecurePath when GODEBUG asks for it, and still return the archive.
	// The names are only used to find entries, never to make files.
	if err != nil && (!errors.Is(err, zip.ErrInsecurePath) || zr == nil) {
		return nil, nil, nil, err
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		switch classifyPage(f.Name) {
		case pageImage:
			if tooLarge(f) {
				oversized = append(oversized, Oversized{Name: f.Name, Size: f.UncompressedSize64, Compressed: f.CompressedSize64})
				continue
			}
			pages = append(pages, f.Name)
		case pageOther:
			skipped = append(skipped, f.Name)
		}
	}

	naturalsort.Strings(pages)
	naturalsort.Strings(skipped)
	sort.Slice(oversized, func(i, j int) bool { return naturalsort.Less(oversized[i].Name, oversized[j].Name) })
	return pages, skipped, oversized, nil
}
