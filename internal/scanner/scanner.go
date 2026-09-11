// Package scanner lists the chapter folders and page files that make up a
// downloaded manga, without touching their content.
package scanner

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/gustavommcv/mangabind/internal/naturalsort"
)

// ChapterDir is one chapter folder found directly under a manga's root
// directory.
type ChapterDir struct {
	Name string // folder name, e.g. "Vol.01 Ch.0001 - Title (en) [Group]"
	Path string // absolute path to the folder
}

// Scan lists the immediate subdirectories of root, treating each one as a
// chapter folder. Files directly under root are ignored. Results are
// sorted in natural order for stable, deterministic output; this is not
// yet volume/chapter order, which is the grouper package's job once names
// have been parsed.
func Scan(root string) ([]ChapterDir, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var dirs []ChapterDir
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirs = append(dirs, ChapterDir{
			Name: e.Name(),
			Path: filepath.Join(root, e.Name()),
		})
	}

	sort.Slice(dirs, func(i, j int) bool {
		return naturalsort.Less(dirs[i].Name, dirs[j].Name)
	})
	return dirs, nil
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
