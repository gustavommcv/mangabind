package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCorruptCBZ makes a .cbz whose first image is fine and whose second has
// a flipped byte in its stored data, so that the archive lists both and reading
// the second fails the checksum.
func writeCorruptCBZ(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range [][2]string{{"a.jpg", "FINE-IMAGE-DATA"}, {"b.jpg", "PAYLOAD-AAAA-DATA"}} {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e[0], Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := bytes.Replace(buf.Bytes(), []byte("PAYLOAD-AAAA"), []byte("PAYLOAD-AAAB"), 1)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// aMangaWhoseSecondVolumeCannotBeWritten has three volumes of which the middle
// one is a .cbz that fails while it is copied, and a folder no parser reads.
func aMangaWhoseSecondVolumeCannotBeWritten(t *testing.T) string {
	t.Helper()
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), "001.jpg")
	if err := os.MkdirAll(manga, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCorruptCBZ(t, filepath.Join(manga, "Vol.02 Ch.002.cbz"))
	writeFiles(t, filepath.Join(manga, "Vol.03 Ch.003"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "no parser reads this"), "001.jpg")
	return manga
}

func TestAVolumeThatCannotBeWrittenDoesNotStopTheNextOnes(t *testing.T) {
	manga := aMangaWhoseSecondVolumeCannotBeWritten(t)
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := runText("-input", manga, "-output", out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1: something was not written", code)
	}
	for _, volume := range []string{"Series - Vol.01.cbz", "Series - Vol.03.cbz"} {
		if !strings.Contains(stdout, "wrote "+filepath.Join(out, volume)) {
			t.Errorf("stdout does not say %s was written:\n%s", volume, stdout)
		}
		if got := zipEntryNames(t, filepath.Join(out, volume)); len(got) != 1 {
			t.Errorf("%s holds %v, want its one page", volume, got)
		}
	}
	for _, want := range []string{
		"mangabind: writing volume 2:",
		"1 of 3 volume(s) could not be written",
		// What used to be lost when the loop stopped: the warnings that come after it.
		`warning: could not parse chapter name, skipped: "no parser reads this"`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, stderr)
		}
	}
}

func TestTheReportOfARunWithAFailedVolumeIsComplete(t *testing.T) {
	manga := aMangaWhoseSecondVolumeCannotBeWritten(t)
	out := filepath.Join(t.TempDir(), "out")

	report, _, code := runMachineForTest(t, "-input", manga, "-output", out, "-json")

	if code != 1 || report.Status != "failed" {
		t.Errorf("exit code = %d, status = %s, want 1 and failed", code, report.Status)
	}
	volumes := report.Manga[0].Volumes
	if len(volumes) != 3 || !volumes[0].Written || volumes[1].Written || !volumes[2].Written {
		t.Errorf("volumes = %+v, want the first and the third written and the second not", volumes)
	}
	if report.Summary.Volumes != 2 {
		t.Errorf("summary volumes = %d, want the 2 that were written", report.Summary.Volumes)
	}
	var failed, unparsed int
	for _, issue := range report.Manga[0].Issues {
		switch issue.Code {
		case "volume_write_failed":
			failed++
			if issue.Severity != "error" || issue.Volume == nil || *issue.Volume != 2 || !issue.Recoverable {
				t.Errorf("volume_write_failed = %+v, want a recoverable error for volume 2", issue)
			}
		case "unparsed_chapter":
			unparsed++
		}
	}
	if failed != 1 || unparsed != 1 {
		t.Errorf("volume_write_failed: %d, unparsed_chapter: %d, want 1 and 1", failed, unparsed)
	}
}

func TestProgressNeverGoesBackWhenAVolumeFails(t *testing.T) {
	manga := aMangaWhoseSecondVolumeCannotBeWritten(t)
	out := filepath.Join(t.TempDir(), "out")
	var stdout, stderr bytes.Buffer

	code := runCLI([]string{"-input", manga, "-output", out, "-json", "-progress-json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	last := 0
	var final machineProgress
	for _, event := range parseProgressLines(t, stderr.String()) {
		if event.CompletedPages == nil {
			continue
		}
		if *event.CompletedPages < last {
			t.Fatalf("completed pages went from %d back to %d at %+v", last, *event.CompletedPages, event)
		}
		last = *event.CompletedPages
		final = event
	}
	// Volume 1: 1 page. Volume 2: 1 page copied before it failed. Volume 3: 1 page.
	if final.State != "completed" || final.VolumeIndex != 3 || last != 3 {
		t.Errorf("the last event is %+v with %d pages, want volume 3 completed at 3 pages", final, last)
	}
}
