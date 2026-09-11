// Package grouper buckets parsed chapters into volumes and flattens each
// volume into a collision-free, correctly ordered page list ready for the
// cbz writer.
package grouper

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gustavommcv/mangabind/internal/parser"
)

// Chapter is a parsed chapter folder together with its page files, ready to
// be grouped into a Volume.
type Chapter struct {
	Parsed parser.ParsedChapter
	Dir    string   // absolute path to the chapter folder
	Pages  []string // page filenames, in natural order (see scanner.Pages)
}

// Page is one page ready to be written into a volume's .cbz: where to read
// it from, and the collision-free name it gets inside the archive.
type Page struct {
	SourcePath  string
	ArchiveName string
}

// Volume is one volume's worth of pages, already in final reading order.
type Volume struct {
	Number float64
	Pages  []Page
}

// Gap flags a missing whole-number chapter between two chapters that are
// present, e.g. Ch.0004 immediately followed by Ch.0006 (Ch.0005 missing).
// Fractional/special chapters (interludes, bonus x/y/z chapters) are not
// expected to be strictly sequential, so they never produce a Gap.
type Gap struct {
	Volume float64
	After  float64 // last chapter number seen before the gap
	Before float64 // next chapter number seen after the gap
}

// Conflict flags two or more chapter folders that claim the same volume,
// chapter number, and special suffix - typically the same chapter
// downloaded from two different scan groups. Mangabind has no way to know
// which source is "correct", so none of the conflicting chapters are
// included in the volume; see docs/adr/0005-drop-cover-detection.md for the
// same reasoning applied to a different problem (don't guess).
type Conflict struct {
	Volume  float64
	Chapter float64
	Special string
	Dirs    []string // the conflicting chapter folders, excluded from output
}

// Result is the outcome of grouping a manga's chapters into volumes.
type Result struct {
	Volumes []Volume

	// Unassigned holds chapters that parsed fine but carry no volume number.
	// We don't infer a missing volume from neighboring chapters (see
	// docs/adr/0003-pluggable-chapter-parsing.md), so these are reported
	// rather than grouped.
	Unassigned []Chapter

	// Unparsed holds chapter folder names no registered parser recognized.
	Unparsed []string

	// Gaps holds missing-chapter warnings; see Gap.
	Gaps []Gap

	// Conflicts holds duplicate-chapter warnings; see Conflict. The
	// conflicting chapters are excluded from Volumes, not guessed into it.
	Conflicts []Conflict
}

type chapterKey struct {
	Chapter float64
	Special string
}

// Group buckets chapters by their explicit volume number, resolves any
// same-chapter conflicts (see Conflict), sorts the remaining chapters within
// each volume by chapter number then special-chapter suffix, and flattens
// their pages into a single ordered, collision-free list per volume.
//
// Each chapter gets its own directory inside the archive, named
// "cNNN - Title" (N is the chapter's position within the volume, not its
// chapter number - this guarantees no collision regardless of how the
// source folders numbered their pages). This isn't just cosmetic: KCC
// builds its EPUB table of contents from top-level subdirectories when
// converting a comic archive, using each subdirectory's name as the
// chapter title - a flat archive produces a single, useless TOC entry for
// the whole volume. See docs/adr/0006-chapter-directories-for-kcc-toc.md.
func Group(chapters []Chapter, unparsed []string) Result {
	byVolume := map[float64][]Chapter{}
	var unassigned []Chapter

	for _, c := range chapters {
		if c.Parsed.Volume == nil {
			unassigned = append(unassigned, c)
			continue
		}
		v := *c.Parsed.Volume
		byVolume[v] = append(byVolume[v], c)
	}

	var volNumbers []float64
	for v := range byVolume {
		volNumbers = append(volNumbers, v)
	}
	sort.Float64s(volNumbers)

	var volumes []Volume
	var allGaps []Gap
	var allConflicts []Conflict

	for _, v := range volNumbers {
		chs := byVolume[v]

		allGaps = append(allGaps, detectGaps(v, chs)...)

		resolved, conflicts := resolveConflicts(v, chs)
		allConflicts = append(allConflicts, conflicts...)

		sort.Slice(resolved, func(i, j int) bool {
			ci, cj := resolved[i].Parsed, resolved[j].Parsed
			if ci.Chapter != cj.Chapter {
				return ci.Chapter < cj.Chapter
			}
			return ci.Special < cj.Special
		})

		var pages []Page
		for ci, ch := range resolved {
			chapterDir := chapterDirName(ci+1, ch.Parsed.Title)
			for pi, name := range ch.Pages {
				pages = append(pages, Page{
					SourcePath:  filepath.Join(ch.Dir, name),
					ArchiveName: fmt.Sprintf("%s/p%04d%s", chapterDir, pi+1, filepath.Ext(name)),
				})
			}
		}
		volumes = append(volumes, Volume{Number: v, Pages: pages})
	}

	return Result{
		Volumes:    volumes,
		Unassigned: unassigned,
		Unparsed:   unparsed,
		Gaps:       allGaps,
		Conflicts:  allConflicts,
	}
}

