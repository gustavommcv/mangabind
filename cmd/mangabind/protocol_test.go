package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func runMachineForTest(t *testing.T, args ...string) (machineReport, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(args, &stdout, &stderr)

	var report machineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("machine stdout is not one JSON document: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return report, stderr.String(), exitCode
}

func realFixturePath(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMachineProtocolVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exitCode := runCLI([]string{"--protocol-version"}, &stdout, &stderr); exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	var got protocolInfo
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("protocol information is not JSON: %v", err)
	}
	if got.ProtocolVersion != machineProtocolVersion || got.Tool != "mangabind" || got.ToolVersion == "" {
		t.Fatalf("unexpected protocol information: %+v", got)
	}
	if len(got.Capabilities) != 2 || got.Capabilities[0] != "report" || got.Capabilities[1] != "progress-json" {
		t.Fatalf("capabilities = %v, want [report progress-json]", got.Capabilities)
	}
}

func parseProgressLines(t *testing.T, output string) []machineProgress {
	t.Helper()
	var events []machineProgress
	for _, line := range bytes.Split(bytes.TrimSpace([]byte(output)), []byte("\n")) {
		var event machineProgress
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatalf("progress line is not JSON: %v\nline: %s", err, line)
		}
		if event.ProtocolVersion != machineProtocolVersion || event.Tool != "mangabind" || event.Kind != "progress" {
			t.Fatalf("invalid progress envelope: %+v", event)
		}
		events = append(events, event)
	}
	return events
}

func reportedPages(t *testing.T, event machineProgress) int {
	t.Helper()
	if event.CompletedPages == nil {
		t.Fatalf("write event omitted completed_pages: %+v", event)
	}
	return *event.CompletedPages
}

func TestMachineProgressUsesRealFixtureWithoutChangingFinalReport(t *testing.T) {
	var stdout, stderr bytes.Buffer
	output := filepath.Join(t.TempDir(), "out")
	if code := runCLI([]string{"--input", realFixturePath(t, "sample_manga"), "--output", output, "--json", "--progress-json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var report machineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("final stdout report changed: %v", err)
	}
	if report.Summary.Pages != 4 || !report.Manga[0].Volumes[0].Written {
		t.Fatalf("unexpected final report: %+v", report)
	}
	events := parseProgressLines(t, stderr.String())
	if events[0].Stage != "inspect" || events[0].State != "started" || events[1].State != "completed" {
		t.Fatalf("unexpected inspection progress: %+v", events)
	}
	write := events[2:]
	if write[0].State != "started" || write[0].VolumeIndex != 1 || write[0].VolumeCount != 1 || write[0].TotalPages != 4 {
		t.Fatalf("unexpected write start: %+v", write[0])
	}
	if reportedPages(t, write[0]) != 0 {
		t.Fatalf("write start should report zero copied pages: %+v", write[0])
	}
	for index := 1; index <= 4; index++ {
		if write[index].State != "advanced" || reportedPages(t, write[index]) != index {
			t.Fatalf("page %d progress = %+v", index, write[index])
		}
	}
	if last := write[len(write)-1]; last.State != "completed" || reportedPages(t, last) != 4 {
		t.Fatalf("unexpected final progress: %+v", last)
	}
}

func TestMachineProgressDryRunNeverClaimsPagesWereWritten(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"--input", realFixturePath(t, "sample_manga"), "--output", filepath.Join(t.TempDir(), "out"), "--dry-run", "--json", "--progress-json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	for _, event := range parseProgressLines(t, stderr.String()) {
		if event.Stage == "write" {
			t.Fatalf("dry-run claimed write progress: %+v", event)
		}
	}
}

func TestMachineProgressRequiresJsonMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"--progress-json"}, &stdout, &stderr); code != 2 || !bytes.Contains(stderr.Bytes(), []byte("requires -json")) {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
}

func TestMachineProgressKeepsZeroVolumeNumber(t *testing.T) {
	var output bytes.Buffer
	volume, pages := 0.0, 0
	newProgressSink(&output)(machineProgress{
		Stage: "write", State: "started", Manga: "A Work",
		VolumeIndex: 1, VolumeCount: 1, VolumeNumber: &volume,
		CompletedPages: &pages, TotalPages: 1,
	})
	events := parseProgressLines(t, output.String())
	if len(events) != 1 || events[0].VolumeNumber == nil || *events[0].VolumeNumber != 0 || reportedPages(t, events[0]) != 0 {
		t.Fatalf("zero values were lost: %+v", events)
	}
}

type rejectingProgressWriter struct{}

func (rejectingProgressWriter) Write([]byte) (int, error) {
	return 0, errors.New("progress sink unavailable")
}

