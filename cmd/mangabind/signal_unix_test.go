//go:build unix

package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// blockedRun is the real program, started on a chapter whose second page is a
// named pipe that nothing writes to yet, so the run waits there with the first
// page copied until the test lets it go. startBlockedRun starts it and returns
// once the first page is known to have been copied.
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

// signalHandled is how long a test waits for the program to have taken a signal
// it was sent: far longer than the microseconds that takes, so that a loaded
// machine does not turn the wait into a race.
const signalHandled = time.Second

// letTheSecondPageGo gives the named pipe its page, when the run is waiting for
// it. It never blocks on the pipe: if the run was stopped before it got there
// (it saw the interruption before it started the second page) nobody will ever
// read it, and opening it for writing in the ordinary way would wait for ever.
func (r *blockedRun) letTheSecondPageGo(t *testing.T) {
	t.Helper()
	deadline := time.After(30 * time.Second)
	for {
		pipe, err := os.OpenFile(r.fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			if _, err := io.WriteString(pipe, "two"); err != nil {
				t.Fatal(err)
			}
			pipe.Close()
			return
		}
		if !errors.Is(err, syscall.ENXIO) { // ENXIO: nobody has the pipe open for reading yet
			t.Fatal(err)
		}
		select {
		case <-r.stderrEnd: // the run is over
			return
		case <-deadline:
			t.Fatal("the run neither read its second page nor ended")
		case <-time.After(20 * time.Millisecond):
		}
	}
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
	// A signal is handled by another goroutine of the program, a moment after
	// it is sent. Give it that moment, so that what follows cannot overtake it.
	time.Sleep(signalHandled)
	run.letTheSecondPageGo(t)

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
	time.Sleep(signalHandled)
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
