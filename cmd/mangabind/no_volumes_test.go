package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// noVolumesManga is a series as it comes from the sites: some chapters say their volume, one is in a volume folder
// and two say none, so that with volumes the last ones would have none to be bound into.
func noVolumesManga(t *testing.T) string {
	t.Helper()
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{"Vol.01 Ch.001 - One", "Vol.01 Ch.002", "Vol.02 Ch.003 - Three", "Ch.004 (en)", "Ch.0005 (en)"} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}
	return manga
}

// -no-volumes binds every chapter into one book: no volume of the names is used, the chapters that name none are
// in it too, and the table of contents is the chapters in order (docs/adr/0030).
func TestNoVolumesBindsEveryChapterIntoOneBook(t *testing.T) {
	manga := noVolumesManga(t)
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json", "-no-volumes", "-chapter-labels", "number")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	want := []string{
		"Ch.1 - One/p0001.jpg", "Ch.2/p0001.jpg", "Ch.3 - Three/p0001.jpg", "Ch.4/p0001.jpg", "Ch.5/p0001.jpg",
	}
	if got := zipEntryNames(t, filepath.Join(out, "Series.cbz")); !reflect.DeepEqual(got, want) {
		t.Errorf("the book holds %v, want %v", got, want)
	}
	files, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("%d files were written, want the one book", len(files))
	}
	manga0 := report.Manga[0]
	if !manga0.WithoutVolumes {
		t.Error("the report does not say the book has no volumes")
	}
	if len(manga0.Volumes) != 1 || manga0.Volumes[0].Number != 1 || manga0.Volumes[0].OutputPath != absolutePath(filepath.Join(out, "Series.cbz")) || !manga0.Volumes[0].Written {
		t.Errorf("volumes = %+v, want the one book, written, in Series.cbz", manga0.Volumes)
	}
	if manga0.CombinedOutputPath != "" {
		t.Errorf("combined_output_path = %q, want none: the book is a volume's file", manga0.CombinedOutputPath)
	}
	for _, unit := range manga0.Units {
		if unit.Disposition != "included" || unit.EffectiveVolume == nil || *unit.EffectiveVolume != 1 {
			t.Errorf("unit %s is %s in volume %v, want included in the one volume", unit.Name, unit.Disposition, unit.EffectiveVolume)
		}
	}
	if len(manga0.Issues) != 0 {
		t.Errorf("issues = %v, want none: a chapter with no volume is not left out", issueCodes(manga0.Issues))
	}
}

// What the names say stays in the report, since it is what the person wrote, and the mapping file may not set
// a volume against it: there is no volume to disagree about.
func TestNoVolumesKeepsWhatTheNamesSayAndIgnoresTheMappingFilesVolumes(t *testing.T) {
	manga := noVolumesManga(t)
	writeFile(t, filepath.Join(manga, "mangabind.json"),
		`{"schema_version":1,"volumes":[{"number":"7","chapters":["1-3"]}]}`)

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-json", "-dry-run", "-no-volumes")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	for _, unit := range report.Manga[0].Units {
		if unit.MetadataAssignment.Status != "not_applicable" {
			t.Errorf("unit %s: metadata assignment %q, want not_applicable", unit.Name, unit.MetadataAssignment.Status)
		}
		if unit.Name == "Vol.02 Ch.003 - Three" && (unit.Parser.Volume == nil || *unit.Parser.Volume != 2) {
			t.Errorf("the volume the name says is %v, want 2", unit.Parser.Volume)
		}
	}
	if codes := issueCodes(report.Manga[0].Issues); len(codes) != 0 {
		t.Errorf("issues = %v, want none", codes)
	}
}

// A volume folder's chapters are chapters of the book as well, in the order of their numbers.
func TestNoVolumesTakesTheChaptersOfAVolumeFolder(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01", "Chapter 1"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "Vol.02", "Chapter 2"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	_, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json", "-no-volumes")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	want := []string{"c001/p0001.jpg", "c002/p0001.jpg"}
	if got := zipEntryNames(t, filepath.Join(out, "Series.cbz")); !reflect.DeepEqual(got, want) {
		t.Errorf("the book holds %v, want %v", got, want)
	}
}

