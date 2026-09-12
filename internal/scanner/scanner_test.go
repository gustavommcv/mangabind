package scanner

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScan(t *testing.T) {
	entries, skipped, err := Scan("../../testdata/sample_manga")
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped = %v, want none", skipped)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}

	// Natural order, not volume/chapter order - grouping happens later.
	want := []string{
		"Chapter 3",
		"Vol.01 Ch.0001 - Title One (en) [GroupA]",
		"Vol.01 Ch.0002 - Title Two (en) [GroupA]",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("Scan() names = %v, want %v", names, want)
	}
	for _, e := range entries {
		if e.IsArchive {
			t.Fatalf("entry %q: IsArchive = true, want false (no .cbz in this fixture)", e.Name)
		}
	}
}

func TestScanMixedFormats(t *testing.T) {
	entries, skipped, err := Scan("../../testdata/sample_manga_mixed")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(skipped, []string{"notes.txt"}) {
		t.Fatalf("skipped = %v, want [notes.txt]", skipped)
	}

	got := map[string]bool{} // name -> IsArchive
	for _, e := range entries {
		got[e.Name] = e.IsArchive
	}
	want := map[string]bool{
		"Vol.01 Ch.0001 - Title One (en) [GroupA]": false,
		"Vol.01 Ch.0002 - Title Two (en) [GroupA]": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestPages(t *testing.T) {
	pages, err := Pages("../../testdata/sample_manga/Chapter 3")
	if err != nil {
		t.Fatal(err)
	}

	// Proves numeric (not lexicographic) ordering: "2.jpg" before "10.jpg".
	want := []string{"1.jpg", "2.jpg", "10.jpg"}
	if !reflect.DeepEqual(pages, want) {
		t.Fatalf("Pages() = %v, want %v", pages, want)
	}
}

func TestScanNonexistentDir(t *testing.T) {
	_, _, err := Scan(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error scanning a nonexistent directory")
	}
}

func TestPagesNonexistentDir(t *testing.T) {
	_, err := Pages(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error listing pages in a nonexistent directory")
	}
}

func TestPagesInArchiveNotAZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-zip.cbz")
	if err := os.WriteFile(path, []byte("this is not a zip file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PagesInArchive(path); err == nil {
		t.Fatal("expected an error reading a corrupt/non-zip .cbz")
	}
}

func TestIsIgnorableJunk(t *testing.T) {
	cases := map[string]bool{
		".DS_Store":   true,
		"Thumbs.db":   true,
		"desktop.ini": true,
		"notes.txt":   false,
		"cover.jpg":   false,
	}
	for name, want := range cases {
		if got := isIgnorableJunk(name); got != want {
			t.Errorf("isIgnorableJunk(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestPagesInArchive(t *testing.T) {
	cbzPath := filepath.Join(t.TempDir(), "chapter.cbz")
	f, err := os.Create(cbzPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"01.jpg", "10.jpg", "02.jpg"} { // written out of order on purpose
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	pages, err := PagesInArchive(cbzPath)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"01.jpg", "02.jpg", "10.jpg"}
	if !reflect.DeepEqual(pages, want) {
		t.Fatalf("PagesInArchive() = %v, want %v", pages, want)
	}
}
