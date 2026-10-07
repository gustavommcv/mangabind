package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeDeflateBomb makes a .cbz whose one entry is size bytes of zeros,
// deflated: about a thousand times smaller than it expands.
func writeDeflateBomb(t *testing.T, zw *zip.Writer, name string, size int) {
	t.Helper()
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
}

func TestAnEntryThatExpandsAThousandFoldIsLeftOutAndNamed(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	if err := os.MkdirAll(manga, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(manga, "Vol.01 Ch.001.cbz")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "001.jpg", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("a real page")); err != nil {
		t.Fatal(err)
	}
	writeDeflateBomb(t, zw, "002.jpg", 24<<20) // 24 MiB of zeros in about 24 KiB
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 200<<10 {
		t.Fatalf("the test archive is %d bytes: it is not the small file the test means", buf.Len())
	}
	out := filepath.Join(t.TempDir(), "out")

	report, stderr, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 0 {
		t.Fatalf("exit code = %d\nstderr: %s", code, stderr)
	}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")); len(got) != 1 || got[0] != "c001/p0001.jpg" {
		t.Errorf("the volume holds %v, want only the real page", got)
	}
	if info, err := os.Stat(filepath.Join(out, "Series - Vol.01.cbz")); err != nil || info.Size() > 4<<10 {
		t.Errorf("the volume is %v (err %v): the 24 MiB entry was copied", info, err)
	}
	if report.Manga[0].Units[0].PageCount != 1 {
		t.Errorf("page count = %d, want 1", report.Manga[0].Units[0].PageCount)
	}
	var found []machineIssue
	for _, issue := range report.Manga[0].Issues {
		if issue.Code == "page_entries_too_large" {
			found = append(found, issue)
		}
	}
	if len(found) != 1 {
		t.Fatalf("issues = %+v, want exactly one page_entries_too_large", report.Manga[0].Issues)
	}
	issue := found[0]
	if issue.Severity != "warning" || issue.Stage != "inspect" || issue.Path != archive {
		t.Errorf("issue = %+v, want a warning at inspect naming the archive", issue)
	}
	if issue.Volume == nil || *issue.Volume != 1 || issue.Chapter == nil || *issue.Chapter != 1 {
		t.Errorf("issue names volume %v and chapter %v, want 1 and 1", issue.Volume, issue.Chapter)
	}
	for _, want := range []string{`skipped 1 entry that expands to far more than a page can be`, `"002.jpg"`, "24.0 MiB", "from "} {
		if !strings.Contains(issue.Message, want) {
			t.Errorf("message %q is missing %q", issue.Message, want)
		}
	}
	if !strings.HasPrefix(issue.Diagnostic, "entries: 002.jpg (25165824 bytes from ") {
		t.Errorf("diagnostic = %q, want the entry with its exact sizes", issue.Diagnostic)
	}
}

func TestACbzOfNothingButEntriesThatExpandTooFarIsAnEmptyChapter(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	if err := os.MkdirAll(manga, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeDeflateBomb(t, zw, "001.jpg", 24<<20)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manga, "Vol.01 Ch.001.cbz"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"))

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	for _, want := range []string{
		`warning: chapter "Vol.01 Ch.001": skipped 1 entry that expands to far more than a page can be: "001.jpg" (24.0 MiB, from `,
		`warning: chapter has no pages, skipped: "Vol.01 Ch.001"`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stdout, "wrote ") {
		t.Errorf("a volume was written from nothing:\n%s", stdout)
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		bytes uint64
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{400 << 10, "400.0 KiB"},
		{25165824, "24.0 MiB"},
		{1 << 30, "1.0 GiB"},
		{5 << 40, "5.0 TiB"},
	}
	for _, c := range cases {
		if got := formatSize(c.bytes); got != c.want {
			t.Errorf("formatSize(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

func TestAChapterThatWasReadButHoldsNoPagesIsNotSaidToBeUnparsed(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), ".DS_Store") // read, and empty
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.002"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "no parser reads this"), "001.jpg")

	code, _, stderr := runText("-input", manga, "-output", filepath.Join(t.TempDir(), "out"))

	if code != 0 {
		t.Fatalf("exit code = %d\n%s", code, stderr)
	}
	if !strings.Contains(stderr, `warning: chapter has no pages, skipped: "Vol.01 Ch.001"`) {
		t.Errorf("stderr does not say the empty chapter has no pages:\n%s", stderr)
	}
	if !strings.Contains(stderr, `warning: could not parse chapter name, skipped: "no parser reads this"`) {
		t.Errorf("stderr does not say the unparsed one could not be parsed:\n%s", stderr)
	}
	if strings.Contains(stderr, `could not parse chapter name, skipped: "Vol.01 Ch.001"`) {
		t.Errorf("a chapter that parsed is said to be unparsed:\n%s", stderr)
	}
}