// A chapter that two sources claim is left out, as in a volume, and a missing one is told: without a volume
// to name, the messages are about the book.
func TestNoVolumesTellsOfAGapAndOfAConflictOfTheBook(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{"Vol.01 Ch.001", "Vol.02 Ch.001", "Vol.02 Ch.004"} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}

	code, _, stderr := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-no-volumes")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	for _, want := range []string{"chapter 1 has 2 conflicting sources, all skipped", "the book is missing chapter(s) between 1 and 4"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not say %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "volume") {
		t.Errorf("a message names a volume of a book with none:\n%s", stderr)
	}
}

func TestNoVolumesTellsOfAGapAndOfAConflictInTheReportWithoutAVolume(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{"Vol.01 Ch.001", "Vol.02 Ch.001", "Vol.02 Ch.004"} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-json", "-dry-run", "-no-volumes")

	if got, want := issueCodes(report.Manga[0].Issues), []string{"chapter_gap", "chapter_conflict"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %v, want %v", got, want)
	}
	for _, issue := range report.Manga[0].Issues {
		if issue.Volume != nil {
			t.Errorf("issue %s names volume %v, want none", issue.Code, *issue.Volume)
		}
	}
}

// A run that only plans says where the book would be written.
func TestNoVolumesPlansTheBookByTheNameOfTheManga(t *testing.T) {
	manga := noVolumesManga(t)
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := runText("-input", manga, "-output", out, "-dry-run", "-no-volumes")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	if want := "would write " + filepath.Join(out, "Series.cbz") + " (5 pages)"; !strings.Contains(stdout, want) {
		t.Errorf("stdout does not say %q:\n%s", want, stdout)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a plan wrote the output folder")
	}
}

// In a library every manga is a book of its own.
func TestNoVolumesInALibraryMakesOneBookForEachManga(t *testing.T) {
	library := t.TempDir()
	writeFiles(t, filepath.Join(library, "Alpha", "Ch.1 (en)"), "001.jpg")
	writeFiles(t, filepath.Join(library, "Beta", "Vol.03 Ch.9"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	_, stderr, code := runMachineForTest(t, "-input", library, "-batch", "-output", out, "-json", "-no-volumes")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	for _, name := range []string{"Alpha.cbz", "Beta.cbz"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("%s was not written: %v", name, err)
		}
	}
}

// Without the flag a report does not carry the field, and the volumes are what they were.
func TestAReportOfABookWithVolumesDoesNotSayItHasNone(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), "001.jpg")
	var stdoutText string
	{
		code, stdout, stderr := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-json", "-dry-run")
		if code != 0 {
			t.Fatalf("exit code = %d\n%s", code, stderr)
		}
		stdoutText = stdout
	}
	if strings.Contains(stdoutText, "without_volumes") {
		t.Errorf("the report has without_volumes:\n%s", stdoutText)
	}
}

// A book with no volumes has none to nest in one file, so the two flags are refused, in the words of the mode
// that was asked for.
func TestNoVolumesCannotBeCombined(t *testing.T) {
	manga := noVolumesManga(t)

	code, _, stderr := runText("-input", manga, "-no-volumes", "-combine")
	if code != 2 || !strings.Contains(stderr, "-no-volumes can't be combined with -combine") {
		t.Errorf("text: exit code %d, stderr %q, want 2 and the refusal", code, stderr)
	}

	report, _, code := runMachineForTest(t, "-input", manga, "-json", "-no-volumes", "-combine")
	if code != 2 || !reflect.DeepEqual(issueCodes(report.Issues), []string{"no_volumes_with_combine"}) {
		t.Errorf("json: exit code %d, issues %v, want 2 and no_volumes_with_combine", code, issueCodes(report.Issues))
	}
}
