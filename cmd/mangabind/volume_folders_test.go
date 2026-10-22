package main

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func issueCodes(issues []machineIssue) []string {
	codes := make([]string, len(issues))
	for i, issue := range issues {
		codes[i] = issue.Code
	}
	return codes
}

// A folder named for a volume alone that holds chapter folders or archives gives those chapters its volume
// (docs/adr/0027-volume-folders-hold-chapters.md), so a chapter that is only "第1話" or "Chapter 5" has one,
// and the numbers of the chapters can start again in every volume.
func TestAVolumeFolderGivesItsVolumeToTheChaptersInIt(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1巻", "第1話 始まり"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第1巻", "第2話"), "001.jpg", "002.jpg")
	writeFiles(t, filepath.Join(manga, "第2巻", "第1話"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "第2巻", "第2話.zip"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "Volume 3", "001"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "Volume 3", "002"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Series - v04", "Chapter 5.cbz"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\nstderr: %s", code, stderr)
	}
	for volume, want := range map[string][]string{
		"Series - Vol.01.cbz": {"c001 - 始まり/p0001.jpg", "c002/p0001.jpg", "c002/p0002.jpg"},
		"Series - Vol.02.cbz": {"c001/p0001.jpg", "c002/p0001.jpg"},
		"Series - Vol.03.cbz": {"c001/p0001.jpg", "c002/p0001.jpg"},
		"Series - Vol.04.cbz": {"c001/p0001.jpg"},
	} {
		if got := zipEntryNames(t, filepath.Join(out, volume)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s holds %v, want %v", volume, got, want)
		}
	}
	units := report.Manga[0].Units
	if len(units) != 7 {
		t.Fatalf("the report has %d units, want the 7 chapters and not the volume folders: %+v", len(units), units)
	}
	want := map[string]struct {
		folder string
		volume float64
	}{
		filepath.Join(manga, "第1巻", "第1話 始まり"):                {"第1巻", 1},
		filepath.Join(manga, "第1巻", "第2話"):                    {"第1巻", 1},
		filepath.Join(manga, "第2巻", "第1話"):                    {"第2巻", 2},
		filepath.Join(manga, "第2巻", "第2話.zip"):                {"第2巻", 2},
		filepath.Join(manga, "Volume 3", "001"):               {"Volume 3", 3},
		filepath.Join(manga, "Volume 3", "002"):               {"Volume 3", 3},
		filepath.Join(manga, "Series - v04", "Chapter 5.cbz"): {"Series - v04", 4},
	}
	for _, unit := range units {
		w, ok := want[unit.Path]
		if !ok {
			t.Errorf("unexpected unit %q", unit.Path)
			continue
		}
		if unit.Disposition != "included" || unit.VolumeFolder != filepath.Join(manga, w.folder) {
			t.Errorf("unit %q = disposition %q in folder %q, want an included chapter in %q", unit.Path, unit.Disposition, unit.VolumeFolder, filepath.Join(manga, w.folder))
		}
		if unit.EffectiveVolume == nil || *unit.EffectiveVolume != w.volume {
			t.Errorf("unit %q: effective volume = %v, want %v", unit.Path, unit.EffectiveVolume, w.volume)
		}
		if unit.Parser.Volume != nil {
			t.Errorf("unit %q: the parser's own volume was rewritten to %v", unit.Path, *unit.Parser.Volume)
		}
	}
}

