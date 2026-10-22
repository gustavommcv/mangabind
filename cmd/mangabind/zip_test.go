package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeArchive makes the folder of the archive, which writeCBZ expects to be there, and the archive.
func writeArchive(t *testing.T, path string, entries ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCBZ(t, path, entries...)
}

// A collection scanned from the printed books keeps a volume as a .zip of its pages, often with a folder
// inside (docs/adr/0025-zip-archives-are-chapters-like-cbz.md). It is bound as a .cbz is, and reported as
// the kind of unit it is: an archive of images.
func TestAZipOfAVolumeIsBoundLikeACbz(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, "(一般コミック) [奥浩哉] いぬやしき 第01巻.zip"),
		"第01巻/001.jpg", "第01巻/002.jpg", "第01巻/Thumbs.db", "__MACOSX/第01巻/._001.jpg")
	writeArchive(t, filepath.Join(manga, "いぬやしき 第02巻.ZIP"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "いぬやしき 3巻.cbz"), "001.jpg", "002.jpg", "003.jpg")
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\nstderr: %s", code, stderr)
	}
	for _, unit := range report.Manga[0].Units {
		if unit.Kind != "cbz" || unit.Disposition == "unparsed" {
			t.Errorf("unit %q = kind %q, disposition %q, want an archive that was read", unit.Name, unit.Kind, unit.Disposition)
		}
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
}

// Chapters kept as .zip files are chapters like any other, and a .zip beside a .cbz of the same chapter is
// a conflict, as two .cbz would be.
func TestAZipChapterIsAChapterAndConflictsWithACbzOfTheSame(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, "Vol.01 Ch.001.zip"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Vol.01 Ch.002.zip"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Vol.01 Ch.002.cbz"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := runText("-input", manga, "-output", out, "-dry-run")

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "volume 1 chapter 2 has 2 conflicting sources") {
		t.Errorf("the .zip and the .cbz of one chapter are not a conflict:\n%s", stderr)
	}
	if !strings.Contains(stdout, filepath.Join(out, "Series - Vol.01.cbz")) {
		t.Errorf("stdout does not say the volume would be written:\n%s", stdout)
	}
}

// A .rar is not read, and it is said what can be done about it instead of just that it was not recognized.
func TestARarVolumeIsReportedWithWhatToDo(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFile(t, filepath.Join(manga, "第1巻.rar"), "not read")
	writeArchive(t, filepath.Join(manga, "第2巻.zip"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	_, _, stderr := runText("-input", manga, "-output", out, "-dry-run")

	if !strings.Contains(stderr, `found "第1巻.rar" - .rar archives aren't supported (only folders of images, .cbz or .zip; extract it or save it as .zip), skipped`) {
		t.Errorf("the .rar is not reported with what to do:\n%s", stderr)
	}
}
