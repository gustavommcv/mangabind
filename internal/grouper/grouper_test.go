package grouper

import (
	"reflect"
	"testing"

	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

func TestGroup(t *testing.T) {
	dirs, err := scanner.Scan("../../testdata/sample_manga")
	if err != nil {
		t.Fatal(err)
	}

	reg := parser.DefaultRegistry()
	var chapters []Chapter
	var unparsed []string
	for _, d := range dirs {
		parsed, _, ok := reg.Parse(d.Name)
		if !ok {
			unparsed = append(unparsed, d.Name)
			continue
		}
		pages, err := scanner.Pages(d.Path)
		if err != nil {
			t.Fatal(err)
		}
		chapters = append(chapters, Chapter{Parsed: parsed, Dir: d.Path, Pages: pages})
	}

	result := Group(chapters, unparsed)

	if len(unparsed) != 0 {
		t.Fatalf("all sample_manga folder names should parse, got unparsed: %v", unparsed)
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("got %d volumes, want 1", len(result.Volumes))
	}
	if result.Volumes[0].Number != 1 {
		t.Fatalf("volume number = %v, want 1", result.Volumes[0].Number)
	}

	var names []string
	for _, p := range result.Volumes[0].Pages {
		names = append(names, p.ArchiveName)
	}
	// Ch.0001 (3 pages: 01,02,10.jpg) then Ch.0002 (1 page), each chapter in
	// its own directory named by position (not source chapter number) and
	// title - see chapterDirName.
	want := []string{
		"c001 - Title One/p0001.jpg",
		"c001 - Title One/p0002.jpg",
		"c001 - Title One/p0003.jpg",
		"c002 - Title Two/p0001.jpg",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("archive names = %v, want %v", names, want)
	}

	if len(result.Unassigned) != 1 {
		t.Fatalf("got %d unassigned chapters, want 1", len(result.Unassigned))
	}
	if result.Unassigned[0].Parsed.Chapter != 3 {
		t.Fatalf("unassigned chapter = %v, want chapter 3 (\"Chapter 3\" carries no volume number)",
			result.Unassigned[0].Parsed.Chapter)
	}
}
