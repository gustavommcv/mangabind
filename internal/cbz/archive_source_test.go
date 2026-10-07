package cbz

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gustavommcv/mangabind/internal/grouper"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

type sourceEntry struct{ name, content string }

// writeSourceCBZ makes a .cbz of stored entries, in the given order. Names are
// written as they are, duplicates and unusual ones included.
func writeSourceCBZ(t *testing.T, path string, entries ...sourceEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range entries {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// readVolume returns the content of each entry of a written volume, in order.
func readVolume(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var contents []string
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		contents = append(contents, string(data))
	}
	return contents
}

func archivePages(archive string, names ...string) []grouper.Page {
	pages := make([]grouper.Page, len(names))
	for i, name := range names {
		pages[i] = grouper.Page{SourcePath: archive, SourceInArchive: name, ArchiveName: fmt.Sprintf("c001/p%04d.jpg", i+1)}
	}
	return pages
}

// countOpens makes the writer's opens of source archives countable for the
// length of the test.
func countOpens(t *testing.T) *int {
	t.Helper()
	opens := 0
	original := openArchive
	openArchive = func(name string) (*zip.ReadCloser, error) {
		opens++
		return original(name)
	}
	t.Cleanup(func() { openArchive = original })
	return &opens
}

func TestPagesComeFromEntriesWhoseNamesAreNotLocal(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	names := []string{"./a.jpg", "../b.jpg", "/c.jpg", `dir\d.png`, "x/../e.jpg", "plain.jpg"}
	var entries []sourceEntry
	for i, name := range names {
		entries = append(entries, sourceEntry{name, fmt.Sprintf("content-%d", i)})
	}
	writeSourceCBZ(t, source, entries...)
	out := filepath.Join(dir, "Vol.01.cbz")

	if err := Write(out, archivePages(source, names...)); err != nil {
		t.Fatalf("Write() = %v, want every page copied whatever the entry is called", err)
	}

	want := []string{"content-0", "content-1", "content-2", "content-3", "content-4", "content-5"}
	if got := readVolume(t, out); !reflect.DeepEqual(got, want) {
		t.Errorf("volume holds %v, want %v", got, want)
	}
}

func TestEntriesWithTheSameNameEachGiveTheirOwnContent(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	writeSourceCBZ(t, source,
		sourceEntry{"p1.jpg", "first"},
		sourceEntry{"p1.jpg", "second"},
		sourceEntry{"p2.jpg", "other"},
		sourceEntry{"p1.jpg", "third"},
	)
	out := filepath.Join(dir, "Vol.01.cbz")

	// The scanner lists every entry, in order: the name appears once per entry.
	if err := Write(out, archivePages(source, "p1.jpg", "p1.jpg", "p1.jpg", "p2.jpg")); err != nil {
		t.Fatal(err)
	}

	want := []string{"first", "second", "third", "other"}
	if got := readVolume(t, out); !reflect.DeepEqual(got, want) {
		t.Errorf("volume holds %v, want %v: the second image was the first again", got, want)
	}
}

func TestAnEntryAskedForMoreOftenThanItExistsIsAnError(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	writeSourceCBZ(t, source, sourceEntry{"p1.jpg", "only"})

	err := Write(filepath.Join(dir, "Vol.01.cbz"), archivePages(source, "p1.jpg", "p1.jpg"))

	if err == nil || !strings.Contains(err.Error(), `has no entry "p1.jpg"`) {
		t.Errorf("error = %v, want one that names the missing entry", err)
	}
}

func TestEachSourceArchiveIsOpenedOnceForAsLongAsItsPagesComeInARow(t *testing.T) {
	dir := t.TempDir()
	var pages []grouper.Page
	for _, name := range []string{"A.cbz", "B.cbz"} {
		path := filepath.Join(dir, name)
		var entries []sourceEntry
		var names []string
		for i := 1; i <= 40; i++ {
			entries = append(entries, sourceEntry{fmt.Sprintf("%03d.jpg", i), name})
			names = append(names, fmt.Sprintf("%03d.jpg", i))
		}
		writeSourceCBZ(t, path, entries...)
		pages = append(pages, archivePages(path, names...)...)
	}
	for i := range pages {
		pages[i].ArchiveName = fmt.Sprintf("c%03d/p%04d.jpg", i/40+1, i%40+1)
	}
	opens := countOpens(t)

	if err := Write(filepath.Join(dir, "Vol.01.cbz"), pages); err != nil {
		t.Fatal(err)
	}

	if *opens != 2 {
		t.Errorf("source archives were opened %d times for 80 pages from 2 archives, want 2", *opens)
	}
}

func TestPagesThatAlternateBetweenArchivesStillGetTheRightContent(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "A.cbz"), filepath.Join(dir, "B.cbz")
	writeSourceCBZ(t, a, sourceEntry{"p.jpg", "a1"}, sourceEntry{"p.jpg", "a2"})
	writeSourceCBZ(t, b, sourceEntry{"p.jpg", "b1"})
	pages := []grouper.Page{
		{SourcePath: a, SourceInArchive: "p.jpg", ArchiveName: "1.jpg"},
		{SourcePath: b, SourceInArchive: "p.jpg", ArchiveName: "2.jpg"},
		{SourcePath: a, SourceInArchive: "p.jpg", ArchiveName: "3.jpg"},
	}
	opens := countOpens(t)
	out := filepath.Join(dir, "Vol.01.cbz")

	if err := Write(out, pages); err != nil {
		t.Fatal(err)
	}

	// A is opened again, and its duplicates carry on where they stopped.
	if want := []string{"a1", "b1", "a2"}; !reflect.DeepEqual(readVolume(t, out), want) {
		t.Errorf("volume holds %v, want %v", readVolume(t, out), want)
	}
	if *opens != 3 {
		t.Errorf("opens = %d, want 3", *opens)
	}
}

