// Package grouper buckets parsed chapters into volumes and flattens each
// volume into a collision-free, correctly ordered page list ready for the
// cbz writer.
package grouper

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/gustavommcv/Mangabind/internal/parser"
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
}

// Group buckets chapters by their explicit volume number, sorts chapters
// within each volume by chapter number (then special-chapter suffix), and
// flattens their pages into a single ordered, collision-free list per
// volume. Archive names use a "cNNN_pNNNN.ext" scheme, where N is the
// chapter's position within the volume (not its chapter number) - this
// guarantees no collision regardless of how the source folders numbered
// their pages.
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
	for _, v := range volNumbers {
		chs := byVolume[v]
		sort.Slice(chs, func(i, j int) bool {
			ci, cj := chs[i].Parsed, chs[j].Parsed
			if ci.Chapter != cj.Chapter {
				return ci.Chapter < cj.Chapter
			}
			return ci.Special < cj.Special
		})

		var pages []Page
		for ci, ch := range chs {
			for pi, name := range ch.Pages {
				pages = append(pages, Page{
					SourcePath:  filepath.Join(ch.Dir, name),
					ArchiveName: fmt.Sprintf("c%03d_p%04d%s", ci+1, pi+1, filepath.Ext(name)),
				})
			}
		}
		volumes = append(volumes, Volume{Number: v, Pages: pages})
	}

	return Result{Volumes: volumes, Unassigned: unassigned, Unparsed: unparsed}
}
