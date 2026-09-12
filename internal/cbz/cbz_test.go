package cbz

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	src1 := filepath.Join(dir, "a.jpg")
	src2 := filepath.Join(dir, "b.jpg")
	if err := os.WriteFile(src1, []byte("AAA"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src2, []byte("BB"), 0o644); err != nil {
		t.Fatal(err)
	}

	pages := []grouper.Page{
		{SourcePath: src1, ArchiveName: "c001 - Chapter One/p0001.jpg"},
		{SourcePath: src2, ArchiveName: "c002 - Chapter Two/p0001.jpg"},
	}

	out := filepath.Join(dir, "out", "Vol.01.cbz")
	if err := Write(out, pages); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	if len(zr.File) != 2 {
		t.Fatalf("got %d files, want 2", len(zr.File))
	}
	// A "/" in ArchiveName must produce a real per-chapter directory inside
	// the archive - this is what lets KCC build a per-chapter TOC entry.
	if zr.File[0].Name != "c001 - Chapter One/p0001.jpg" || zr.File[1].Name != "c002 - Chapter Two/p0001.jpg" {
		t.Fatalf("unexpected names/order: %q, %q", zr.File[0].Name, zr.File[1].Name)
	}
	if zr.File[0].Method != zip.Store {
		t.Fatalf("method = %v, want zip.Store (no recompression)", zr.File[0].Method)
	}

	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "AAA" {
		t.Fatalf("content = %q, want %q", data, "AAA")
	}
}

func TestWriteFromArchiveSource(t *testing.T) {
	dir := t.TempDir()

	// A source .cbz, as if downloaded directly by HakuNeko in that format.
	srcCbz := filepath.Join(dir, "source-chapter.cbz")
	sf, err := os.Create(srcCbz)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(sf)
	w, err := zw.Create("01.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("PAGE-ONE")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sf.Close()

	pages := []grouper.Page{
		{SourcePath: srcCbz, SourceInArchive: "01.jpg", ArchiveName: "c001 - Chapter One/p0001.jpg"},
	}

	out := filepath.Join(dir, "out", "Vol.01.cbz")
	if err := Write(out, pages); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	if len(zr.File) != 1 || zr.File[0].Name != "c001 - Chapter One/p0001.jpg" {
		t.Fatalf("unexpected output: %+v", zr.File)
	}

	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "PAGE-ONE" {
		t.Fatalf("content = %q, want %q (bytes should pass through from inside the source archive)", data, "PAGE-ONE")
	}
}

func TestWriteMissingSource(t *testing.T) {
	dir := t.TempDir()
	pages := []grouper.Page{
		{SourcePath: filepath.Join(dir, "does-not-exist.jpg"), ArchiveName: "c001/p0001.jpg"},
	}
	out := filepath.Join(dir, "Vol.01.cbz")
	if err := Write(out, pages); err == nil {
		t.Fatal("expected an error when a page's source file doesn't exist")
	}
}

func TestWriteMissingArchiveEntry(t *testing.T) {
	dir := t.TempDir()
	srcCbz := filepath.Join(dir, "source.cbz")
	sf, err := os.Create(srcCbz)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(sf)
	if err := zw.Close(); err != nil { // empty archive, no "01.jpg" entry
		t.Fatal(err)
	}
	sf.Close()

	pages := []grouper.Page{
		{SourcePath: srcCbz, SourceInArchive: "01.jpg", ArchiveName: "c001/p0001.jpg"},
	}
	out := filepath.Join(dir, "Vol.01.cbz")
	if err := Write(out, pages); err == nil {
		t.Fatal("expected an error when the requested entry isn't in the source archive")
	}
}

func TestVolumeFileName(t *testing.T) {
	cases := []struct {
		manga string
		vol   float64
		want  string
	}{
		{"Chainsaw Man", 1, "Chainsaw Man - Vol.01.cbz"},
		{"Some Manga", 12, "Some Manga - Vol.12.cbz"},
		{"Omnibus", 1.5, "Omnibus - Vol.1.5.cbz"},
	}
	for _, c := range cases {
		if got := VolumeFileName(c.manga, c.vol); got != c.want {
			t.Errorf("VolumeFileName(%q, %v) = %q, want %q", c.manga, c.vol, got, c.want)
		}
	}
}
