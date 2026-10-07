package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runText runs the command line the way a person's shell would, and returns the
// exit code and what went to each stream.
func runText(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = runCLI(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// writeTestManga makes a manga folder with one chapter of one page, and returns
// its path. The folder's default output, "<name> (mangabind)", is a sibling in
// the same temporary directory.
func writeTestManga(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Manga")
	chapter := filepath.Join(root, "Vol.01 Ch.001")
	if err := os.MkdirAll(chapter, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chapter, "001.jpg"), []byte("page"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func lineCount(text string) int {
	return len(strings.Split(strings.TrimRight(text, "\n"), "\n"))
}

func TestStrayArgumentIsRefusedAndNothingIsWritten(t *testing.T) {
	cases := []struct {
		name string
		args func(manga, out string) []string
		// whether the message should say that flags after the argument were skipped
		flagsSkipped bool
	}{
		{
			name: "before the flags that follow it, with -dry-run among them",
			args: func(manga, out string) []string {
				return []string{"-input", manga, "extra", "-dry-run", "-output", out}
			},
			flagsSkipped: true,
		},
		{
			name: "after every flag",
			args: func(manga, out string) []string {
				return []string{"-input", manga, "-output", out, "extra"}
			},
		},
		{
			name: "after the end of the flags",
			args: func(manga, out string) []string {
				return []string{"-input", manga, "-output", out, "--", "extra"}
			},
		},
		{
			name: "a path with an unquoted space",
			args: func(manga, out string) []string {
				return []string{"-input", manga, "Two", "-n", "-o", out}
			},
			flagsSkipped: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			manga := writeTestManga(t)
			out := filepath.Join(t.TempDir(), "out")

			code, stdout, stderr := runText(c.args(manga, out)...)

			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if !strings.Contains(stderr, "unexpected argument") {
				t.Errorf("stderr does not name the argument:\n%s", stderr)
			}
			if got := strings.Contains(stderr, "flags after it were not read"); got != c.flagsSkipped {
				t.Errorf("says the flags after it were skipped = %v, want %v\n%s", got, c.flagsSkipped, stderr)
			}
			if n := lineCount(stderr); n > 5 {
				t.Errorf("the message is %d lines long, want a few:\n%s", n, stderr)
			}
			for _, path := range []string{out, manga + " (mangabind)"} {
				if exists(path) {
					t.Errorf("%s was created by a run that was refused", path)
				}
			}
		})
	}
}

func TestStrayArgumentWithJSONGetsAReport(t *testing.T) {
	manga := writeTestManga(t)

	// -json comes after the argument, where the flag package never reads it.
	report, stderr, code := runMachineForTest(t, "-input", manga, "extra", "-json")

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if report.Status != "failed" || len(report.Issues) != 1 {
		t.Fatalf("report = %+v, want one failing issue", report)
	}
	got := report.Issues[0]
	if got.Code != "invalid_arguments" || got.Stage != "configuration" || !strings.Contains(got.Diagnostic, `unexpected argument "extra"`) {
		t.Errorf("issue = %+v, want invalid_arguments naming the argument", got)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing: stdout carries the answer", stderr)
	}
}

func TestParseFlagsRefusesAStrayArgument(t *testing.T) {
	_, _, err := parseFlags([]string{"-input", "in", "extra", "-dry-run"})

	var unexpected *unexpectedArgumentError
	if !errors.As(err, &unexpected) {
		t.Fatalf("error = %v, want an unexpectedArgumentError", err)
	}
	if strings.Join(unexpected.args, " ") != "extra -dry-run" {
		t.Errorf("args = %q, want the argument and everything after it", unexpected.args)
	}
}

func TestMachineOutputIsRequestedInEverySpellingOfTheFlag(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"-json"}, true},
		{[]string{"--json"}, true},
		{[]string{"-json=true"}, true},
		{[]string{"--json=true"}, true},
		{[]string{"-json=1"}, true},
		{[]string{"-json=false"}, false},
		{[]string{"-json=0"}, false},
		{[]string{"-json=maybe"}, false},
		{[]string{"-jsonx"}, false},
		{[]string{"json"}, false},
		{[]string{"-json=false", "-json"}, true},
		{[]string{"--", "-json"}, false},
		{nil, false},
	}
	for _, c := range cases {
		if got := flagRequested(c.args, "json"); got != c.want {
			t.Errorf("flagRequested(%q, json) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestInvalidArgumentsAreReportedAsJSONOnlyWhenAskedFor(t *testing.T) {
	for _, spelling := range []string{"-json", "--json", "-json=true", "--json=true"} {
		t.Run(spelling, func(t *testing.T) {
			report, _, code := runMachineForTest(t, spelling, "-not-a-flag")
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if len(report.Issues) != 1 || report.Issues[0].Code != "invalid_arguments" {
				t.Errorf("issues = %+v, want one invalid_arguments", report.Issues)
			}
		})
	}
	for _, spelling := range []string{"-json=false", "-json=0"} {
		t.Run(spelling, func(t *testing.T) {
			code, stdout, stderr := runText(spelling, "-not-a-flag")
			if code != 2 || stdout != "" || !strings.Contains(stderr, "unknown flag -not-a-flag") {
				t.Errorf("code = %d, stdout = %q, stderr = %q, want a human error on stderr", code, stdout, stderr)
			}
		})
	}
}

func TestHelpIsAskedForAndGoesToStdout(t *testing.T) {
	for _, flagName := range []string{"-h", "--help", "-help"} {
		t.Run(flagName, func(t *testing.T) {
			code, stdout, stderr := runText(flagName)
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
			for _, want := range []string{"Usage:", "Examples:", "Flags:", "-input", "-dry-run", "Exit codes:", "130  interrupted", "Output:", "https://github.com/gustavommcv/mangabind"} {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout is missing %q", want)
				}
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want nothing", stderr)
			}
		})
	}
}

func TestAMistakeGetsAShortMessageNotTheWholeHelp(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"-not-a-flag"}, "mangabind: unknown flag -not-a-flag"},
		{"flag without its value", []string{"-input"}, "needs an argument"},
		{"bad boolean", []string{"-batch=maybe"}, "invalid boolean value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, stdout, stderr := runText(c.args...)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if !strings.Contains(stderr, c.want) {
				t.Errorf("stderr is missing %q:\n%s", c.want, stderr)
			}
			if !strings.Contains(stderr, "Try 'mangabind -h'") {
				t.Errorf("stderr does not point to the help:\n%s", stderr)
			}
			if strings.Contains(stderr, "Flags:") || lineCount(stderr) > 3 {
				t.Errorf("the whole help was printed for a mistake:\n%s", stderr)
			}
		})
	}
}

