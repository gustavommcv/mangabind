package grouper

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gustavommcv/mangabind/internal/parser"
)

// TestCombineVolumesNestsVolumeAboveChapter proves CombineVolumes wraps
// Group's own per-chapter directories in one more directory level per
// volume, and that two volumes each producing a chapter at the same
// position (both start at "c001") never collide once nested under their
// own, different volume directories - see
// docs/adr/0012-combine-series-into-one-volume.md.
func TestCombineVolumesNestsVolumeAboveChapter(t *testing.T) {
	ch := func(volume, chapter float64, path string) Chapter {
		v := volume
		return Chapter{
			Parsed: parser.ParsedChapter{Volume: &v, Chapter: chapter, Title: "Title"},
			Path:   path,
			Pages:  []string{"01.jpg"},
		}
	}

	chapters := []Chapter{
		ch(1, 1, "/v1c1"),
		ch(1, 2, "/v1c2"),
		ch(2, 1, "/v2c1"),
	}

	result := Group(chapters, nil)
	if len(result.Volumes) != 2 {
		t.Fatalf("got %d volumes, want 2", len(result.Volumes))
	}

	combined := CombineVolumes(result.Volumes)

	var names []string
	for _, p := range combined {
		names = append(names, p.ArchiveName)
	}
	want := []string{
		"v001 - Vol.01/c001 - Title/p0001.jpg",
		"v001 - Vol.01/c002 - Title/p0001.jpg",
		"v002 - Vol.02/c001 - Title/p0001.jpg",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("archive names = %v, want %v", names, want)
	}

	// Every source page still maps back to its real, distinct source path -
	// combining never merges or drops a page, only renames where it lands.
	wantFirst := filepath.Join("/v1c1", "01.jpg")
	wantThird := filepath.Join("/v2c1", "01.jpg")
	if combined[0].SourcePath != wantFirst || combined[2].SourcePath != wantThird {
		t.Fatalf("SourcePath not preserved: %+v", combined)
	}
}

// TestVolumeDirNameFractional proves a fractional volume number (e.g. a
// bonus "Vol.3.5") gets a directory name distinct from any whole-number
// volume, the same way chapterDirName already handles fractional chapters.
func TestVolumeDirNameFractional(t *testing.T) {
	if got := volumeDirName(1, 3.5); got != "v001 - Vol.3.5" {
		t.Fatalf("volumeDirName(1, 3.5) = %q, want %q", got, "v001 - Vol.3.5")
	}
	if got := volumeDirName(2, 4); got != "v002 - Vol.04" {
		t.Fatalf("volumeDirName(2, 4) = %q, want %q", got, "v002 - Vol.04")
	}
}
