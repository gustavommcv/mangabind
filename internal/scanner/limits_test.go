package scanner

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// declared is an archive entry that says how big it is without holding that
// much: the header carries the sizes, the data is not there. The scanner only
// reads headers, which is what makes it possible to test sizes of gigabytes.
type declared struct {
	name         string
	size, packed uint64
}

func writeDeclaredArchive(t *testing.T, path string, entries ...declared) *zip.ReadCloser {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range entries {
		w, err := zw.CreateRaw(&zip.FileHeader{Name: e.name, Method: zip.Deflate, CompressedSize64: e.packed, UncompressedSize64: e.size})
		if err != nil {
			t.Fatal(err)
		}
		_ = w
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { zr.Close() })
	return zr
}

func TestAnEntryIsTooLargeToBeAPageByItsSizeOrByItsRatio(t *testing.T) {
	const (
		MiB = 1 << 20
		KiB = 1 << 10
	)
	cases := []struct {
		name         string
		size, packed uint64
		tooLarge     bool
	}{
		{"an ordinary page", 3 * MiB, 3 * MiB, false},
		{"a big scan stored as it is", 60 * MiB, 60 * MiB, false},
		{"exactly the ceiling", 256 * MiB, 256 * MiB, false},
		{"one byte over the ceiling", 256*MiB + 1, 256*MiB + 1, true},
		{"a gigabyte stored as it is", 1024 * MiB, 1024 * MiB, true},
		{"over the ratio and over the floor", 20 * MiB, 20 * KiB, true},
		{"just inside the ratio, over the floor", 20 * MiB, 21 * KiB, false},
		{"exactly the ratio, over the floor", 20 * MiB, 20 * MiB / 1000, false},
		{"over the ratio but below the floor: a blank bitmap", 15 * MiB, 1 * KiB, false},
		{"exactly the floor, however packed", 16 * MiB, 1, false},
		{"over the floor and stored in nothing", 17 * MiB, 0, true},
		{"over the ceiling and stored in nothing", 300 * MiB, 0, true},
	}
	var entries []declared
	for _, c := range cases {
		entries = append(entries, declared{c.name + ".jpg", c.size, c.packed})
	}
	zr := writeDeclaredArchive(t, filepath.Join(t.TempDir(), "sizes.cbz"), entries...)

	for i, f := range zr.File {
		if got := tooLarge(f); got != cases[i].tooLarge {
			t.Errorf("%s (%d bytes from %d): tooLarge = %v, want %v", cases[i].name, cases[i].size, cases[i].packed, got, cases[i].tooLarge)
		}
		if got := IsPageEntry(f); got != !cases[i].tooLarge {
			t.Errorf("%s: IsPageEntry = %v, want %v", cases[i].name, got, !cases[i].tooLarge)
		}
	}
}

func TestPagesInAnArchiveLeaveOutTheEntriesThatWouldExpandTooFar(t *testing.T) {
	const MiB = 1 << 20
	path := filepath.Join(t.TempDir(), "Ch.001.cbz")
	writeDeclaredArchive(t, path,
		declared{"002.jpg", 2 * MiB, 2 * MiB},
		declared{"bomb-b.jpg", 400 * MiB, 400 << 10}, // 400 MiB from 400 KiB
		declared{"001.png", 1 * MiB, 1 * MiB},
		declared{"bomb-a.png", 300 * MiB, 300 * MiB},
		declared{"notes.txt", 300 * MiB, 300 * MiB}, // not an image: skipped for that, not for its size
		declared{"__MACOSX/._001.png", 300 * MiB, 300 * MiB},
	)

	pages, skipped, oversized, err := PagesInArchive(path)

	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"001.png", "002.jpg"}; !reflect.DeepEqual(pages, want) {
		t.Errorf("pages = %v, want %v", pages, want)
	}
	if want := []string{"notes.txt"}; !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped = %v, want %v", skipped, want)
	}
	want := []Oversized{
		{Name: "bomb-a.png", Size: 300 * MiB, Compressed: 300 * MiB},
		{Name: "bomb-b.jpg", Size: 400 * MiB, Compressed: 400 << 10},
	}
	if !reflect.DeepEqual(oversized, want) {
		t.Errorf("oversized = %+v, want %+v", oversized, want)
	}
}
