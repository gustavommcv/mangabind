package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultOutputDir(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{
			input: filepath.Join("C:", "Users", "me", "Manga", "Chainsaw Man"),
			want:  filepath.Join("C:", "Users", "me", "Manga", "Chainsaw Man (mangabind)"),
		},
		{
			input: filepath.Join("manga", "One Piece"),
			want:  filepath.Join("manga", "One Piece (mangabind)"),
		},
	}
	for _, c := range cases {
		if got := defaultOutputDir(c.input); got != c.want {
			t.Errorf("defaultOutputDir(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// makeChapter creates a chapter folder with n placeholder pages under root.
func makeChapter(t *testing.T, root, name string, n int) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		name := fmt.Sprintf("%02d.jpg", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func countEntries(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return len(entries)
}

func TestProcessMangaEndToEnd(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "Chainsaw Man")
	makeChapter(t, input, "Vol.01 Ch.0001 - Alpha (en) [Group]", 2)
	makeChapter(t, input, "Vol.01 Ch.0002 - Beta (en) [Group]", 3)

	output := filepath.Join(root, "out")
	summary, err := processManga(input, output, "", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.volumes != 1 || summary.pages != 5 {
		t.Fatalf("summary = %+v, want 1 volume, 5 pages", summary)
	}

	outPath := filepath.Join(output, "Chainsaw Man - Vol.01.cbz")
	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("opening %s: %v", outPath, err)
	}
	defer zr.Close()
	if len(zr.File) != 5 {
		t.Fatalf("got %d pages in archive, want 5", len(zr.File))
	}
}

func TestProcessMangaDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "Chainsaw Man")
	makeChapter(t, input, "Vol.01 Ch.0001 - Alpha (en) [Group]", 1)

	output := filepath.Join(root, "out")
	summary, err := processManga(input, output, "", true, true /* dryRun */)
	if err != nil {
		t.Fatal(err)
	}
	if summary.volumes != 1 {
		t.Fatalf("summary = %+v, want 1 volume counted even in dry-run", summary)
	}
	if countEntries(t, output) != 0 {
		t.Fatal("dry-run should not have written any files")
	}
}

func writeMetadataFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "mangabind.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProcessMangaMetadataFillsMissingVolume(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "Some Manga")
	// "Chapter N" carries no volume on its own - see ChapterOnlyParser.
	makeChapter(t, input, "Chapter 1", 1)
	makeChapter(t, input, "Chapter 2", 1)
	makeChapter(t, input, "Chapter 8", 1)
	writeMetadataFile(t, input, `{
		"schema_version": 1,
		"volumes": [
			{"number": "1", "chapters": ["1-2"]},
			{"number": "2", "chapters": ["8"]}
		]
	}`)

	output := filepath.Join(root, "out")
	// No explicit -metadata-file: relies on the mangabind.json convention.
	summary, err := processManga(input, output, "", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.volumes != 2 {
		t.Fatalf("summary = %+v, want 2 volumes (metadata should have resolved them)", summary)
	}
	for _, want := range []string{"Some Manga - Vol.01.cbz", "Some Manga - Vol.02.cbz"} {
		if _, err := os.Stat(filepath.Join(output, want)); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}

func TestProcessMangaMetadataNeverOverridesFilename(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "Some Manga")
	makeChapter(t, input, "Vol.01 Ch.0001 - Alpha (en) [Group]", 1)
	metaPath := writeMetadataFile(t, root, `{
		"schema_version": 1,
		"volumes": [{"number": "2", "chapters": ["1"]}]
	}`)

	output := filepath.Join(root, "out")
	summary, err := processManga(input, output, metaPath, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.volumes != 1 {
		t.Fatalf("summary = %+v, want 1 volume", summary)
	}
	// The filename said volume 1; metadata disagreeing (volume 2) must not
	// change that - only warn (checked manually via -v; not asserted here
	// since it goes to stderr, not the return value).
	if _, err := os.Stat(filepath.Join(output, "Some Manga - Vol.01.cbz")); err != nil {
		t.Errorf("expected the filename's volume (1) to win: %v", err)
	}
}

func TestParseFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want cliConfig
	}{
		{
			name: "long flags",
			args: []string{"-input", "in", "-output", "out", "-batch", "-quiet", "-dry-run"},
			want: cliConfig{input: "in", output: "out", batch: true, quiet: true, dryRun: true},
		},
		{
			name: "short aliases behave the same as their long form",
			args: []string{"-i", "in", "-o", "out", "-q", "-n"},
			want: cliConfig{input: "in", output: "out", quiet: true, dryRun: true},
		},
		{
			name: "version",
			args: []string{"-version"},
			want: cliConfig{showVersion: true},
		},
		{
			name: "metadata file",
			args: []string{"-input", "in", "-metadata-file", "volumes.json"},
			want: cliConfig{input: "in", metadataFile: "volumes.json"},
		},
		{
			name: "no args",
			args: nil,
			want: cliConfig{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _, err := parseFlags(c.args)
			if err != nil {
				t.Fatalf("parseFlags(%v) error = %v", c.args, err)
			}
			if got != c.want {
				t.Fatalf("parseFlags(%v) = %+v, want %+v", c.args, got, c.want)
			}
		})
	}
}

func TestParseFlagsHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"-help"}} {
		_, _, err := parseFlags(args)
		if !errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseFlags(%v) error = %v, want flag.ErrHelp", args, err)
		}
	}
}

func TestParseFlagsRejectsUnknownFlag(t *testing.T) {
	_, _, err := parseFlags([]string{"-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected an error for an unrecognized flag")
	}
}

func TestUnsupportedFileMessage(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"chapter 5.epub", `found "chapter 5.epub" - .epub chapters aren't supported (only folders of images or .cbz), skipped`},
		{"chapter 5.pdf", `found "chapter 5.pdf" - .pdf chapters aren't supported (only folders of images or .cbz), skipped`},
		{"notes.txt", `found unrecognized file, skipped: "notes.txt"`},
	}
	for _, c := range cases {
		if got := unsupportedFileMessage(c.name); got != c.want {
			t.Errorf("unsupportedFileMessage(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSummarizeNames(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := summarizeNames("no volume", nil, ""); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})

	t.Run("at or below the cap: one line per name", func(t *testing.T) {
		names := []string{"Chapter 1", "Chapter 2"}
		want := []string{`no volume: "Chapter 1"`, `no volume: "Chapter 2"`}
		if got := summarizeNames("no volume", names, ""); !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("over the cap: truncated with a summary line", func(t *testing.T) {
		names := make([]string, 200)
		for i := range names {
			names[i] = fmt.Sprintf("Chapter %d", i+1)
		}
		got := summarizeNames("no volume", names, "see -metadata-file")
		if len(got) != maxIndividualWarnings+1 {
			t.Fatalf("got %d lines, want %d", len(got), maxIndividualWarnings+1)
		}
		for i := 0; i < maxIndividualWarnings; i++ {
			want := fmt.Sprintf("no volume: %q", names[i])
			if got[i] != want {
				t.Fatalf("line %d = %q, want %q", i, got[i], want)
			}
		}
		wantLast := "... and 190 more (see -metadata-file)"
		if got[len(got)-1] != wantLast {
			t.Fatalf("last line = %q, want %q", got[len(got)-1], wantLast)
		}
	})
}

func TestRunBatchProcessesEveryMangaIndependently(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "Library")

	makeChapter(t, filepath.Join(library, "Chainsaw Man"), "Vol.01 Ch.0001 - Alpha (en) [Group]", 1)
	makeChapter(t, filepath.Join(library, "One Piece"), "Vol.01 Ch.0001 - Beta (en) [Group]", 2)
	makeChapter(t, filepath.Join(library, "One Piece"), "Vol.01 Ch.0002 - Gamma (en) [Group]", 1)

	output := filepath.Join(root, "out")
	if err := runBatch(library, output, true, false); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"Chainsaw Man - Vol.01.cbz", "One Piece - Vol.01.cbz"} {
		if _, err := os.Stat(filepath.Join(output, want)); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}
