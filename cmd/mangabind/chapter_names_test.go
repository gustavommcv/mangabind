package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// A folder named with the full-width digits and space a Japanese input method types is the same chapter
// as one named in ASCII (docs/adr/0020-full-width-digits-and-space-in-chapter-names.md): it is grouped
// into its volume, and two copies of one chapter in the two widths conflict, as two in ASCII do.
func TestAChapterNamedInFullWidthDigitsIsTheSameChapter(t *testing.T) {
	manga := filepath.Join(t.TempDir(), "Series")
	for _, name := range []string{
		"Vol.０１ Ch.００１",
		"Vol.０１ Ch.００２",
		"Vol.０２　Ch.００３",
		"Vol.01 Ch.002",
	} {
		writeFiles(t, filepath.Join(manga, name), "001.jpg")
	}
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := runText("-input", manga, "-output", out, "-dry-run")

	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	for _, volume := range []string{"Series - Vol.01.cbz", "Series - Vol.02.cbz"} {
		if !strings.Contains(stdout, filepath.Join(out, volume)) {
			t.Errorf("stdout does not say %s would be written:\n%s", volume, stdout)
		}
	}
	if !strings.Contains(stderr, "volume 1 chapter 2 has 2 conflicting sources") {
		t.Errorf("the chapter in both widths is not a conflict:\n%s", stderr)
	}
	if strings.Contains(stderr, "could not be read") || strings.Contains(stderr, "could not parse chapter name") {
		t.Errorf("a folder was not read:\n%s", stderr)
	}
}
