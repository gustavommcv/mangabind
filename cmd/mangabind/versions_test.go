package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The same chapter downloaded from two scan groups is two files that claim one chapter. These tests use the
// shape of a real folder of a series that has no volumes, with two versions of one of its chapters
// (docs/adr/0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md).

const (
	versionsFive  = "Ch.0005 - Five (en).cbz"
	versionsSixA  = "Ch.0006 - Six (en) [GroupA].cbz"
	versionsSixB  = "Ch.0006 - Six (en) [GroupB].cbz"
	versionsSeven = "Ch.0007 (en).cbz"
)

// versionsManga makes the folder: the two versions of chapter 6 have two and three pages.
func versionsManga(t *testing.T) string {
	t.Helper()
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, versionsFive), "001.jpg")
	writeArchive(t, filepath.Join(manga, versionsSixA), "001.jpg", "002.jpg")
	writeArchive(t, filepath.Join(manga, versionsSixB), "001.jpg", "002.jpg", "003.jpg")
	writeArchive(t, filepath.Join(manga, versionsSeven), "001.jpg")
	return manga
}

func writeMapping(t *testing.T, manga, versions string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mapping.json")
	content := `{"schema_version":1,"volumes":[{"number":"1","chapters":["5-7"]}]`
	if versions != "" {
		content += `,"chapter_versions":` + versions
	}
	writeFile(t, path, content+"}")
	return path
}

func unitNamed(t *testing.T, units []unitReport, name string) unitReport {
	t.Helper()
	for _, unit := range units {
		if unit.Name+".cbz" == name {
			return unit
		}
	}
	t.Fatalf("no unit for %q", name)
	return unitReport{}
}

func dispositions(units []unitReport) map[string]string {
	got := map[string]string{}
	for _, unit := range units {
		got[unit.Name+".cbz"] = unit.Disposition
	}
	return got
}

// With no volume the files are not a conflict, since a conflict is inside a volume. They are told all the same,
// so that whoever is to ask which to use can do it before there is a mapping.
func TestFilesThatClaimOneChapterAreToldAsVersionsBeforeTheyHaveAVolume(t *testing.T) {
	manga := versionsManga(t)

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	groups := report.Manga[0].ChapterVersions
	if len(groups) != 1 {
		t.Fatalf("%d version groups, want 1: %+v", len(groups), groups)
	}
	group := groups[0]
	if group.Chapter != 6 || group.Special != "" || group.Volume != nil || group.Resolution != "unresolved" {
		t.Errorf("group = %+v, want chapter 6 with no volume, unresolved", group)
	}
	wantUnits := []string{
		absolutePath(filepath.Join(manga, versionsSixA)),
		absolutePath(filepath.Join(manga, versionsSixB)),
	}
	if !reflect.DeepEqual(group.Units, wantUnits) {
		t.Errorf("units = %v, want %v", group.Units, wantUnits)
	}
	for _, unit := range report.Manga[0].Units {
		if unit.Disposition != "unassigned" {
			t.Errorf("%q is %q, want unassigned", unit.Name, unit.Disposition)
		}
	}
}

// What the report tells of a unit is enough to show it: how many pages it has and which is the first.
func TestAUnitTellsItsFirstPage(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, "Ch.1.cbz"), "ComicInfo.xml", "010.jpg", "002.jpg")
	writeFiles(t, filepath.Join(manga, "Ch.2"), "b.png", "a.png", "notes.txt")

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json")

	got := map[string]string{}
	for _, unit := range report.Manga[0].Units {
		got[unit.Name] = unit.FirstPage
	}
	want := map[string]string{"Ch.1": "002.jpg", "Ch.2": "a.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("first pages = %v, want %v", got, want)
	}
}

