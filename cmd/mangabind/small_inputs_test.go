package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMetadata(t *testing.T, manga, content string) string {
	t.Helper()
	path := filepath.Join(manga, "mangabind.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// aMangaWithoutVolumesInItsNames has chapters that only a metadata file can
// place in volumes.
func aMangaWithoutVolumesInItsNames(t *testing.T) string {
	t.Helper()
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{"Ch.001", "Ch.002", "Ch.003"} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}
	return manga
}

func TestAChapterListedUnderTwoVolumesIsWarnedAboutOncePerFile(t *testing.T) {
	manga := aMangaWithoutVolumesInItsNames(t)
	metadataPath := writeMetadata(t, manga, `{"schema_version": 1, "volumes": [
		{"number": "1", "chapters": ["1-2"]},
		{"number": "2", "chapters": ["2-3"]}
	]}`)

	report, _, code := runMachineForTest(t, "-input", manga, "-dry-run", "-json")

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var found []machineIssue
	for _, issue := range report.Manga[0].Issues {
		if issue.Code == "metadata_duplicate_chapter" {
			found = append(found, issue)
		}
	}
	if len(found) != 1 {
		t.Fatalf("issues = %+v, want exactly one metadata_duplicate_chapter", report.Manga[0].Issues)
	}
	issue := found[0]
	if issue.Severity != "warning" || issue.Stage != "metadata" || issue.Path != metadataPath {
		t.Errorf("issue = %+v, want a warning at the metadata stage naming the file", issue)
	}
	if want := "chapter 2 under volumes 1 and 2; the last one, volume 2, is used"; !strings.Contains(issue.Message, want) {
		t.Errorf("message = %q, want it to contain %q", issue.Message, want)
	}
	// The last listing is what applies.
	for _, unit := range report.Manga[0].Units {
		if unit.Name == "Ch.002" && (unit.EffectiveVolume == nil || *unit.EffectiveVolume != 2) {
			t.Errorf("Ch.002 went to volume %v, want 2", unit.EffectiveVolume)
		}
	}
}

func TestSeveralDuplicatesMakeOneWarningThatNamesTheFirst(t *testing.T) {
	manga := aMangaWithoutVolumesInItsNames(t)
	writeMetadata(t, manga, `{"schema_version": 1, "volumes": [
		{"number": "1", "chapters": ["1-3"]},
		{"number": "2", "chapters": ["1-3"]}
	]}`)

	code, _, stderr := runText("-input", manga, "-dry-run")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	want := "warning: the metadata file lists 3 chapters under more than one volume (the first is chapter 1, under volumes 1 and 2); the last listing of each is used"
	if strings.Count(stderr, "the metadata file lists") != 1 || !strings.Contains(stderr, want) {
		t.Errorf("stderr is not the one warning %q:\n%s", want, stderr)
	}
}

func TestAMetadataFileThatCouldNotBeUsedIsAnErrorNotACrash(t *testing.T) {
	cases := []struct {
		name, content, want string
	}{
		{"a volume that is not a number", `{"schema_version": 1, "volumes": [{"number": "NaN", "chapters": ["1"]}]}`, "invalid number"},
		{"a negative volume", `{"schema_version": 1, "volumes": [{"number": "-3", "chapters": ["1"]}]}`, "invalid number"},
		{"a range that asks for hundreds of gigabytes", `{"schema_version": 1, "volumes": [{"number": "1", "chapters": ["1-9000000000"]}]}`, "too large"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			manga := aMangaWithoutVolumesInItsNames(t)
			writeMetadata(t, manga, c.content)

			code, stdout, stderr := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"))

			if code != 1 || stdout != "" {
				t.Errorf("code = %d, stdout = %q, want 1 and nothing", code, stdout)
			}
			if !strings.Contains(stderr, c.want) || strings.Contains(stderr, "goroutine") {
				t.Errorf("stderr = %q, want a message with %q and no stack dump", stderr, c.want)
			}

			report, _, jsonCode := runMachineForTest(t, "-input", manga, "-dry-run", "-json")
			if jsonCode != 1 || len(report.Manga[0].Issues) != 1 || report.Manga[0].Issues[0].Code != "metadata_load_failed" {
				t.Errorf("machine: code = %d, issues = %+v, want one metadata_load_failed", jsonCode, report.Manga[0].Issues)
			}
		})
	}
}

func TestMetadataRangesAtTheIntLimitDoNotBlockProcessing(t *testing.T) {
	manga := aMangaWithoutVolumesInItsNames(t)
	writeMetadata(t, manga, fmt.Sprintf(`{"schema_version": 1, "volumes": [
		{"number": "1", "chapters": ["%d-%d", "1-3"]}
	]}`, math.MaxInt, math.MaxInt))
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 || report.Status != "completed" || stderr != "" {
		t.Fatalf("code = %d, status = %s, stderr = %q, want success", code, report.Status, stderr)
	}
	if report.Summary.Volumes != 1 || report.Summary.Pages != 3 || report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("summary = %+v, want one volume of three pages without issues", report.Summary)
	}
	volumes := report.Manga[0].Volumes
	if len(volumes) != 1 || !volumes[0].Written || volumes[0].Number != 1 {
		t.Fatalf("volumes = %+v, want volume 1 written", volumes)
	}
	if pages := zipEntryNames(t, volumes[0].OutputPath); len(pages) != 3 {
		t.Errorf("the written volume holds %v, want three pages", pages)
	}
}

