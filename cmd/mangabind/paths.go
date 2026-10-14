package main

import (
	"os"
	"path/filepath"
)

// defaultOutputDir picks a sibling directory next to input, named after it,
// used when -output isn't given. A sibling rather than a subdirectory of
// input matters: writing inside the scanned input tree would make the next
// run see the output folder itself as an unrecognized chapter.
//
// The sibling is worked out from the folder's real name, not from the text
// that was typed. A trailing separator ("Manga/"), "." and ".." are all ways
// to name a folder that say nothing of what it is called: taken as typed,
// "Manga/" put the output inside Manga, and "." made a hidden folder named
// ". (mangabind)" holding files named ". - Vol.01.cbz". A path that already
// ends in a name is only cleaned, so that what is printed stays as the person
// wrote it; only one that does not is made absolute.
func defaultOutputDir(input string) string {
	dir, name := splitFolder(input)
	return filepath.Join(dir, name+" (mangabind)")
}

// folderName is what the manga folder is called, which is what its volumes are
// named after ("<name> - Vol.01.cbz"): the same name defaultOutputDir uses,
// and never "." or "..".
func folderName(input string) string {
	_, name := splitFolder(input)
	return name
}

// splitFolder separates the folder input names into where it is and what it is
// called. See defaultOutputDir for why the typed text is not enough.
func splitFolder(input string) (dir, name string) {
	path := filepath.Clean(input)
	if !namesAFolder(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	name = filepath.Base(path)
	if !namesAFolder(name) {
		// A filesystem root has no name to borrow.
		name = "manga"
	}
	return filepath.Dir(path), name
}

// namesAFolder reports whether the last element of path is a name, and not ".",
// ".." or a separator (what Base leaves of a root).
func namesAFolder(path string) bool {
	base := filepath.Base(path)
	if base == "." || base == ".." {
		return false
	}
	return len(base) != 1 || !os.IsPathSeparator(base[0])
}

// metadataConventionFile is the name Mangabind looks for inside a manga's
// own input folder when -metadata-file isn't given explicitly - see
// docs/adr/0010-local-metadata-file.md.
const metadataConventionFile = "mangabind.json"

// resolveMetadataFile picks which metadata file (if any) applies to input:
// an explicit override always wins; otherwise Mangabind looks for its
// naming convention right inside the manga's own folder. Returning "" means
// no metadata file applies - not an error, just nothing to load.
func resolveMetadataFile(input, explicit string) string {
	if explicit != "" {
		return explicit
	}
	candidate := filepath.Join(input, metadataConventionFile)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func absolutePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}
