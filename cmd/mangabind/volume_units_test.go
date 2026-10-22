package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A folder or an archive named for a volume alone in the English way, as Kavita documents the name of a
// volume file (`Series Name v01.cbz`), is a volume of one chapter
// (docs/adr/0026-an-english-volume-named-unit-is-a-volume-of-one-chapter.md), the same as the Japanese
// `第N巻`. A volume followed by a title, or by a chapter, is not read.
func TestAUnitNamedForAnEnglishVolumeIsAVolumeOfOneChapter(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, "Series Name v01.cbz"), "001.jpg", "002.jpg")
	writeFiles(t, filepath.Join(manga, "Series Name - Volume 02 [Scans]"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Battle Royale, v03 (2000) [TokyoPop].zip"), "001.jpg", "002.jpg", "003.jpg")
	writeFiles(t, filepath.Join(manga, "Samurai Jack Vol. 04 - The threads of Time"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	code, _, stderr := runText("-input", manga, "-output", out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	for volume, want := range map[string][]string{
		"Series - Vol.01.cbz": {"c001/p0001.jpg", "c001/p0002.jpg"},
		"Series - Vol.02.cbz": {"c001/p0001.jpg"},
		"Series - Vol.03.cbz": {"c001/p0001.jpg", "c001/p0002.jpg", "c001/p0003.jpg"},
	} {
		if got := zipEntryNames(t, filepath.Join(out, volume)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s holds %v, want %v", volume, got, want)
		}
	}
	if !strings.Contains(stderr, `could not parse chapter name, skipped: "Samurai Jack Vol. 04 - The threads of Time"`) {
		t.Errorf("the volume with a title after it is not reported:\n%s", stderr)
	}
	if strings.Count(stderr, "could not parse chapter name") != 1 {
		t.Errorf("more than one folder was not read:\n%s", stderr)
	}
}
