package grouper

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

func loadChapters(t *testing.T, root string) []Chapter {
	t.Helper()

	dirs, err := scanner.Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	reg := parser.DefaultRegistry()
	var chapters []Chapter
	for _, d := range dirs {
		parsed, _, ok := reg.Parse(d.Name)
		if !ok {
			t.Fatalf("unexpected unparsed folder in fixture: %q", d.Name)
		}
		pages, err := scanner.Pages(d.Path)
		if err != nil {
			t.Fatal(err)
		}
		chapters = append(chapters, Chapter{Parsed: parsed, Dir: d.Path, Pages: pages})
	}
	return chapters
}

func TestGroupDetectsGapAndConflict(t *testing.T) {
	root := "../../testdata/sample_manga_conflicts"
	chapters := loadChapters(t, root)

	result := Group(chapters, nil)

	// Ch.0003 is missing between Ch.0002 and Ch.0004.
	wantGaps := []Gap{{Volume: 1, After: 2, Before: 4}}
	if !reflect.DeepEqual(result.Gaps, wantGaps) {
		t.Fatalf("Gaps = %+v, want %+v", result.Gaps, wantGaps)
	}

	// Ch.0005 exists twice (Epsilon / Epsilon Redux, different groups).
	if len(result.Conflicts) != 1 {
		t.Fatalf("got %d conflicts, want 1: %+v", len(result.Conflicts), result.Conflicts)
	}
	c := result.Conflicts[0]
	if c.Volume != 1 || c.Chapter != 5 || c.Special != "" {
		t.Fatalf("conflict = %+v, want volume 1 chapter 5", c)
	}
	wantDirs := []string{
		filepath.Join(root, "Vol.01 Ch.0005 - Epsilon (en) [GroupX]"),
		filepath.Join(root, "Vol.01 Ch.0005 - Epsilon Redux (en) [GroupY]"),
	}
	if !reflect.DeepEqual(c.Dirs, wantDirs) {
		t.Fatalf("conflict dirs = %v, want %v", c.Dirs, wantDirs)
	}

	// The conflicting chapter 5 is excluded entirely: only 1, 2, 4 remain.
	if len(result.Volumes) != 1 {
		t.Fatalf("got %d volumes, want 1", len(result.Volumes))
	}
	if len(result.Volumes[0].Pages) != 3 {
		t.Fatalf("got %d pages, want 3 (chapters 1, 2, 4 only)", len(result.Volumes[0].Pages))
	}
}
