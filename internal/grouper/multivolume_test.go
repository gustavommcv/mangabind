package grouper

import (
	"reflect"
	"testing"

	"github.com/gustavommcv/mangabind/internal/parser"
)

// TestGroupGapsAndConflictsAreScopedPerVolume proves a gap or conflict in
// one volume doesn't leak into, or get masked by, another volume's chapters
// - each volume's chapter numbers are independent of every other volume's.
func TestGroupGapsAndConflictsAreScopedPerVolume(t *testing.T) {
	ch := func(volume float64, chapter float64, path string) Chapter {
		v := volume
		return Chapter{
			Parsed: parser.ParsedChapter{Volume: &v, Chapter: chapter},
			Path:   path,
			Pages:  []string{"01.jpg"},
		}
	}

	chapters := []Chapter{
		// Volume 1: 1, 2, 4 - gap at 3, no conflict.
		ch(1, 1, "/v1c1"),
		ch(1, 2, "/v1c2"),
		ch(1, 4, "/v1c4"),
		// Volume 2: two chapter-1s (conflict) plus a chapter-2 - no gap.
		ch(2, 1, "/v2c1-a"),
		ch(2, 1, "/v2c1-b"),
		ch(2, 2, "/v2c2"),
	}

	result := Group(chapters, nil)

	wantGaps := []Gap{{Volume: 1, After: 2, Before: 4}}
	if !reflect.DeepEqual(result.Gaps, wantGaps) {
		t.Fatalf("Gaps = %+v, want %+v (volume 2 has no gap and must not appear)", result.Gaps, wantGaps)
	}

	if len(result.Conflicts) != 1 || result.Conflicts[0].Volume != 2 || result.Conflicts[0].Chapter != 1 {
		t.Fatalf("Conflicts = %+v, want exactly one conflict in volume 2 chapter 1", result.Conflicts)
	}

	if len(result.Volumes) != 2 {
		t.Fatalf("got %d volumes, want 2", len(result.Volumes))
	}
	// Volume 1: all 3 chapters survive (1, 2, 4 - none conflict).
	if result.Volumes[0].Number != 1 || len(result.Volumes[0].Pages) != 3 {
		t.Fatalf("volume 1 = %+v, want 3 pages", result.Volumes[0])
	}
	// Volume 2: only chapter 2 survives - both chapter-1 copies were excluded.
	if result.Volumes[1].Number != 2 || len(result.Volumes[1].Pages) != 1 {
		t.Fatalf("volume 2 = %+v, want 1 page (only chapter 2 survives the conflict)", result.Volumes[1])
	}
}