// A volume folder with pages in it is the volume itself (ADR 0024), whatever else it holds; one with nothing
// in it that could be a chapter, and a folder that is not named for a volume, are read as before.
func TestAFolderThatIsNotAVolumeFolderIsReadAsBefore(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1巻"), "001.jpg", "002.jpg")
	writeFiles(t, filepath.Join(manga, "第1巻", "第9話"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第2巻"))
	writeFiles(t, filepath.Join(manga, "Specials", "第1話"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	code, _, stderr := runText("-input", manga, "-output", out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if got, want := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")), []string{"c001/p0001.jpg", "c001/p0002.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("volume 1 holds %v, want its own pages %v", got, want)
	}
	for _, want := range []string{
		`chapter has no pages, skipped: "第2巻"`,
		`could not parse chapter name, skipped: "Specials"`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not say %s:\n%s", want, stderr)
		}
	}
}

// What is left out of a volume folder is named by its path below the manga folder, since two volume folders
// may hold chapters of one name.
func TestWhatIsLeftOutOfAVolumeFolderIsNamedByItsPath(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1巻", "第1話"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第1巻", "おまけ"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第2巻", "おまけ"), "001.jpg")
	writeFile(t, filepath.Join(manga, "第2巻", "notes.txt"), "x")
	writeFiles(t, filepath.Join(manga, "第2巻", "第1話"))
	out := filepath.Join(t.TempDir(), "out")

	_, _, stderr := runText("-input", manga, "-output", out, "-dry-run")

	// The messages quote the path as Go does, which doubles the separator of Windows.
	for _, want := range []string{
		"could not parse chapter name, skipped: " + strconv.Quote(filepath.Join("第1巻", "おまけ")),
		"could not parse chapter name, skipped: " + strconv.Quote(filepath.Join("第2巻", "おまけ")),
		"chapter has no pages, skipped: " + strconv.Quote(filepath.Join("第2巻", "第1話")),
		`found unrecognized file, skipped: "notes.txt"`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not say %s:\n%s", want, stderr)
		}
	}

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", out, "-json", "-dry-run")
	var unparsed, empty, files []string
	for _, issue := range report.Manga[0].Issues {
		switch issue.Code {
		case "unparsed_chapter":
			unparsed = append(unparsed, issue.Path)
		case "empty_chapter":
			empty = append(empty, issue.Path)
		case "unsupported_input_file":
			files = append(files, issue.Path)
		}
	}
	if want := []string{filepath.Join(manga, "第1巻", "おまけ"), filepath.Join(manga, "第2巻", "おまけ")}; !reflect.DeepEqual(unparsed, want) {
		t.Errorf("unparsed chapters = %v, want %v", unparsed, want)
	}
	if want := []string{filepath.Join(manga, "第2巻", "第1話")}; !reflect.DeepEqual(empty, want) {
		t.Errorf("empty chapters = %v, want %v", empty, want)
	}
	if want := []string{filepath.Join(manga, "第2巻", "notes.txt")}; !reflect.DeepEqual(files, want) {
		t.Errorf("unsupported files = %v, want %v", files, want)
	}
}

// A name that says a volume other than the folder it is in keeps the volume of its name, and says so.
func TestANameThatSaysAnotherVolumeThanItsFolderKeepsItsName(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1巻", "第1話"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第1巻", "Vol.02 Ch.005"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\nstderr: %s", code, stderr)
	}
	if !reflect.DeepEqual(issueCodes(report.Manga[0].Issues), []string{"volume_folder_conflict"}) {
		t.Errorf("issues = %v, want the one conflict", issueCodes(report.Manga[0].Issues))
	}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.02.cbz")); !reflect.DeepEqual(got, []string{"c001/p0001.jpg"}) {
		t.Errorf("volume 2 holds %v, want the chapter that says it", got)
	}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")); !reflect.DeepEqual(got, []string{"c001/p0001.jpg"}) {
		t.Errorf("volume 1 holds %v, want the chapter that says nothing else", got)
	}
}

// A chapter in a volume folder and one beside the folders that claim one chapter of one volume conflict, and
// both are said to be in conflict in the report, whichever kind of unit they are.
func TestChaptersOfOneVolumeInAFolderAndBesideItConflict(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1巻", "第1話"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "第1巻", "第2話"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Vol.01 Ch.002.cbz"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Vol.01 Ch.002.zip"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, _ := runMachineForTest(t, "-input", manga, "-output", out, "-json", "-dry-run")

	var conflicting, included []string
	for _, unit := range report.Manga[0].Units {
		switch unit.Disposition {
		case "conflict":
			conflicting = append(conflicting, unit.Path)
		case "included":
			included = append(included, unit.Name)
		}
	}
	if len(conflicting) != 3 || !reflect.DeepEqual(included, []string{"第1話"}) {
		t.Errorf("conflicting = %v, included = %v, want the three sources of chapter 2 in conflict and only chapter 1 included\nstderr: %s", conflicting, included, stderr)
	}
	var related []string
	for _, issue := range report.Manga[0].Issues {
		if issue.Code == "chapter_conflict" {
			related = issue.RelatedPaths
		}
	}
	if len(related) != 3 {
		t.Errorf("the conflict names %v, want its three sources", related)
	}
}

// The conflict is told in the words of the terminal with the path of a chapter in a volume folder, since its
// name alone would not say which one it is.
func TestAConflictNamesTheChapterOfAVolumeFolderByItsPath(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "第1巻", "第2話"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.002"), "001.jpg")
	out := filepath.Join(t.TempDir(), "out")

	_, _, stderr := runText("-input", manga, "-output", out, "-dry-run")

	for _, want := range []string{
		"volume 1 chapter 2 has 2 conflicting sources",
		strconv.Quote(filepath.Join("第1巻", "第2話")),
		`"Vol.01 Ch.002"`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not say %s:\n%s", want, stderr)
		}
	}
}
