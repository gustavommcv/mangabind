package cbz

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

// folderListing is the names in dir, so that a test sees every file a write
// left there, the part file included.
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

// plainPages makes n source files and the pages that copy them.
func plainPages(t *testing.T, dir string, n int) []grouper.Page {
	t.Helper()
	var pages []grouper.Page
	for i := 1; i <= n; i++ {
		src := filepath.Join(dir, "src", strings.Repeat("x", i)+".jpg")
		if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(src, []byte("page"), 0o644); err != nil {
			t.Fatal(err)
		}
		pages = append(pages, grouper.Page{SourcePath: src, ArchiveName: "c001/p" + string(rune('0'+i)) + ".jpg"})
	}
	return pages
}

func TestAWrittenVolumeLeavesNoPartFileBehind(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out", "Vol.01.cbz")

	if err := Write(out, plainPages(t, dir, 3)); err != nil {
		t.Fatal(err)
	}

	if got, want := folderListing(t, filepath.Dir(out)), []string{"Vol.01.cbz"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the output folder holds %v, want %v", got, want)
	}
	if got := len(zipNames(t, out)); got != 3 {
		t.Errorf("the volume holds %d pages, want 3", got)
	}
}

func TestAVolumeReplacesTheOneThatWasThere(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "Vol.01.cbz")
	if err := os.WriteFile(out, []byte("an older volume"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Write(out, plainPages(t, dir, 2)); err != nil {
		t.Fatal(err)
	}

	if got := len(zipNames(t, out)); got != 2 {
		t.Errorf("the volume holds %d pages, want the 2 of the new one", got)
	}
	if got := folderListing(t, dir); !reflect.DeepEqual(got, []string{"Vol.01.cbz", "src"}) {
		t.Errorf("the folder holds %v, want the volume and the sources only", got)
	}
}

func TestAFailedWriteLeavesTheVolumeThatWasThereAsItWas(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "Vol.01.cbz")
	if err := os.WriteFile(out, []byte("the good volume"), 0o644); err != nil {
		t.Fatal(err)
	}
	pages := plainPages(t, dir, 3)
	pages[2].SourcePath = filepath.Join(dir, "gone.jpg") // the third page cannot be read

	if err := Write(out, pages); err == nil {
		t.Fatal("Write() succeeded, want the missing page to fail it")
	}

	got, err := os.ReadFile(out)
	if err != nil || string(got) != "the good volume" {
		t.Errorf("the volume is %q (err %v), want it untouched", got, err)
	}
	if listing := folderListing(t, dir); !reflect.DeepEqual(listing, []string{"Vol.01.cbz", "src"}) {
		t.Errorf("the folder holds %v: a part file was left behind", listing)
	}
}

func TestAFailedFirstWriteLeavesNothingAtAll(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out", "Vol.01.cbz")
	pages := plainPages(t, dir, 2)
	pages[1].SourcePath = filepath.Join(dir, "gone.jpg")

	if err := Write(out, pages); err == nil {
		t.Fatal("Write() succeeded, want an error")
	}

	if got := folderListing(t, filepath.Dir(out)); len(got) != 0 {
		t.Errorf("the output folder holds %v, want it empty: no half-written volume", got)
	}
}

func TestAWriteThatIsCanceledStopsAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "Vol.01.cbz")
	if err := os.WriteFile(out, []byte("the good volume"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	copied := 0

	err := WriteWithProgress(ctx, out, plainPages(t, dir, 5), func(completed int) {
		copied = completed
		if completed == 2 {
			cancel() // the person pressed Ctrl-C while the second page was copied
		}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want one that wraps context.Canceled", err)
	}
	if copied != 2 {
		t.Errorf("%d pages were copied, want it to stop after the second", copied)
	}
	if got, _ := os.ReadFile(out); string(got) != "the good volume" {
		t.Errorf("the volume is %q, want the one that was there", got)
	}
	if listing := folderListing(t, dir); !reflect.DeepEqual(listing, []string{"Vol.01.cbz", "src"}) {
		t.Errorf("the folder holds %v: the part file was left behind", listing)
	}
}

func TestAWriteThatIsAlreadyCanceledCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out", "Vol.01.cbz")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WriteWithProgress(ctx, out, plainPages(t, dir, 2), nil)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want one that wraps context.Canceled", err)
	}
	if got := folderListing(t, filepath.Dir(out)); len(got) != 0 {
		t.Errorf("the output folder holds %v, want it empty", got)
	}
}

func TestAVolumeHasThePermissionsOfAnOrdinaryNewFile(t *testing.T) {
	dir := t.TempDir()
	reference := filepath.Join(dir, "reference")
	f, err := os.OpenFile(reference, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	want, err := os.Stat(reference)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "Vol.01.cbz")

	if err := Write(out, plainPages(t, dir, 1)); err != nil {
		t.Fatal(err)
	}

	got, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != want.Mode().Perm() {
		t.Errorf("the volume has mode %v, want %v: the one a file created with os.Create gets, not the 0600 of a temporary file", got.Mode().Perm(), want.Mode().Perm())
	}
}

func TestPartFileNamesDoNotLookLikeVolumes(t *testing.T) {
	dir := t.TempDir()
	f, err := createPart(filepath.Join(dir, "Manga - Vol.01.cbz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	name := filepath.Base(f.Name())
	if !strings.HasPrefix(name, ".Manga - Vol.01.cbz.") || !strings.HasSuffix(name, ".part") {
		t.Errorf("part file is called %q, want .<volume>.<random>.part", name)
	}
	if strings.HasSuffix(strings.ToLower(name), ".cbz") {
		t.Errorf("part file %q ends in .cbz, and a later run would take it for a chapter", name)
	}
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("%s is not a readable archive: %v", path, err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}