// With a volume, the same files are a conflict and are all left out, as they always were.
func TestWithNoChoiceTheVersionsOfAChapterAreStillAConflict(t *testing.T) {
	manga := versionsManga(t)

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json",
		"-metadata-file", writeMapping(t, manga, ""))

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	want := map[string]string{versionsFive: "included", versionsSixA: "conflict", versionsSixB: "conflict", versionsSeven: "included"}
	if got := dispositions(report.Manga[0].Units); !reflect.DeepEqual(got, want) {
		t.Errorf("dispositions = %v, want %v", got, want)
	}
	if got := report.Manga[0].ChapterVersions; len(got) != 1 || got[0].Resolution != "unresolved" || got[0].Volume == nil || *got[0].Volume != 1 {
		t.Errorf("version groups = %+v, want one, in volume 1, unresolved", got)
	}
	if codes := issueCodes(report.Manga[0].Issues); !reflect.DeepEqual(codes, []string{"chapter_conflict"}) {
		t.Errorf("issues = %v, want the conflict alone: a chapter in conflict is present, not missing", codes)
	}
}

// Choosing one version binds that one, whole, and the other is left out of the book and told so.
func TestAChoiceOfOneVersionBindsThatOne(t *testing.T) {
	manga := versionsManga(t)
	out := filepath.Join(t.TempDir(), "out")
	mapping := writeMapping(t, manga, `[{"chapter":"6","use":["`+versionsSixB+`"]}]`)

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json", "-metadata-file", mapping)

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	want := map[string]string{versionsFive: "included", versionsSixA: "not_chosen", versionsSixB: "included", versionsSeven: "included"}
	if got := dispositions(report.Manga[0].Units); !reflect.DeepEqual(got, want) {
		t.Errorf("dispositions = %v, want %v", got, want)
	}
	if got := report.Manga[0].ChapterVersions; len(got) != 1 || got[0].Resolution != "chosen" {
		t.Errorf("version groups = %+v, want one, chosen", got)
	}
	if codes := issueCodes(report.Manga[0].Issues); len(codes) != 0 {
		t.Errorf("issues = %v, want none: the chapter is not missing and not in conflict", codes)
	}
	entries := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz"))
	wantEntries := []string{
		"c001 - Five/p0001.jpg",
		"c002 - Six/p0001.jpg", "c002 - Six/p0002.jpg", "c002 - Six/p0003.jpg",
		"c003/p0001.jpg",
	}
	if !reflect.DeepEqual(entries, wantEntries) {
		t.Errorf("the volume holds %v, want %v: the version with three pages", entries, wantEntries)
	}
}

// Keeping both binds both, side by side in the order of the files, each told which it is of how many, and with
// the numbers of the chapters unchanged: the chapter after them is still chapter 7.
func TestKeepingBothVersionsBindsBothAsAlternativesAndKeepsTheNumbers(t *testing.T) {
	manga := versionsManga(t)
	out := filepath.Join(t.TempDir(), "out")
	mapping := writeMapping(t, manga, `[{"chapter":"6","use":["`+versionsSixA+`","`+versionsSixB+`"]}]`)

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json", "-metadata-file", mapping, "-chapter-labels", "number")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	for name, disposition := range dispositions(report.Manga[0].Units) {
		if disposition != "included" {
			t.Errorf("%q is %q, want included", name, disposition)
		}
	}
	if codes := issueCodes(report.Manga[0].Issues); len(codes) != 0 {
		t.Errorf("issues = %v, want none", codes)
	}
	wantEntries := []string{
		"Ch.5 - Five/p0001.jpg",
		"Ch.6 - Six (version 1 of 2)/p0001.jpg", "Ch.6 - Six (version 1 of 2)/p0002.jpg",
		"Ch.6 - Six (version 2 of 2)/p0001.jpg", "Ch.6 - Six (version 2 of 2)/p0002.jpg", "Ch.6 - Six (version 2 of 2)/p0003.jpg",
		"Ch.7/p0001.jpg",
	}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")); !reflect.DeepEqual(got, wantEntries) {
		t.Errorf("the volume holds %v, want %v", got, wantEntries)
	}
	// The report names each chapter as the book does, and says which file it came from.
	volume := report.Manga[0].Volumes[0]
	labels := make([]string, len(volume.ChapterEntries))
	for i, entry := range volume.ChapterEntries {
		labels[i] = entry.Label
	}
	wantLabels := []string{"Ch.5 - Five", "Ch.6 - Six (version 1 of 2)", "Ch.6 - Six (version 2 of 2)", "Ch.7"}
	if !reflect.DeepEqual(labels, wantLabels) {
		t.Errorf("chapter labels = %v, want %v", labels, wantLabels)
	}
	if got, want := volume.ChapterEntries[2], (chapterEntryReport{Label: "Ch.6 - Six (version 2 of 2)", Path: absolutePath(filepath.Join(manga, versionsSixB)), PageCount: 3}); got != want {
		t.Errorf("third chapter = %+v, want %+v", got, want)
	}
}

