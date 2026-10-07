//go:build unix

package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// aRunBlockedOnItsSecondPage starts the real program on a chapter whose second
// page is a named pipe that nothing writes to yet, so the run stops there, with
// the first page copied, until the test lets it go. It returns when the first
// page is known to have been copied.
type blockedRun struct {
	cmd       *exec.Cmd
	stdout    *bytes.Buffer
	fifo      string
	out       string
	previous  string
	stderrEnd chan struct{}
}

func startBlockedRun(t *testing.T) *blockedRun {
	t.Helper()
	root := t.TempDir()
	chapter := filepath.Join(root, "Series", "Vol.01 Ch.001")
	writeFiles(t, chapter, "001.jpg", "003.jpg")
	fifo := filepath.Join(chapter, "002.jpg")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("cannot make a named pipe here: %v", err)
	}
	out := filepath.Join(root, "out")
	writeFiles(t, out) // the folder
	previous := filepath.Join(out, "Series - Vol.01.cbz")
	if err := os.WriteFile(previous, []byte("the previous volume"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-input", filepath.Join(root, "Series"), "-output", out, "-json", "-progress-json")
	cmd.Env = append(os.Environ(), "MANGABIND_RUN_MAIN=1")
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	watchdog := time.AfterFunc(60*time.Second, func() { _ = cmd.Process.Kill() })
	t.Cleanup(func() { watchdog.Stop(); _ = cmd.Process.Kill() })

	run := &blockedRun{cmd: cmd, stdout: stdout, fifo: fifo, out: out, previous: previous, stderrEnd: make(chan struct{})}
	copied := make(chan struct{})
	go func() {
		defer close(run.stderrEnd)
		announced := false
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if !announced && strings.Contains(scanner.Text(), `"state":"advanced"`) {
				announced = true
				close(copied)
			}
		}
		if !announced {
			close(copied)
		}
	}()
	select {
	case <-copied:
	case <-time.After(30 * time.Second):
		t.Fatal("the run never copied its first page")
	}
	return run
}

// finish waits for the run to end, after its stderr has been read to the end.
func (r *blockedRun) finish(t *testing.T) *os.ProcessState {
	t.Helper()
	select {
	case <-r.stderrEnd:
	case <-time.After(30 * time.Second):
		t.Fatal("the run did not end")
	}
	_ = r.cmd.Wait()
	return r.cmd.ProcessState
}

func TestCtrlCStopsTheRealProgramAndLeavesNothingHalfWritten(t *testing.T) {
	run := startBlockedRun(t)

	if err := run.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	// Let the page it is blocked on go: the run copies it, sees that it was
	// asked to stop before the third page, and cleans up.
	pipe, err := os.OpenFile(run.fifo, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(pipe, "two"); err != nil {
		t.Fatal(err)
	}
	pipe.Close()

	state := run.finish(t)

	if state.ExitCode() != exitInterrupted {
		t.Errorf("exit code = %d, want %d\nstdout: %s", state.ExitCode(), exitInterrupted, run.stdout.String())
	}
	report := decodeReport(t, run.stdout.Bytes())
	found := false
	for _, issue := range report.Manga[0].Issues {
		found = found || issue.Code == "interrupted"
	}
	if !found {
		t.Errorf("the report has no interrupted issue: %+v", report.Manga[0].Issues)
	}
	// The volume that was there is as it was, and no part file is left.
	got, err := os.ReadFile(run.previous)
	if err != nil || string(got) != "the previous volume" {
		t.Errorf("the previous volume is %q (err %v), want it untouched", got, err)
	}
	if listing := folderListing(t, run.out); len(listing) != 1 {
		t.Errorf("the output folder holds %v, want only the previous volume", listing)
	}
}

func TestASecondCtrlCEndsTheRealProgramAtOnce(t *testing.T) {
	run := startBlockedRun(t)

	if err := run.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	// The first interrupt cancels the run and lets the handler go; give that a
	// moment, then the second one is not caught.
	time.Sleep(500 * time.Millisecond)
	if err := run.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}

	state := run.finish(t)

	status, ok := state.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGINT {
		t.Errorf("the program ended with %v, want it killed by the second SIGINT", state)
	}
	// Even killed, it cannot have damaged the volume that was there.
	got, err := os.ReadFile(run.previous)
	if err != nil || string(got) != "the previous volume" {
		t.Errorf("the previous volume is %q (err %v), want it untouched", got, err)
	}
}
