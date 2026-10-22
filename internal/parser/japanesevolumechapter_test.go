package parser

import "testing"

func TestJapaneseVolumeChapterParser(t *testing.T) {
	cases := []struct {
		name    string
		dir     string
		ok      bool
		volume  float64
		chapter float64
		title   string
	}{
		{name: "第N巻 第N話", dir: "第1巻 第2話", ok: true, volume: 1, chapter: 2},
		{name: "N巻 N話", dir: "1巻 2話", ok: true, volume: 1, chapter: 2},
		{name: "zero padded", dir: "第01巻 第012話", ok: true, volume: 1, chapter: 12},
		{name: "with #N", dir: "第3巻 #25", ok: true, volume: 3, chapter: 25},
		{name: "with a title", dir: "第1巻 第2話 出会い", ok: true, volume: 1, chapter: 2, title: "出会い"},
		{name: "with a title after a dash", dir: "第1巻 - 第2話 - 出会い", ok: true, volume: 1, chapter: 2, title: "出会い"},
		{name: "ideographic space", dir: "第1巻　第2話", ok: true, volume: 1, chapter: 2},
		{name: "no space between", dir: "第1巻第2話", ok: true, volume: 1, chapter: 2},
		{name: "full-width digits", dir: "第１巻 第１０話", ok: true, volume: 1, chapter: 10},
		{name: "decimal chapter", dir: "第2巻 第10.5話", ok: true, volume: 2, chapter: 10.5},
		{name: "decimal volume", dir: "第1.5巻 第3話", ok: true, volume: 1.5, chapter: 3},
		{name: "space inside the volume", dir: "第 4 巻 第 9 話", ok: true, volume: 4, chapter: 9},
		{name: "not this convention: a volume alone", dir: "第1巻", ok: false},
		{name: "not this convention: a volume and no chapter", dir: "第1巻 おまけ", ok: false},
		{name: "not this convention: a range of volumes", dir: "第1-3巻 第2話", ok: false},
		{name: "not this convention: a range of chapters", dir: "第1巻 第1-3話", ok: false},
		{name: "not this convention: a chapter alone", dir: "第2話", ok: false},
		{name: "not this convention: the chapter first", dir: "第2話 第1巻", ok: false},
		{name: "after an author and a title", dir: "[作者] タイトル 第1巻 第2話", ok: true, volume: 1, chapter: 2},
		{name: "after a real name, N巻 without 第", dir: "(一般コミック) [奥浩哉] いぬやしき 2巻 第3話 出会い", ok: true, volume: 2, chapter: 3, title: "出会い"},
		{name: "after a scanlator and an underscore", dir: "Raw_第3巻 #25", ok: true, volume: 3, chapter: 25},
		{name: "not this convention: N巻 stuck to the title", dir: "タイトル2巻 第3話", ok: false},
		{name: "not this convention: a volume alone after a title", dir: "(一般コミック) [奥浩哉] いぬやしき 第09巻", ok: false},
		{name: "not this convention: the English convention", dir: "Vol.01 Ch.002", ok: false},
		{name: "not this convention: empty", dir: "", ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := JapaneseVolumeChapterParser{}.Parse(c.dir)
			if ok != c.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v (got %+v)", c.dir, ok, c.ok, got)
			}
			if !ok {
				return
			}
			if got.Volume == nil || *got.Volume != c.volume {
				t.Fatalf("Parse(%q) Volume = %v, want %v", c.dir, got.Volume, c.volume)
			}
			if got.Chapter != c.chapter || got.Title != c.title || got.Special != "" || got.Group != "" || got.Lang != "" {
				t.Fatalf("Parse(%q) = %+v, want chapter %v, title %q", c.dir, got, c.chapter, c.title)
			}
		})
	}
}

// The registry reads a name with a volume as one, and a name with only a chapter as the other, whichever
// parser comes first.
func TestTheRegistryReadsTheJapaneseVolumeAndChapterNames(t *testing.T) {
	reg := DefaultRegistry()
	cases := []struct {
		dir    string
		parser string
		volume bool
	}{
		{"第1巻 第2話", "japanese-volume-chapter", true},
		{"1巻 2話", "japanese-volume-chapter", true},
		{"(一般コミック) [奥浩哉] いぬやしき 第09巻", "japanese-volume", true},
		{"Series Name v01", "volume-only", true},
		{"Love Hina - Volume 01 [Scans]", "volume-only", true},
		{"第2話", "japanese-chapter", false},
		{"#7", "japanese-chapter", false},
	}
	for _, c := range cases {
		got, name, ok := reg.Parse(c.dir)
		if !ok || name != c.parser || (got.Volume != nil) != c.volume {
			t.Errorf("Parse(%q) = %+v, %q, %v; want parser %q, volume %v", c.dir, got, name, ok, c.parser, c.volume)
		}
	}
}
