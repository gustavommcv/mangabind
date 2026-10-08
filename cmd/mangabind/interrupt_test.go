package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestMain lets a test run the real main() in a child process: when the
// variable is set the test binary is the program, not the tests.
func TestMain(m *testing.M) {
	if os.Getenv("MANGABIND_RUN_MAIN") == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

// cancelOn is a writer that cancels a run the first time what is written to it
// satisfies trigger, which is how a test presses Ctrl-C at an exact moment of a
// run: the writer is called from inside the run, so there is no timing in it.
type cancelOn struct {
	bytes.Buffer
	cancel  context.CancelFunc
	trigger func(written string) bool
	fired   bool
}

func (w *cancelOn) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if !w.fired && w.trigger(string(p)) {
		w.fired = true
		w.cancel()
	}
	return n, err
}

// aSeriesOfThreeVolumes has volume 2 three pages long, so that a run can be
// stopped in the middle of it.
func aSeriesOfThreeVolumes(t *testing.T) string {
	t.Helper()
	manga := filepath.Join(t.TempDir(), "Series")
	writeFiles(t, filepath.Join(manga, "Vol.01 Ch.001"), "001.jpg")
	writeFiles(t, filepath.Join(manga, "Vol.02 Ch.002"), "001.jpg", "002.jpg", "003.jpg")
	writeFiles(t, filepath.Join(manga, "Vol.03 Ch.003"), "001.jpg")
	return manga
}

func TestCtrlCInTheMiddleOfAVolumeRemovesItAndStopsTheRun(t *testing.T) {
	manga := aSeriesOfThreeVolumes(t)
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Press Ctrl-C when the first page of volume 2 has been copied.
	stderr := &cancelOn{cancel: cancel, trigger: func(written string) bool {
		return strings.Contains(written, `"state":"advanced"`) && strings.Contains(written, `"volume_index":2`)
	}}
	var stdout bytes.Buffer

	code := runCLIContext(ctx, []string{"-input", manga, "-output", out, "-json", "-progress-json"}, &stdout, stderr)

	if code != exitInterrupted {
		t.Errorf("exit code = %d, want %d", code, exitInterrupted)
	}
	if got, want := folderListing(t, out), []string{"Series - Vol.01.cbz"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the output folder holds %v, want only the finished volume %v: no half volume, no part file", got, want)
	}
	if got := zipEntryNames(t, filepath.Join(out, "Series - Vol.01.cbz")); len(got) != 1 {
		t.Errorf("volume 1 holds %v, want its page", got)
	}

	report := decodeReport(t, stdout.Bytes())
	if report.Status != "failed" {
		t.Errorf("status = %s, want failed", report.Status)
	}
	var interrupted []machineIssue
	for _, issue := range report.Manga[0].Issues {
		if issue.Code == "interrupted" {
			interrupted = append(interrupted, issue)
		}
	}
	if len(interrupted) != 1 {
		t.Fatalf("issues = %+v, want exactly one interrupted", report.Manga[0].Issues)
	}
	issue := interrupted[0]
	if issue.Severity != "error" || issue.Stage != "write" || !issue.Recoverable {
		t.Errorf("issue = %+v, want a recoverable error at the write stage", issue)
	}
	if !strings.Contains(issue.Message, "volume 2") || filepath.Base(issue.Path) != "Series - Vol.02.cbz" {
		t.Errorf("message = %q, path = %q, want them to name volume 2", issue.Message, issue.Path)
	}
	volumes := report.Manga[0].Volumes
	if len(volumes) != 3 || !volumes[0].Written || volumes[1].Written || volumes[2].Written {
		t.Errorf("volumes = %+v, want all three planned and only the first written", volumes)
	}
}

func TestCtrlCBetweenVolumesStopsBeforeTheNextOne(t *testing.T) {
	manga := aSeriesOfThreeVolumes(t)
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdout := &cancelOn{cancel: cancel, trigger: func(written string) bool {
		return strings.Contains(written, "wrote ") && strings.Contains(written, "Vol.01.cbz")
	}}
	var stderr bytes.Buffer

	code := runCLIContext(ctx, []string{"-input", manga, "-output", out}, stdout, &stderr)

	if code != exitInterrupted {
		t.Errorf("exit code = %d, want %d", code, exitInterrupted)
	}
	if got, want := folderListing(t, out), []string{"Series - Vol.01.cbz"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the output folder holds %v, want %v", got, want)
	}
	if !strings.Contains(stderr.String(), "Interrupted before volume 2 was written.") {
		t.Errorf("stderr does not say where it stopped:\n%s", stderr.String())
	}
	if n := strings.Count(stderr.String(), "nterrupted"); n != 1 {
		t.Errorf("the interruption is said %d times, want once:\n%s", n, stderr.String())
	}
	if strings.Contains(stderr.String(), "could not be written") || strings.Contains(stderr.String(), "context canceled") {
		t.Errorf("an interruption is reported as a failure:\n%s", stderr.String())
	}
}