// detectGaps looks for missing whole-number chapters in chs. It considers
// every chapter present in the source (including ones that will later be
// excluded as conflicts - a duplicated chapter is still "present", just
// ambiguous), so it never overlaps with a Conflict for the same chapter.
func detectGaps(volume float64, chs []Chapter) []Gap {
	seen := map[int]bool{}
	for _, c := range chs {
		if c.Parsed.Chapter == math.Trunc(c.Parsed.Chapter) {
			seen[int(c.Parsed.Chapter)] = true
		}
	}

	var nums []int
	for n := range seen {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	var gaps []Gap
	for i := 0; i < len(nums)-1; i++ {
		if nums[i+1]-nums[i] > 1 {
			gaps = append(gaps, Gap{Volume: volume, After: float64(nums[i]), Before: float64(nums[i+1])})
		}
	}
	return gaps
}

// chapterDirName builds the in-archive directory name for the chapter at
// the given position within its volume. The position prefix guarantees
// correct ordering and uniqueness even if two chapters share a title; the
// title (when present) becomes the chapter's title in KCC's generated TOC.
func chapterDirName(position int, title string) string {
	base := fmt.Sprintf("c%03d", position)
	title = sanitizeFolderName(title)
	if title == "" {
		return base
	}
	return base + " - " + title
}

// sanitizeFolderName strips characters that would break a single-level
// directory name inside the archive - a slash in a chapter title would
// otherwise silently create unintended nested directories.
func sanitizeFolderName(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return strings.TrimSpace(s)
}

// resolveConflicts splits chs into chapters with a unique (Chapter, Special)
// key and a Conflict for every key claimed by more than one folder. Order is
// preserved for the non-conflicting chapters.
func resolveConflicts(volume float64, chs []Chapter) (resolved []Chapter, conflicts []Conflict) {
	byKey := map[chapterKey][]Chapter{}
	for _, c := range chs {
		k := chapterKey{c.Parsed.Chapter, c.Parsed.Special}
		byKey[k] = append(byKey[k], c)
	}

	for _, c := range chs {
		k := chapterKey{c.Parsed.Chapter, c.Parsed.Special}
		if len(byKey[k]) == 1 {
			resolved = append(resolved, c)
		}
	}

	var keys []chapterKey
	for k, group := range byKey {
		if len(group) > 1 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Chapter != keys[j].Chapter {
			return keys[i].Chapter < keys[j].Chapter
		}
		return keys[i].Special < keys[j].Special
	})

	for _, k := range keys {
		var dirs []string
		for _, c := range byKey[k] {
			dirs = append(dirs, c.Dir)
		}
		conflicts = append(conflicts, Conflict{Volume: volume, Chapter: k.Chapter, Special: k.Special, Dirs: dirs})
	}

	return resolved, conflicts
}
