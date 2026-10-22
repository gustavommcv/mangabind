package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A folder named with the full-width digits and space a Japanese input method types is the same chapter
// as one named in ASCII (docs/adr/0020-full-width-digits-and-space-in-chapter-names.md): it is grouped
// into its volume, and two copies of one chapter in the two widths conflict, as two in ASCII do.
func TestAChapterNamedInFullWidthDigitsIsTheSameChapter(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{
		"Vol.０１ Ch.００１",
		"Vol.０１ Ch.００２",
		"Vol.０２　Ch.００３",
		"Vol.01 Ch.002",
	} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := runText("-input", manga, "-output", out, "-dry-run")

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	for _, volume := range []string{"Series - Vol.01.cbz", "Series - Vol.02.cbz"} {
		if !strings.Contains(stdout, filepath.Join(out, volume)) {
			t.Errorf("stdout does not say %s would be written:\n%s", volume, stdout)
		}
	}
	if !strings.Contains(stderr, "volume 1 chapter 2 has 2 conflicting sources") {
		t.Errorf("the chapter in both widths is not a conflict:\n%s", stderr)
	}
	if strings.Contains(stderr, "could not be read") || strings.Contains(stderr, "could not parse chapter name") {
		t.Errorf("a folder was not read:\n%s", stderr)
	}
}

// A folder named `第N話`, `N話` or `#N` is a chapter (docs/adr/0021-japanese-chapter-markers.md). These
// names carry no volume, so the mapping file supplies it, as it does for `Chapter 5`; the title after the
// number becomes the name of the chapter inside the book, as it was written.
func TestAChapterNamedInTheJapaneseWayIsGroupedAndKeepsItsTitle(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{"第1話 始まり", "2話　出会い", "#3"} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}
	writeFile(t, filepath.Join(manga, "mangabind.json"),
		`{"schema_version":1,"volumes":[{"number":"1","chapters":["1-3"]}]}`)
	out := filepath.Join(t.TempDir(), "out")

	code, _, stderr := runText("-input", manga, "-output", out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if strings.Contains(stderr, "could not parse chapter name") {
		t.Errorf("a folder was not read:\n%s", stderr)
	}
	want := []string{"c001 - 始まり/p0001.jpg", "c002 - 出会い/p0001.jpg", "c003/p0001.jpg"}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")); !reflect.DeepEqual(got, want) {
		t.Errorf("the volume holds %v, want %v", got, want)
	}
}

// Without a mapping the same names are chapters with no volume, so they are reported as such and not as
// names that could not be read.
func TestAChapterNamedInTheJapaneseWayWithoutAVolumeIsReportedAsUnassigned(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1話"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	_, _, stderr := runText("-input", manga, "-output", out, "-dry-run")

	if strings.Contains(stderr, "could not parse chapter name") {
		t.Errorf("the folder was not read:\n%s", stderr)
	}
	if !strings.Contains(stderr, `chapter has no volume number, skipped: "第1話"`) {
		t.Errorf("the chapter is not reported as lacking a volume:\n%s", stderr)
	}
}

// A folder that begins with its volume in the Japanese way, `第N巻` or `N巻`, and goes on with a chapter
// needs no mapping file (docs/adr/0022-japanese-volume-markers.md): the volume is in its name, and a
// chapter that has only the chapter marker still takes its volume from the mapping.
func TestAChapterNamedWithItsJapaneseVolumeNeedsNoMapping(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{"第1巻 第1話 始まり", "第1巻 第2話", "2巻 #3 旅立ち", "第2巻　第4話"} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}
	out := filepath.Join(t.TempDir(), "out")

	code, _, stderr := runText("-input", manga, "-output", out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if strings.Contains(stderr, "could not parse chapter name") || strings.Contains(stderr, "no volume number") {
		t.Errorf("a folder was not read or has no volume:\n%s", stderr)
	}
	for volume, want := range map[string][]string{
		"Series - Vol.01.cbz": {"c001 - 始まり/p0001.jpg", "c002/p0001.jpg"},
		"Series - Vol.02.cbz": {"c001 - 旅立ち/p0001.jpg", "c002/p0001.jpg"},
	} {
		if got := zipEntryNames(t, filepath.Join(out, volume)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s holds %v, want %v", volume, got, want)
		}
	}
}

// The volume in the name of a chapter wins over the mapping file, as it does for the other conventions.
func TestTheJapaneseVolumeInANameWinsOverTheMapping(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第2巻 第5話"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第6話"), "001.jpg")
	writeFile(t, filepath.Join(manga, "mangabind.json"),
		`{"schema_version":1,"volumes":[{"number":"1","chapters":["5-6"]}]}`)
	out := filepath.Join(t.TempDir(), "out")

	code, _, stderr := runText("-input", manga, "-output", out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if got, want := zipEntryNames(t, filepath.Join(out, "Series - Vol.02.cbz")), []string{"c001/p0001.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("volume 2 holds %v, want %v", got, want)
	}
	if got, want := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")), []string{"c001/p0001.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("volume 1 holds %v, want %v", got, want)
	}
}