func TestSourceArchivesAreClosedWhenTheVolumeIsWritten(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	writeSourceCBZ(t, source, sourceEntry{"p.jpg", "x"})

	if err := Write(filepath.Join(dir, "Vol.01.cbz"), archivePages(source, "p.jpg")); err != nil {
		t.Fatal(err)
	}

	// Windows will not delete a file that is still open.
	if err := os.Remove(source); err != nil {
		t.Errorf("the source archive is still open: %v", err)
	}
}

func TestAnArchiveThatGODEBUGCallsInsecureIsStillRead(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	writeSourceCBZ(t, source, sourceEntry{"../b.jpg", "bee"}, sourceEntry{"c.jpg", "sea"})
	t.Setenv("GODEBUG", "zipinsecurepath=0")
	if zr, err := zip.OpenReader(source); !errors.Is(err, zip.ErrInsecurePath) || zr == nil {
		t.Skipf("this Go does not report ErrInsecurePath here (err = %v)", err)
	} else {
		zr.Close()
	}
	out := filepath.Join(dir, "Vol.01.cbz")

	if err := Write(out, archivePages(source, "../b.jpg", "c.jpg")); err != nil {
		t.Fatalf("Write() = %v, want the archive read despite the name", err)
	}

	if want := []string{"bee", "sea"}; !reflect.DeepEqual(readVolume(t, out), want) {
		t.Errorf("volume holds %v, want %v", readVolume(t, out), want)
	}
}

func TestAnEntryLeftOutForItsSizeDoesNotTakeTheNameOfOneThatWasListed(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	// The first entry called p1.jpg declares 300 MiB and is not a page; the
	// second is. The nth page that asks for the name must get the nth entry
	// that the scanner listed, not the nth in the archive.
	if _, err := zw.CreateRaw(&zip.FileHeader{Name: "p1.jpg", Method: zip.Deflate, CompressedSize64: 300 << 20, UncompressedSize64: 300 << 20}); err != nil {
		t.Fatal(err)
	}
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "p1.jpg", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("the real page")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	names, _, oversized, err := scanner.PagesInArchive(source)
	if err != nil || !reflect.DeepEqual(names, []string{"p1.jpg"}) || len(oversized) != 1 {
		t.Fatalf("PagesInArchive() = %v, oversized %v, err %v, want one page and one left out", names, oversized, err)
	}
	out := filepath.Join(dir, "Vol.01.cbz")

	if err := Write(out, archivePages(source, names...)); err != nil {
		t.Fatalf("Write() = %v, want the listed entry copied", err)
	}

	if want := []string{"the real page"}; !reflect.DeepEqual(readVolume(t, out), want) {
		t.Errorf("volume holds %v, want %v", readVolume(t, out), want)
	}
}

func TestAnEntryThatUnderstatesItsSizeCannotExpandPastIt(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.cbz")
	// 5 MiB of zeros, deflated by hand, under a header that says the entry is
	// 1000 bytes: small enough that the scanner takes it for a page.
	var packed bytes.Buffer
	fw, err := flate.NewWriter(&packed, flate.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(make([]byte, 5<<20)); err != nil {
		t.Fatal(err)
	}
	if err := fw.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.CreateRaw(&zip.FileHeader{Name: "p1.jpg", Method: zip.Deflate, CompressedSize64: uint64(packed.Len()), UncompressedSize64: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(packed.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	names, _, oversized, err := scanner.PagesInArchive(source)
	if err != nil || len(names) != 1 || len(oversized) != 0 {
		t.Fatalf("PagesInArchive() = %v, %v, %v: the header was meant to pass as a page", names, oversized, err)
	}
	out := filepath.Join(dir, "Vol.01.cbz")

	err = Write(out, archivePages(source, names...))

	if err == nil {
		t.Fatal("Write() succeeded: the entry expanded to 5 MiB under a header that says 1000 bytes")
	}
	if _, statErr := os.Stat(out); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("a volume was left behind: %v", statErr)
	}
}