func TestCtrlCBeforeAnythingIsReadWritesNothing(t *testing.T) {
	manga := aSeriesOfThreeVolumes(t)
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer

	code := runCLIContext(ctx, []string{"-input", manga, "-output", out}, &stdout, &stderr)

	if code != exitInterrupted || stdout.Len() != 0 {
		t.Errorf("code = %d, stdout = %q, want %d and nothing", code, stdout.String(), exitInterrupted)
	}
	if !strings.Contains(stderr.String(), "Interrupted while inspecting the chapters; nothing was written.") {
		t.Errorf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the output folder was created: %v", err)
	}
}

func TestCtrlCWhileACombinedSeriesIsWrittenLeavesNothing(t *testing.T) {
	manga := aSeriesOfThreeVolumes(t)
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stderr := &cancelOn{cancel: cancel, trigger: func(written string) bool {
		return strings.Contains(written, `"state":"advanced"`)
	}}
	var stdout bytes.Buffer

	code := runCLIContext(ctx, []string{"-input", manga, "-output", out, "-combine", "-json", "-progress-json"}, &stdout, stderr)

	if code != exitInterrupted {
		t.Errorf("exit code = %d, want %d", code, exitInterrupted)
	}
	if got := folderListing(t, out); len(got) != 0 {
		t.Errorf("the output folder holds %v, want it empty", got)
	}
	report := decodeReport(t, stdout.Bytes())
	found := false
	for _, issue := range report.Manga[0].Issues {
		if issue.Code == "interrupted" && strings.Contains(issue.Message, "combined series") {
			found = true
		}
	}
	if !found {
		t.Errorf("issues = %+v, want an interrupted one about the combined series", report.Manga[0].Issues)
	}
}

func TestCtrlCOnTheFinalPageDoesNotCommitOrReportCompletion(t *testing.T) {
	for _, tc := range []struct {
		name           string
		input          func(*testing.T) string
		flags          []string
		manga          string
		canceledFile   string
		files          []string
		plannedVolumes int
		writtenVolumes int
		writtenPages   int
	}{
		{
			name: "separate volumes", input: aSeriesOfThreeVolumes, manga: "Series",
			canceledFile:   "Series - Vol.03.cbz",
			files:          []string{"Series - Vol.01.cbz", "Series - Vol.02.cbz", "Series - Vol.03.cbz"},
			plannedVolumes: 3, writtenVolumes: 2, writtenPages: 4,
		},
		{
			name: "combined series", input: aSeriesOfThreeVolumes, flags: []string{"-combine"}, manga: "Series",
			canceledFile: "Series.cbz", files: []string{"Series.cbz"}, plannedVolumes: 3,
		},
		{
			name: "batch", input: aLibraryOfTwoManga, flags: []string{"-batch"}, manga: "Alpha",
			canceledFile: "Alpha - Vol.01.cbz", files: []string{"Alpha - Vol.01.cbz"}, plannedVolumes: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input(t)
			out := t.TempDir()
			canceledPath := filepath.Join(out, tc.canceledFile)
			if err := os.WriteFile(canceledPath, []byte("the good volume"), 0o644); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stderr := &cancelOn{cancel: cancel, trigger: func(written string) bool {
				var event machineProgress
				if err := json.Unmarshal([]byte(written), &event); err != nil {
					return false
				}
				return event.Stage == "write" && event.State == "advanced" && event.CompletedPages != nil &&
					*event.CompletedPages == event.TotalPages
			}}
			var stdout bytes.Buffer
			args := append([]string{"-input", input, "-output", out, "-json", "-progress-json"}, tc.flags...)

			code := runCLIContext(ctx, args, &stdout, stderr)

			if !stderr.fired || code != exitInterrupted {
				t.Errorf("cancel fired = %v, exit code = %d, want true and %d", stderr.fired, code, exitInterrupted)
			}
			if got, err := os.ReadFile(canceledPath); err != nil || string(got) != "the good volume" {
				t.Errorf("the volume is %q (err %v), want it untouched", got, err)
			}
			if got := folderListing(t, out); !reflect.DeepEqual(got, tc.files) {
				t.Errorf("the output folder holds %v, want %v: no part file or next manga", got, tc.files)
			}

			report := decodeReport(t, stdout.Bytes())
			if report.Status != "failed" || len(report.Manga) != 1 {
				t.Fatalf("report = %+v, want a failed run with one manga", report)
			}
			manga := report.Manga[0]
			if manga.Name != tc.manga || manga.Status != "failed" || len(manga.Issues) != 1 {
				t.Fatalf("manga = %+v, want %s failed with one interruption", manga, tc.manga)
			}
			issue := manga.Issues[0]
			if issue.Code != "interrupted" || issue.Severity != "error" || issue.Stage != "write" ||
				!issue.Recoverable || issue.Path != absolutePath(canceledPath) {
				t.Errorf("issue = %+v, want a recoverable write interruption at %s", issue, canceledPath)
			}
			if report.Summary.Volumes != tc.writtenVolumes || report.Summary.Pages != tc.writtenPages ||
				manga.Summary.Volumes != tc.writtenVolumes || manga.Summary.Pages != tc.writtenPages {
				t.Errorf("summaries = %+v, %+v, want only %d committed volumes and %d pages", report.Summary, manga.Summary, tc.writtenVolumes, tc.writtenPages)
			}
			if len(manga.Volumes) != tc.plannedVolumes {
				t.Fatalf("volumes = %+v, want %d planned", manga.Volumes, tc.plannedVolumes)
			}
			for i, volume := range manga.Volumes {
				if volume.Written != (i < tc.writtenVolumes) {
					t.Errorf("volume %d = %+v, want only previously committed volumes marked written", i+1, volume)
				}
			}
			completed := 0
			for _, event := range parseProgressLines(t, stderr.String()) {
				if event.Manga != tc.manga {
					t.Errorf("progress for an unexpected manga: %+v", event)
				}
				if event.Stage == "write" && event.State == "completed" {
					completed++
					if reportedPages(t, event) == event.TotalPages {
						t.Errorf("the interrupted archive was reported completed: %+v", event)
					}
				}
			}
			if completed != tc.writtenVolumes {
				t.Errorf("%d completed write events, want %d", completed, tc.writtenVolumes)
			}
		})
	}
}