func TestAMistypedFlagSuggestsTheFlagItWasMeantToBe(t *testing.T) {
	cases := []struct {
		typed string
		want  string
	}{
		{"inupt", "input"},
		{"inpt", "input"},
		{"dryrun", "dry-run"},
		{"outptu", "output"},
		{"verson", "version"},
		{"batchh", "batch"},
		{"metadata-fle", "metadata-file"},
		{"progress-jsn", "progress-json"},
		// Too far from anything to be a slip, or too short to tell: no guess.
		{"zzzzzz", ""},
		{"jsno", ""},
		{"x", ""},
	}
	for _, c := range cases {
		_, _, err := parseFlags([]string{"-" + c.typed})
		if err == nil {
			t.Fatalf("-%s was accepted", c.typed)
		}
		var out bytes.Buffer
		fs := flagSetForTest(t)
		printUsageError(&out, fs, err)
		message := out.String()

		suggested := strings.Contains(message, "did you mean")
		if c.want == "" && suggested {
			t.Errorf("-%s: a guess was made where there should be none:\n%s", c.typed, message)
		}
		if c.want != "" && !strings.Contains(message, "(did you mean -"+c.want+"?)") {
			t.Errorf("-%s: want a suggestion of -%s:\n%s", c.typed, c.want, message)
		}
	}
}

func flagSetForTest(t *testing.T) *flag.FlagSet {
	t.Helper()
	_, fs, _ := parseFlags(nil)
	return fs
}

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"input", "input", 0},
		{"input", "inupt", 2},
		{"input", "inpt", 1},
		{"dryrun", "dry-run", 1},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
	}
	for _, c := range cases {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestRunningWithNothingShowsHowToUseIt(t *testing.T) {
	code, stdout, stderr := runText()

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	for _, want := range []string{"Usage:", "Examples:"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr is missing %q", want)
		}
	}
}

func TestAMissingInputWithOtherFlagsGetsAShortMessage(t *testing.T) {
	code, stdout, stderr := runText("-batch")

	if code != 2 || stdout != "" {
		t.Errorf("code = %d, stdout = %q, want 2 and nothing", code, stdout)
	}
	if !strings.Contains(stderr, "no -input given") || !strings.Contains(stderr, "Try 'mangabind -h'") || lineCount(stderr) > 3 {
		t.Errorf("stderr is not a short message that points to the help:\n%s", stderr)
	}
}

func TestNoticesGoToStderrAndTheResultToStdout(t *testing.T) {
	manga := writeTestManga(t)
	if err := os.WriteFile(filepath.Join(manga, "mangabind.json"), []byte(`{"schema_version":1,"volumes":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runText("-input", manga)

	if code != 0 {
		t.Fatalf("exit code = %d, stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "wrote ") {
		t.Errorf("stdout = %q, want the line that says what was written", stdout)
	}
	for _, notice := range []string{"no -output given, writing to", "using metadata file"} {
		if strings.Contains(stdout, notice) {
			t.Errorf("stdout carries the notice %q, which is not the result", notice)
		}
		if !strings.Contains(stderr, notice) {
			t.Errorf("stderr is missing the notice %q:\n%s", notice, stderr)
		}
	}
}

func TestAFolderWithNoChaptersIsAWarningThatQuietDoesNotHide(t *testing.T) {
	for _, extra := range [][]string{nil, {"-quiet"}} {
		empty := filepath.Join(t.TempDir(), "Empty")
		if err := os.MkdirAll(empty, 0o755); err != nil {
			t.Fatal(err)
		}
		args := append([]string{"-input", empty, "-output", filepath.Join(t.TempDir(), "out")}, extra...)

		code, stdout, stderr := runText(args...)

		if code != 0 {
			t.Errorf("%v: exit code = %d, want 0 (nothing to do is not a failure)", extra, code)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want nothing", extra, stdout)
		}
		if !strings.Contains(stderr, "warning: no chapter folders or .cbz files found in "+empty) {
			t.Errorf("%v: stderr does not warn:\n%s", extra, stderr)
		}
	}
}
