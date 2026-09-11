package grouper

import (
	"reflect"
	"testing"

	"github.com/gustavommcv/mangabind/internal/parser"
)

func TestGroupArchiveSourcedChapter(t *testing.T) {
	vol := 1.0
	chapters := []Chapter{
		{
			Parsed:    parser.ParsedChapter{Volume: &vol, Chapter: 1, Title: "Chapter One"},
			Path:      "/manga/Vol.01 Ch.0001 - Chapter One (en) [Group].cbz",
			Pages:     []string{"01.jpg", "02.jpg"},
			IsArchive: true,
		},
	}

	result := Group(chapters, nil)

	if len(result.Volumes) != 1 || len(result.Volumes[0].Pages) != 2 {
		t.Fatalf("got %+v, want 1 volume with 2 pages", result.Volumes)
	}

	want := []Page{
		{SourcePath: chapters[0].Path, SourceInArchive: "01.jpg", ArchiveName: "c001 - Chapter One/p0001.jpg"},
		{SourcePath: chapters[0].Path, SourceInArchive: "02.jpg", ArchiveName: "c001 - Chapter One/p0002.jpg"},
	}
	if !reflect.DeepEqual(result.Volumes[0].Pages, want) {
		t.Fatalf("pages = %+v, want %+v", result.Volumes[0].Pages, want)
	}
}