func TestAVolumeNumberTooLargeForAnIntStillGivesAFileName(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.99999999999999999999 Ch.001"), "001.jpg")

	code, stdout, stderr := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run")

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Series - Vol.100000000000000000000.cbz") || strings.Contains(stdout, "-9223372036854775808") {
		t.Errorf("stdout = %q, want the volume named in full", stdout)
	}
}

// aLibraryWhereBetaFailsItsSecondVolume has two manga: Alpha is one volume of
// one page; Beta's first volume is written and its second cannot be.
func aLibraryWhereBetaFailsItsSecondVolume(t *testing.T) string {
	t.Helper()
	library := t.TempDir()
	writeFiles(t, filepath.Join(library, "Alpha", "Vol.01 Ch.001"), "001.jpg")
	writeFiles(t, filepath.Join(library, "Beta", "Vol.01 Ch.001"), "001.jpg")
	writeCorruptCBZ(t, filepath.Join(library, "Beta", "Vol.02 Ch.002.cbz"))
	return library
}

func TestTheBatchTallyCountsWhatAFailingMangaDidWrite(t *testing.T) {
	library := aLibraryWhereBetaFailsItsSecondVolume(t)
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := runText("-input", library, "-batch", "-output", out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	// Alpha's volume and Beta's first are on disk: 2 volumes and 2 pages. The
	// tally used to say 1 and 1, leaving out Beta's because Beta had an error.
	if want := "processed 2 manga, 2 volume(s), 2 page(s) total"; !strings.Contains(stdout, want) {
		t.Errorf("stdout is missing %q:\n%s", want, stdout)
	}
	if got := folderListing(t, out); len(got) != 2 {
		t.Errorf("the output folder holds %v, want the two finished volumes", got)
	}
	if !strings.Contains(stderr, "1 of 2 manga had errors") && !strings.Contains(stderr, "Beta: 1 of 2 volume(s) could not be written") {
		t.Errorf("stderr does not say Beta had an error:\n%s", stderr)
	}
}

func TestACombinedSeriesThatCannotBeWrittenCountsAsNothingWritten(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), "001.jpg")
	writeCorruptCBZ(t, filepath.Join(manga, "Vol.02 Ch.002.cbz"))

	report, _, code := runMachineForTest(t, "-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-combine", "-json")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	// The volumes were counted while the plan was made; the file never existed.
	if got := report.Manga[0].Summary; got.Volumes != 0 || got.Pages != 0 {
		t.Errorf("manga summary = %+v, want no volumes and no pages written", got)
	}
	if report.Summary.Volumes != 0 || report.Summary.Pages != 0 {
		t.Errorf("summary = %+v, want no volumes and no pages written", report.Summary)
	}
}

func aLibraryWithLinksToFolders(t *testing.T) (library, outside string) {
	t.Helper()
	library = t.TempDir()
	outside = filepath.Join(t.TempDir(), "Elsewhere")
	writeFiles(t, filepath.Join(library, "Alpha", "Vol.01 Ch.001"), "001.jpg")
	writeFiles(t, filepath.Join(outside, "Vol.01 Ch.001"), "001.jpg")
	symlink(t, "Alpha", filepath.Join(library, "AlphaLink"))
	symlink(t, outside, filepath.Join(library, "Seeded"))
	symlink(t, filepath.Join(library, "gone"), filepath.Join(library, "Ghost"))
	return library, outside
}

func TestABatchSaysWhenALinkToAMangaFolderWasNotFollowed(t *testing.T) {
	library, _ := aLibraryWithLinksToFolders(t)

	code, stdout, stderr := runText("-input", library, "-batch", "-output", filepath.Join(t.TempDir(), "out"))

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	for _, want := range []string{
		`warning: found a link "AlphaLink" that leads to a folder, and the manga folders of a library are not followed through links, skipped`,
		`warning: found a link "Seeded" that leads to a folder`,
		`warning: found a link "Ghost" that leads nowhere, skipped`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stdout, "AlphaLink") || strings.Contains(stdout, "Seeded") || !strings.Contains(stdout, "processed 1 manga") {
		t.Errorf("a linked folder was processed as a manga:\n%s", stdout)
	}
}

func TestAMachineBatchReportsALinkedMangaFolderAtTheTopLevel(t *testing.T) {
	library, _ := aLibraryWithLinksToFolders(t)

	report, _, code := runMachineForTest(t, "-input", library, "-batch", "-dry-run", "-json")

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if len(report.Manga) != 1 || report.Manga[0].Name != "Alpha" {
		t.Errorf("manga = %+v, want Alpha alone", report.Manga)
	}
	paths := map[string]string{}
	for _, issue := range report.Issues {
		if issue.Code != "link_skipped" || issue.Severity != "warning" || issue.Stage != "inspect" {
			t.Errorf("issue = %+v, want only link_skipped warnings at inspect", issue)
		}
		paths[filepath.Base(issue.Path)] = issue.Message
	}
	if len(paths) != 3 {
		t.Errorf("top-level issues name %v, want AlphaLink, Seeded and Ghost", paths)
	}
	if report.Status != "completed_with_warnings" {
		t.Errorf("status = %s, want completed_with_warnings", report.Status)
	}
}