func TestMachineProgressSinkFailureDoesNotFailArchive(t *testing.T) {
	output := filepath.Join(t.TempDir(), "out")
	report, err := runMachine(cliConfig{
		input:  realFixturePath(t, "sample_manga"),
		output: output,
	}, output, newProgressSink(rejectingProgressWriter{}))
	if err != nil || report.Manga[0].Volumes[0].Written != true {
		t.Fatalf("progress sink affected the final result: %v, %+v", err, report)
	}
	if _, err := os.Stat(report.Manga[0].Volumes[0].OutputPath); err != nil {
		t.Fatalf("archive was not written: %v", err)
	}
}

func TestMachineCombinedProgressFollowsSourceVolumes(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "Series")
	makeChapter(t, input, "Vol.01 Ch.0001 - First", 2)
	makeChapter(t, input, "Vol.02 Ch.0002 - Second", 1)
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"--input", input, "--output", filepath.Join(root, "out"), "--combine", "--json", "--progress-json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	events := parseProgressLines(t, stderr.String())
	var advanced []machineProgress
	for _, event := range events {
		if event.Stage == "write" && event.State == "advanced" {
			advanced = append(advanced, event)
		}
	}
	if len(advanced) != 3 || advanced[0].VolumeIndex != 1 || advanced[1].VolumeIndex != 1 || advanced[2].VolumeIndex != 2 {
		t.Fatalf("combined progress did not follow volumes: %+v", advanced)
	}
	if last := events[len(events)-1]; last.State != "completed" || reportedPages(t, last) != 3 || last.VolumeCount != 2 {
		t.Fatalf("combined archive did not complete: %+v", last)
	}
	var report machineReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Manga[0].CombinedOutputPath == "" {
		t.Fatalf("unexpected combined report: %v, %+v", err, report)
	}
}

func TestMachineBatchProgressNamesEachMangaAndResetsCounts(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "Library")
	makeChapter(t, filepath.Join(input, "Alpha"), "Vol.01 Ch.0001 - First", 1)
	makeChapter(t, filepath.Join(input, "Beta"), "Vol.01 Ch.0001 - First", 2)
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"--input", input, "--output", filepath.Join(root, "out"), "--batch", "--json", "--progress-json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var completed []machineProgress
	for _, event := range parseProgressLines(t, stderr.String()) {
		if event.Stage == "write" && event.State == "completed" {
			completed = append(completed, event)
		}
	}
	if len(completed) != 2 || completed[0].Manga != "Alpha" || reportedPages(t, completed[0]) != 1 || completed[1].Manga != "Beta" || reportedPages(t, completed[1]) != 2 {
		t.Fatalf("batch progress = %+v", completed)
	}
}

func TestMachinePlanUsesRealFixtureAndStableOrdering(t *testing.T) {
	fixture := realFixturePath(t, "sample_manga")
	output := filepath.Join(t.TempDir(), "planned output")
	report, stderr, exitCode := runMachineForTest(t,
		"--input", fixture,
		"--output", output,
		"--dry-run",
		"--json",
	)

	if exitCode != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr)
	}
	if report.ProtocolVersion != 1 || report.Mode != "plan" || report.Status != "completed_with_warnings" {
		t.Fatalf("unexpected report header: %+v", report)
	}
	if len(report.Manga) != 1 {
		t.Fatalf("manga count = %d, want 1", len(report.Manga))
	}

	manga := report.Manga[0]
	if manga.Summary.Units != 3 || manga.Summary.Volumes != 1 || manga.Summary.Pages != 4 {
		t.Fatalf("unexpected real-fixture summary: %+v", manga.Summary)
	}
	wantUnits := []string{
		"Chapter 3",
		"Vol.01 Ch.0001 - Title One (en) [GroupA]",
		"Vol.01 Ch.0002 - Title Two (en) [GroupA]",
	}
	for i, want := range wantUnits {
		if manga.Units[i].Name != want {
			t.Fatalf("unit %d = %q, want %q", i, manga.Units[i].Name, want)
		}
	}
	if manga.Units[0].Parser.Name != "chapter-only" || manga.Units[0].Disposition != "unassigned" {
		t.Fatalf("unexpected unassigned unit: %+v", manga.Units[0])
	}
	if manga.Volumes[0].Written || filepath.Base(manga.Volumes[0].OutputPath) != "sample_manga - Vol.01.cbz" {
		t.Fatalf("unexpected planned volume: %+v", manga.Volumes[0])
	}
	if len(manga.Issues) != 1 || manga.Issues[0].Code != "unassigned_chapter" || manga.Issues[0].Chapter == nil || *manga.Issues[0].Chapter != 3 {
		t.Fatalf("unexpected issues: %+v", manga.Issues)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("planning mode wrote to %s", output)
	}
}