// Without the labels by number, alternatives are still told apart.
func TestAlternativesAreToldApartWhateverTheLabelStyle(t *testing.T) {
	manga := versionsManga(t)
	out := filepath.Join(t.TempDir(), "out")
	mapping := writeMapping(t, manga, `[{"chapter":"6","use":["`+versionsSixA+`","`+versionsSixB+`"]}]`)

	code, _, stderr := runText("-input", manga, "-output", out, "-metadata-file", mapping)

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	got := strings.Join(zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")), "\n")
	for _, label := range []string{"c002 - Six (version 1 of 2)/", "c003 - Six (version 2 of 2)/", "c004/"} {
		if !strings.Contains(got, label) {
			t.Errorf("the volume has no %q:\n%s", label, got)
		}
	}
}

// A choice that names no file the chapter has is not used, and is told: the files are then in conflict, and
// nothing is guessed in their place.
func TestAChoiceThatNamesNoFileOfTheChapterIsNotUsed(t *testing.T) {
	manga := versionsManga(t)
	mapping := writeMapping(t, manga, `[{"chapter":"6","use":["Ch.0006 - Six (en) [Gone].cbz"]}]`)

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json", "-metadata-file", mapping)

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if codes := issueCodes(report.Manga[0].Issues); !reflect.DeepEqual(codes, []string{"chapter_choice_unmatched", "chapter_conflict"}) {
		t.Errorf("issues = %v, want the unused choice and the conflict", codes)
	}
	if got := report.Manga[0].ChapterVersions; len(got) != 1 || got[0].Resolution != "unresolved" {
		t.Errorf("version groups = %+v, want one, unresolved", got)
	}
	if got := unitNamed(t, report.Manga[0].Units, versionsSixA).Disposition; got != "conflict" {
		t.Errorf("a version is %q, want conflict", got)
	}
}

// A choice of no file is the choice to leave the chapter out, and it is a choice: the chapter is not in
// conflict, and it is a gap.
func TestAChoiceOfNoFileLeavesTheChapterOut(t *testing.T) {
	manga := versionsManga(t)
	mapping := writeMapping(t, manga, `[{"chapter":"6","use":[]}]`)

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json", "-metadata-file", mapping)

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	want := map[string]string{versionsFive: "included", versionsSixA: "not_chosen", versionsSixB: "not_chosen", versionsSeven: "included"}
	if got := dispositions(report.Manga[0].Units); !reflect.DeepEqual(got, want) {
		t.Errorf("dispositions = %v, want %v", got, want)
	}
	if codes := issueCodes(report.Manga[0].Issues); !reflect.DeepEqual(codes, []string{"chapter_gap"}) {
		t.Errorf("issues = %v, want the gap alone", codes)
	}
}

// A choice about a chapter that one file claims has nothing to decide: the file stays, whatever the choice
// names, so that a choice made before a duplicate was removed does not take the chapter away.
func TestAChoiceAboutAChapterOfOneFileChangesNothing(t *testing.T) {
	manga := versionsManga(t)
	if err := os.Remove(filepath.Join(manga, versionsSixA)); err != nil {
		t.Fatal(err)
	}
	mapping := writeMapping(t, manga, `[{"chapter":"6","use":["`+versionsSixA+`"]}]`)

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json", "-metadata-file", mapping)

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := unitNamed(t, report.Manga[0].Units, versionsSixB).Disposition; got != "included" {
		t.Errorf("the one file of the chapter is %q, want included", got)
	}
	if len(report.Manga[0].ChapterVersions) != 0 || len(report.Manga[0].Issues) != 0 {
		t.Errorf("versions %+v, issues %+v; want none", report.Manga[0].ChapterVersions, report.Manga[0].Issues)
	}
}

// The files in a volume folder are named in the choice by their path below the manga folder, with "/": two
// volume folders may hold chapters of one name.
func TestVersionsInAVolumeFolderAreChosenByTheirPath(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, "Volume 1", "Ch.0001 (en) [A].cbz"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Volume 1", "Ch.0001 (en) [B].cbz"), "001.jpg", "002.jpg")
	writeArchive(t, filepath.Join(manga, "Volume 2", "Ch.0001 (en).cbz"), "001.jpg")
	mapping := filepath.Join(t.TempDir(), "mapping.json")
	writeFile(t, mapping, `{"schema_version":1,"volumes":[],"chapter_versions":[{"chapter":"1","use":["Volume 1/Ch.0001 (en) [B].cbz"]}]}`)

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json", "-metadata-file", mapping)

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	got := map[string]string{}
	for _, unit := range report.Manga[0].Units {
		got[filepath.ToSlash(strings.TrimPrefix(unit.Path, absolutePath(manga)+string(filepath.Separator)))] = unit.Disposition
	}
	want := map[string]string{
		"Volume 1/Ch.0001 (en) [A].cbz": "not_chosen",
		"Volume 1/Ch.0001 (en) [B].cbz": "included",
		"Volume 2/Ch.0001 (en).cbz":     "included",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dispositions = %v, want %v", got, want)
	}
}

// Backslashes, as a Windows path is written, name the same files.
func TestAChoiceMayNameAFileWithBackslashes(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeArchive(t, filepath.Join(manga, "Volume 1", "Ch.0001 (en) [A].cbz"), "001.jpg")
	writeArchive(t, filepath.Join(manga, "Volume 1", "Ch.0001 (en) [B].cbz"), "001.jpg")
	mapping := filepath.Join(t.TempDir(), "mapping.json")
	writeFile(t, mapping, `{"schema_version":1,"volumes":[],"chapter_versions":[{"chapter":"1","use":["Volume 1\\Ch.0001 (en) [A].cbz"]}]}`)

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json", "-metadata-file", mapping)

	if got := report.Manga[0].ChapterVersions; len(got) != 1 || got[0].Resolution != "chosen" {
		t.Errorf("version groups = %+v, want one, chosen", got)
	}
}

func TestAnUnknownChapterLabelStyleIsRefused(t *testing.T) {
	report, _, code := runMachineForTest(t, "-input", versionsManga(t), "-chapter-labels", "fancy", "-json")

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if len(report.Issues) != 1 || report.Issues[0].Code != "invalid_arguments" || !strings.Contains(report.Issues[0].Diagnostic, "fancy") {
		t.Errorf("issues = %+v, want invalid_arguments naming the style", report.Issues)
	}
}

// The choices travel in the mapping file, which a bad one makes unusable as for any other fault in it.
func TestABadChoiceInTheMappingFileIsAMetadataError(t *testing.T) {
	manga := versionsManga(t)
	mapping := writeMapping(t, manga, `[{"chapter":"5-7","use":[]}]`)

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run", "-json", "-metadata-file", mapping)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if codes := issueCodes(report.Manga[0].Issues); !reflect.DeepEqual(codes, []string{"metadata_load_failed"}) {
		t.Errorf("issues = %v, want metadata_load_failed", codes)
	}
}
