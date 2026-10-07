package scanner

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClassifyPage(t *testing.T) {
	cases := []struct {
		name string
		want pageClass
	}{
		// What a comic reader opens, in any case.
		{"001.jpg", pageImage},
		{"001.JPG", pageImage},
		{"001.jpeg", pageImage},
		{"p0001.png", pageImage},
		{"a.gif", pageImage},
		{"a.WebP", pageImage},
		{"a.bmp", pageImage},
		{"a.avif", pageImage},
		{"a.jxl", pageImage},
		{"a.tif", pageImage},
		{"a.TIFF", pageImage},
		{"my.page.001.png", pageImage},
		{".hidden-but-an-image.jpg", pageImage},
		{"chapter/001.jpg", pageImage},
		{`chapter\001.jpg`, pageImage}, // an archive made on Windows
		// Leftovers that are never pages and never worth a word.
		{".DS_Store", pageJunk},
		{".ds_store", pageJunk},
		{"Thumbs.db", pageJunk},
		{"THUMBS.DB", pageJunk},
		{"desktop.ini", pageJunk},
		{"ComicInfo.xml", pageJunk},
		{"chapter/comicinfo.xml", pageJunk},
		{"._001.jpg", pageJunk},
		{"chapter/._001.jpg", pageJunk},
		{"__MACOSX/001.jpg", pageJunk},
		{"__MACOSX/chapter/._001.jpg", pageJunk},
		{"chapter/__MACOSX/001.png", pageJunk},
		{`__MACOSX\001.jpg`, pageJunk},
		// Anything else is not a page, and the person is told it was left out.
		{"notes.txt", pageOther},
		{"credits.html", pageOther},
		{"readme", pageOther},
		{"001", pageOther},
		{"book.pdf", pageOther},
		{"volume.epub", pageOther},
		{"archive.zip", pageOther},
		{"001.jpg.bak", pageOther},
		{"jpg", pageOther},
		{"chapter/notes.txt", pageOther},
		{"__MACOSX", pageOther}, // a file by that name, not the folder
	}
	for _, c := range cases {
		if got := classifyPage(c.name); got != c.want {
			t.Errorf("classifyPage(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPagesAreTheImagesOfAChapterFolder(t *testing.T) {
	root := t.TempDir()
	chapter := filepath.Join(root, "Vol.01 Ch.001")
	if err := os.MkdirAll(filepath.Join(chapter, "extras"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"010.jpg", "002.JPG", "001.png", "003.webp",
		".DS_Store", "Thumbs.db", "desktop.ini", "ComicInfo.xml", "._001.png",
		"credits.html", "notes.txt", "readme", "book.pdf",
	} {
		if err := os.WriteFile(filepath.Join(chapter, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A folder inside a chapter is not read, as before.
	if err := os.WriteFile(filepath.Join(chapter, "extras", "bonus.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	pages, skipped, links, err := Pages(root, chapter)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"001.png", "002.JPG", "003.webp", "010.jpg"}; !reflect.DeepEqual(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
	// The junk is not named; what is left out and might matter is.
	if want := []string{"book.pdf", "credits.html", "notes.txt", "readme"}; !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped = %v, want %v", skipped, want)
	}
	if len(links) != 0 {
		t.Errorf("links = %v, want none", links)
	}
}

func TestAFolderOfOnlyJunkHasNoPagesAndNothingToSay(t *testing.T) {
	root := t.TempDir()
	chapter := filepath.Join(root, "Ch.001")
	if err := os.MkdirAll(chapter, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".DS_Store", "Thumbs.db"} {
		if err := os.WriteFile(filepath.Join(chapter, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	pages, skipped, _, err := Pages(root, chapter)

	if err != nil || len(pages) != 0 || len(skipped) != 0 {
		t.Errorf("pages = %v, skipped = %v, err = %v, want nothing at all", pages, skipped, err)
	}
}

func TestPagesInAnArchiveAreItsImages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Ch.001.cbz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{
		"010.jpg", "002.JPG", "dir/001.png", `win\003.webp`,
		"__MACOSX/", "__MACOSX/010.jpg", "__MACOSX/._002.JPG", "dir/._001.png",
		"ComicInfo.xml", "dir/Thumbs.db",
		"notes.txt", "dir/credits.html",
		"empty-folder/",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("x")); err != nil && name[len(name)-1] != '/' {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	pages, skipped, _, err := PagesInArchive(path)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"002.JPG", "010.jpg", "dir/001.png", `win\003.webp`}; !reflect.DeepEqual(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
	if want := []string{"dir/credits.html", "notes.txt"}; !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped = %v, want %v", skipped, want)
	}
}

func TestPagesInAnArchiveThatGODEBUGCallsInsecureAreStillListed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Ch.001.cbz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"../002.jpg", "001.jpg", "notes.txt"} {
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
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODEBUG", "zipinsecurepath=0")
	if zr, err := zip.OpenReader(path); !errors.Is(err, zip.ErrInsecurePath) || zr == nil {
		t.Skipf("this Go does not report ErrInsecurePath here (err = %v)", err)
	} else {
		zr.Close()
	}

	pages, skipped, _, err := PagesInArchive(path)

	if err != nil {
		t.Fatalf("PagesInArchive() = %v, want the entries listed despite the name", err)
	}
	if want := []string{"../002.jpg", "001.jpg"}; !reflect.DeepEqual(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
	if want := []string{"notes.txt"}; !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped = %v, want %v", skipped, want)
	}
}
