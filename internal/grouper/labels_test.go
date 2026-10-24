package grouper

import (
	"reflect"
	"testing"

	"github.com/gustavommcv/mangabind/internal/parser"
)

func chapterOf(volume, chapter float64, special, title string) Chapter {
	v := volume
	return Chapter{
		Parsed: parser.ParsedChapter{Volume: &v, Chapter: chapter, Special: special, Title: title},
		Path:   "/in/" + title,
		Pages:  []string{"001.jpg"},
	}
}

func dirsOf(v Volume) []string {
	dirs := make([]string, len(v.Chapters))
	for i, c := range v.Chapters {
		dirs[i] = c.Dir
	}
	return dirs
}

func TestLabelsByNumberNameAChapterByItsOwnNumber(t *testing.T) {
	chapters := []Chapter{
		chapterOf(1, 12, "", "Twelve"),
		chapterOf(1, 5.5, "", ""),
		chapterOf(1, 5, "", "Five"),
		chapterOf(1, 100, "x1", ""),
	}

	result := GroupLabeled(chapters, nil, LabelsByNumber)

	want := []string{"Ch.5 - Five", "Ch.5.5", "Ch.12 - Twelve", "Ch.100x1"}
	if got := dirsOf(result.Volumes[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("directories = %v, want %v", got, want)
	}
}

// The numbers are those of the chapters, not their places: a chapter that is missing, or that was left out, does
// not move the ones after it.
func TestLabelsByNumberDoNotMoveWhenAChapterIsMissing(t *testing.T) {
	result := GroupLabeled([]Chapter{chapterOf(1, 1, "", ""), chapterOf(1, 3, "", "")}, nil, LabelsByNumber)

	if got, want := dirsOf(result.Volumes[0]), []string{"Ch.1", "Ch.3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("directories = %v, want %v", got, want)
	}
}

func TestTheDefaultLabelsAreByPosition(t *testing.T) {
	chapters := []Chapter{chapterOf(1, 3, "", "Three"), chapterOf(1, 1, "", "")}

	result := Group(chapters, nil)

	if got, want := dirsOf(result.Volumes[0]), []string{"c001", "c002 - Three"}; !reflect.DeepEqual(got, want) {
		t.Errorf("directories = %v, want %v", got, want)
	}
}

// The converter reads the directories of a volume in natural order of their names. Where the names by number
// would be read in another order than the chapters are, the volume is labelled by position as well.
func TestLabelsByNumberKeepTheReadingOrderOfTheChapters(t *testing.T) {
	// 21x1 comes before 21.5 in a volume, and "Ch.21.5" is before "Ch.21x1" by name.
	chapters := []Chapter{chapterOf(1, 21.5, "", ""), chapterOf(1, 21, "x1", ""), chapterOf(1, 21, "", "")}

	result := GroupLabeled(chapters, nil, LabelsByNumber)

	want := []string{"c001 - Ch.21", "c002 - Ch.21x1", "c003 - Ch.21.5"}
	if got := dirsOf(result.Volumes[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("directories = %v, want %v", got, want)
	}
}

func TestChaptersKeptAsAlternativesAreNotAConflict(t *testing.T) {
	first, second := chapterOf(1, 6, "", "Six"), chapterOf(1, 6, "", "Six")
	first.Path, second.Path = "/in/a", "/in/b"
	first.Version, second.Version = &Version{Index: 1, Count: 2}, &Version{Index: 2, Count: 2}
	// Given in the reverse order, they are bound in the order of their versions.
	chapters := []Chapter{chapterOf(1, 7, "", ""), second, first}

	result := GroupLabeled(chapters, nil, LabelsByNumber)

	if len(result.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v, want none", result.Conflicts)
	}
	want := []string{"Ch.6 - Six (version 1 of 2)", "Ch.6 - Six (version 2 of 2)", "Ch.7"}
	if got := dirsOf(result.Volumes[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("directories = %v, want %v", got, want)
	}
	if got := result.Volumes[0].Chapters[0].Path; got != "/in/a" {
		t.Errorf("the first version comes from %q, want /in/a", got)
	}
}

// Only the chapters that all were chosen are kept: one that was not chosen among others is still a conflict.
func TestAChapterNotChosenAmongOthersIsStillAConflict(t *testing.T) {
	chosen, other := chapterOf(1, 6, "", ""), chapterOf(1, 6, "", "")
	chosen.Version = &Version{Index: 1, Count: 2}

	result := Group([]Chapter{chosen, other}, nil)

	if len(result.Conflicts) != 1 || len(result.Volumes[0].Chapters) != 0 {
		t.Errorf("conflicts %+v, chapters %v; want one conflict and no chapter", result.Conflicts, dirsOf(result.Volumes[0]))
	}
}

func TestAlternativesAreToldApartByPositionLabelsToo(t *testing.T) {
	first, second := chapterOf(1, 6, "", "Six"), chapterOf(1, 6, "", "Six")
	first.Version, second.Version = &Version{Index: 1, Count: 2}, &Version{Index: 2, Count: 2}

	result := Group([]Chapter{first, second}, nil)

	want := []string{"c001 - Six (version 1 of 2)", "c002 - Six (version 2 of 2)"}
	if got := dirsOf(result.Volumes[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("directories = %v, want %v", got, want)
	}
}

func TestAVolumeKnowsHowManyPagesEachChapterHas(t *testing.T) {
	two := chapterOf(1, 1, "", "")
	two.Pages = []string{"1.jpg", "2.jpg"}

	result := Group([]Chapter{two}, nil)

	if got := result.Volumes[0].Chapters[0]; got.PageCount != 2 || got.Path != two.Path {
		t.Errorf("chapter = %+v, want 2 pages from %q", got, two.Path)
	}
}
