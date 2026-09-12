package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	summary, err := processManga(input, output, true, false)
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
	summary, err := processManga(input, output, true, true /* dryRun */)
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