func TestMachinePlanReportsRealFixtureGapsAndConflicts(t *testing.T) {
	report, _, exitCode := runMachineForTest(t,
		"--input", realFixturePath(t, "sample_manga_conflicts"),
		"--output", filepath.Join(t.TempDir(), "out"),
		"--dry-run",
		"--json",
	)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, report = %+v", exitCode, report)
	}

	manga := report.Manga[0]
	var gap, conflict *machineIssue
	for i := range manga.Issues {
		switch manga.Issues[i].Code {
		case "chapter_gap":
			gap = &manga.Issues[i]
		case "chapter_conflict":
			conflict = &manga.Issues[i]
		}
	}
	if gap == nil || gap.Volume == nil || *gap.Volume != 1 {
		t.Fatalf("missing volume-one gap: %+v", manga.Issues)
	}
	if conflict == nil || conflict.Chapter == nil || *conflict.Chapter != 5 || len(conflict.RelatedPaths) != 2 {
		t.Fatalf("missing chapter-five conflict: %+v", manga.Issues)
	}
	for _, unit := range manga.Units {
		if unit.Parser.Chapter != nil && *unit.Parser.Chapter == 5 && unit.Disposition != "conflict" {
			t.Fatalf("conflicting unit disposition = %q, want conflict", unit.Disposition)
		}
	}
}

func TestMachinePlanReportsMetadataAssignmentAgainstRealFixture(t *testing.T) {
	metadataPath := filepath.Join(t.TempDir(), "mapping.json")
	if err := os.WriteFile(metadataPath, []byte(`{
  "schema_version": 1,
  "volumes": [{"number": "2", "chapters": ["3"]}],
  "source": {"provider": "test", "id": "real-fixture"}
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, _, exitCode := runMachineForTest(t,
		"--input", realFixturePath(t, "sample_manga"),
		"--output", filepath.Join(t.TempDir(), "out"),
		"--metadata-file", metadataPath,
		"--dry-run",
		"--json",
	)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, report = %+v", exitCode, report)
	}
	unit := report.Manga[0].Units[0]
	if unit.MetadataAssignment.Status != "applied" || unit.MetadataAssignment.MetadataVolume == nil || *unit.MetadataAssignment.MetadataVolume != 2 {
		t.Fatalf("unexpected metadata assignment: %+v", unit.MetadataAssignment)
	}
	if unit.EffectiveVolume == nil || *unit.EffectiveVolume != 2 || unit.Disposition != "included" {
		t.Fatalf("unexpected effective assignment: %+v", unit)
	}
}

func TestMachineExecutionReportsWrittenRealFixture(t *testing.T) {
	output := filepath.Join(t.TempDir(), "written output")
	report, stderr, exitCode := runMachineForTest(t,
		"--input", realFixturePath(t, "sample_manga"),
		"--output", output,
		"--json",
	)
	if exitCode != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr)
	}
	volume := report.Manga[0].Volumes[0]
	if report.Mode != "execute" || !volume.Written {
		t.Fatalf("unexpected execution report: %+v", report)
	}
	if _, err := os.Stat(volume.OutputPath); err != nil {
		t.Fatalf("reported output does not exist: %v", err)
	}
}

func TestMachineBatchKeepsMangaFailuresIsolated(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "Library")
	makeChapter(t, filepath.Join(library, "Good Manga"), "Vol.01 Ch.0001 - Good", 1)
	badManga := filepath.Join(library, "Bad Manga")
	if err := os.MkdirAll(badManga, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badManga, "Vol.01 Ch.0001 - Broken.cbz"), []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(root, "out")
	report, stderr, exitCode := runMachineForTest(t,
		"--input", library,
		"--output", output,
		"--batch",
		"--json",
	)
	if exitCode != 1 || stderr != "" || report.Status != "failed" {
		t.Fatalf("exit code = %d, stderr = %q, status = %q", exitCode, stderr, report.Status)
	}
	if len(report.Manga) != 2 || report.Manga[0].Name != "Bad Manga" || report.Manga[0].Status != "failed" || report.Manga[1].Status != "completed" {
		t.Fatalf("unexpected per-manga reports: %+v", report.Manga)
	}
	if _, err := os.Stat(filepath.Join(output, "Good Manga - Vol.01.cbz")); err != nil {
		t.Fatalf("good manga was not written after another manga failed: %v", err)
	}
}

func TestMachineErrorsRemainStructured(t *testing.T) {
	report, _, exitCode := runMachineForTest(t, "--json")
	if exitCode != 2 || report.Status != "failed" {
		t.Fatalf("exit code = %d, status = %q", exitCode, report.Status)
	}
	if len(report.Issues) != 1 || report.Issues[0].Code != "missing_input" || !report.Issues[0].Recoverable {
		t.Fatalf("unexpected structured error: %+v", report.Issues)
	}
}
