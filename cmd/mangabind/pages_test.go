package main

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeFiles puts the named files, each holding its own name, in dir.
func writeFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// writeCBZ makes a .cbz holding the named entries, each with its own name as
// content; a name that ends in a slash is a folder entry.
func writeCBZ(t *testing.T, path string, entries ...string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, "/") {
			if _, err := w.Write([]byte(name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func zipEntryNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func TestFilesThatAreNotImagesAreNotPagesAndAreReportedOncePerChapter(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	one := filepath.Join(manga, "Vol.01 Ch.001")
	two := filepath.Join(manga, "Vol.01 Ch.002")
	writeFiles(t, one, "001.jpg", "002.jpg", ".DS_Store", "notes.txt", "credits.html")
	writeFiles(t, two, "001.png", "Thumbs.db", "ComicInfo.xml", "._001.png")
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\nstderr: %s", code, stderr)
	}
	if report.Status != "completed_with_warnings" || report.Summary.Warnings != 1 || report.Summary.Pages != 3 {
		t.Errorf("status = %s, summary = %+v, want one warning and 3 pages", report.Status, report.Summary)
	}
	units := report.Manga[0].Units
	if len(units) != 2 || units[0].PageCount != 2 || units[1].PageCount != 1 {
		t.Errorf("units = %+v, want page counts 2 and 1: only the images", units)
	}
	if len(report.Manga[0].Issues) != 1 {
		t.Fatalf("issues = %+v, want exactly one (the junk is not worth a word)", report.Manga[0].Issues)
	}
	issue := report.Manga[0].Issues[0]
	if issue.Code != "unsupported_page_files" || issue.Severity != "warning" || issue.Stage != "inspect" {
		t.Errorf("issue = %+v, want a warning unsupported_page_files at inspect", issue)
	}
	if issue.Volume == nil || *issue.Volume != 1 || issue.Chapter == nil || *issue.Chapter != 1 {
		t.Errorf("issue names volume %v and chapter %v, want 1 and 1", issue.Volume, issue.Chapter)
	}
	if issue.Path != one {
		t.Errorf("issue path = %q, want the chapter %q", issue.Path, one)
	}
	wantRelated := []string{filepath.Join(one, "credits.html"), filepath.Join(one, "notes.txt")}
	if !reflect.DeepEqual(issue.RelatedPaths, wantRelated) {
		t.Errorf("related paths = %v, want %v", issue.RelatedPaths, wantRelated)
	}
	for _, want := range []string{"2 files that are not images", `"credits.html"`, `"notes.txt"`, "Vol.01 Ch.001"} {
		if !strings.Contains(issue.Message, want) {
			t.Errorf("message %q is missing %q", issue.Message, want)
		}
	}

	// c001 is the chapter with the lower number: Ch.001's pages come first.
	wantEntries := []string{"c001/p0001.jpg", "c001/p0002.jpg", "c002/p0001.png"}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")); !reflect.DeepEqual(got, wantEntries) {
		t.Errorf("the volume holds %v, want %v", got, wantEntries)
	}
}

func TestTheWarningAboutSkippedFilesIsOnStderrAndQuietDoesNotHideIt(t *testing.T) {
	for _, extra := range [][]string{nil, {"-quiet"}} {
		manga := filepath.Join(t.TempDir(), "Series")
		writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), "001.jpg", "notes.txt")

		code, stdout, stderr := runText(append([]string{"-input", manga, "-output", filepath.Join(t.TempDir(), "out")}, extra...)...)

		if code != 0 {
			t.Fatalf("%v: exit code = %d\nstderr: %s", extra, code, stderr)
		}
		want := `warning: chapter "Vol.01 Ch.001": skipped 1 file that is not an image: "notes.txt"`
		if !strings.Contains(stderr, want) {
			t.Errorf("%v: stderr is missing %q:\n%s", extra, want, stderr)
		}
		if strings.Contains(stdout, "notes.txt") {
			t.Errorf("%v: stdout names the skipped file: %q", extra, stdout)
		}
	}
}