func aLibraryOfTwoManga(t *testing.T) string {
	t.Helper()
	library := t.TempDir()
	for _, name := range []string{"Alpha", "Beta"} {
		writeFiles(t, filepath.Join(library, name, "Vol.01 Ch.001"), "001.jpg", "002.jpg")
	}
	return library
}

func TestCtrlCDuringABatchDoesNotStartTheNextManga(t *testing.T) {
	library := aLibraryOfTwoManga(t)
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdout := &cancelOn{cancel: cancel, trigger: func(written string) bool {
		return strings.Contains(written, "wrote ") && strings.Contains(written, "Alpha - Vol.01.cbz")
	}}
	var stderr bytes.Buffer

	code := runCLIContext(ctx, []string{"-input", library, "-batch", "-output", out}, stdout, &stderr)

	if code != exitInterrupted {
		t.Errorf("exit code = %d, want %d", code, exitInterrupted)
	}
	if got, want := folderListing(t, out), []string{"Alpha - Vol.01.cbz"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the output folder holds %v, want only %v", got, want)
	}
	if !strings.Contains(stderr.String(), interruptedBeforeNextManga) {
		t.Errorf("stderr does not say the rest was not started:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String(), "== Beta ==") {
		t.Errorf("the next manga was started:\n%s", stdout.String())
	}
}

func TestCtrlCDuringAMachineBatchReportsWhatWasDone(t *testing.T) {
	library := aLibraryOfTwoManga(t)
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stderr := &cancelOn{cancel: cancel, trigger: func(written string) bool {
		return strings.Contains(written, `"state":"completed"`) && strings.Contains(written, `"stage":"write"`)
	}}
	var stdout bytes.Buffer

	code := runCLIContext(ctx, []string{"-input", library, "-batch", "-output", out, "-json", "-progress-json"}, &stdout, stderr)

	if code != exitInterrupted {
		t.Errorf("exit code = %d, want %d", code, exitInterrupted)
	}
	report := decodeReport(t, stdout.Bytes())
	if len(report.Manga) != 1 || report.Manga[0].Name != "Alpha" || report.Manga[0].Status != "completed" {
		t.Errorf("manga = %+v, want Alpha alone, completed", report.Manga)
	}
	if len(report.Issues) != 1 || report.Issues[0].Code != "interrupted" || report.Status != "failed" {
		t.Errorf("status = %s, issues = %+v, want one invocation-level interrupted", report.Status, report.Issues)
	}
	if got, want := folderListing(t, out), []string{"Alpha - Vol.01.cbz"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the output folder holds %v, want only %v", got, want)
	}
}

func TestOnlyAnInterruptionExitsWith130(t *testing.T) {
	if got := exitCode(context.Canceled); got != 130 {
		t.Errorf("exitCode(context.Canceled) = %d, want 130", got)
	}
	if got := exitCode(errors.New("disk full")); got != 1 {
		t.Errorf("exitCode(other) = %d, want 1", got)
	}
	if got := exitCode(errors.Join(errors.New("interrupted"), context.Canceled)); got != 130 {
		t.Errorf("exitCode(wrapped) = %d, want 130", got)
	}
}

func decodeReport(t *testing.T, stdout []byte) machineReport {
	t.Helper()
	var report machineReport
	if err := json.Unmarshal(stdout, &report); err != nil {
		t.Fatalf("stdout is not one JSON report: %v\n%s", err, stdout)
	}
	return report
}

// folderListing is the names in dir, so that a test sees every file a run left
// there, a part file included.
func folderListing(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
