package scanner

import (
	"reflect"
	"testing"
)

func TestScan(t *testing.T) {
	dirs, err := Scan("../../testdata/sample_manga")
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, d := range dirs {
		names = append(names, d.Name)
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