func TestUnsupportedInputFilesAreReportedEvenWithoutChapters(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{"execute", nil},
		{"plan", []string{"-dry-run"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			manga := filepath.Join(t.TempDir(), "Series")
			writeFiles(t, manga, "chapter.pdf", "chapter.EPUB", ".DS_Store", "Thumbs.db")
			out := filepath.Join(t.TempDir(), "out")
			args := append([]string{"-input", manga, "-output", out}, mode.args...)

			report, stderr, code := runMachineForTest(t, append(args, "-json")...)

			if code != 0 || stderr != "" || report.Status != "completed_with_warnings" || len(report.Manga) != 1 {
				t.Fatalf("code = %d, stderr = %q, report = %+v, want one manga with warnings", code, stderr, report)
			}
			result := report.Manga[0]
			if report.Summary.Warnings != 3 || result.Summary.Warnings != 3 || len(result.Issues) != 3 ||
				report.Summary.Errors != 0 || report.Summary.Volumes != 0 || report.Summary.Pages != 0 ||
				len(result.Units) != 0 || len(result.Volumes) != 0 {
				t.Fatalf("result = %+v, summary = %+v, want two file warnings and no_chapters_found with nothing produced", result, report.Summary)
			}
			for _, name := range []string{"chapter.pdf", "chapter.EPUB"} {
				found := false
				for _, issue := range result.Issues {
					if issue.Code == "unsupported_input_file" && issue.Path == absolutePath(filepath.Join(manga, name)) {
						found = true
						if issue.Severity != "warning" || issue.Stage != "inspect" || issue.Manga != "Series" || !issue.Recoverable ||
							!strings.Contains(issue.Message, strings.ToLower(filepath.Ext(name))+" chapters aren't supported") {
							t.Errorf("issue = %+v, want a warning explaining the unsupported format", issue)
						}
					}
				}
				if !found {
					t.Errorf("issues = %+v, want unsupported_input_file for %s", result.Issues, name)
				}
			}
			if issue := result.Issues[2]; issue.Code != "no_chapters_found" || issue.Path != absolutePath(manga) {
				t.Errorf("last issue = %+v, want no_chapters_found for the input folder", issue)
			}

			code, stdout, stderr := runText(append(args, "-quiet")...)
			if code != 0 || stdout != "" || strings.Count(stderr, "warning:") != 3 {
				t.Errorf("quiet run: code = %d, stdout = %q, stderr = %q, want three warnings only on stderr", code, stdout, stderr)
			}
			for _, issue := range result.Issues[:2] {
				if strings.Count(stderr, "warning: "+issue.Message) != 1 {
					t.Errorf("stderr = %q, want the file warning once: %s", stderr, issue.Message)
				}
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Errorf("an input without chapters created the output folder: %v", err)
			}
		})
	}
}

func TestAPlanCountsOnlyTheImages(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), "001.jpg", "002.jpg", ".DS_Store", "notes.txt")

	code, stdout, _ := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"), "-dry-run")

	if code != 0 || !strings.Contains(stdout, "(2 pages)") {
		t.Errorf("code = %d, stdout = %q, want a plan of 2 pages", code, stdout)
	}
}

func TestManySkippedFilesAreNamedFewAtATimeButAllAreListed(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	chapter := filepath.Join(manga, "Vol.01 Ch.001")
	names := []string{"001.jpg"}
	for i := 1; i <= 7; i++ {
		names = append(names, fmt.Sprintf("extra%d.txt", i))
	}
	writeFiles(t, chapter, names...)

	report, _, code := runMachineForTest(t, "-input", manga, "-dry-run", "-json")

	if code != 0 || len(report.Manga[0].Issues) != 1 {
		t.Fatalf("code = %d, issues = %+v, want one issue", code, report.Manga[0].Issues)
	}
	issue := report.Manga[0].Issues[0]
	want := `chapter "Vol.01 Ch.001": skipped 7 files that are not images: "extra1.txt", "extra2.txt", "extra3.txt" and 4 more`
	if issue.Message != want {
		t.Errorf("message = %q\nwant      %q", issue.Message, want)
	}
	if len(issue.RelatedPaths) != 7 {
		t.Errorf("related paths = %d, want all 7", len(issue.RelatedPaths))
	}
}

func TestAChapterOfOnlyJunkIsEmptyWithNoWarningAboutTheJunk(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), ".DS_Store", "Thumbs.db")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.002"), "001.jpg")

	report, _, code := runMachineForTest(t, "-input", manga, "-dry-run", "-json")

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	units := report.Manga[0].Units
	if units[0].Disposition != "empty" || units[0].PageCount != 0 || units[1].PageCount != 1 {
		t.Errorf("units = %+v, want the first empty and the second with one page", units)
	}
	var codes []string
	for _, issue := range report.Manga[0].Issues {
		codes = append(codes, issue.Code)
	}
	if !reflect.DeepEqual(codes, []string{"empty_chapter"}) {
		t.Errorf("issue codes = %v, want only empty_chapter", codes)
	}
}

func TestAnArchiveChapterIsCountedByItsImagesToo(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	if err := os.MkdirAll(manga, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(manga, "Vol.01 Ch.001.cbz")
	writeCBZ(t, archive,
		"001.jpg", "002.jpg",
		"__MACOSX/", "__MACOSX/._001.jpg", "__MACOSX/001.jpg",
		"ComicInfo.xml", "credits.txt",
	)
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\nstderr: %s", code, stderr)
	}
	if unit := report.Manga[0].Units[0]; unit.PageCount != 2 || unit.Kind != "cbz" {
		t.Errorf("unit = %+v, want a cbz of 2 pages", unit)
	}
	if len(report.Manga[0].Issues) != 1 {
		t.Fatalf("issues = %+v, want one", report.Manga[0].Issues)
	}
	issue := report.Manga[0].Issues[0]
	if issue.Code != "unsupported_page_files" || issue.Path != archive {
		t.Errorf("issue = %+v, want unsupported_page_files at the archive", issue)
	}
	if issue.Diagnostic != "entries: credits.txt" || len(issue.RelatedPaths) != 0 {
		t.Errorf("diagnostic = %q, related paths = %v, want the entry named in the diagnostic only", issue.Diagnostic, issue.RelatedPaths)
	}
	if got, want := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")), []string{"c001/p0001.jpg", "c001/p0002.jpg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the volume holds %v, want %v", got, want)
	}
}

func TestSkippedPagesMessageSaysOneFileInTheSingular(t *testing.T) {
	got := skippedPagesMessage("Ch.1", []string{"a.txt"})
	if want := `chapter "Ch.1": skipped 1 file that is not an image: "a.txt"`; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}
