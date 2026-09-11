package parser

import "testing"

func TestChapterOnlyParser(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		ok   bool
		want ParsedChapter
	}{
		{name: "Chapter N", dir: "Chapter 5", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "Ch. N", dir: "Ch. 5", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "Ch.N no space", dir: "Ch.5", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "cNNN zero padded", dir: "c005", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "decimal", dir: "Chapter 12.5", ok: true, want: ParsedChapter{Chapter: 12.5}},
		{name: "special suffix", dir: "Chapter 21x1", ok: true, want: ParsedChapter{Chapter: 21, Special: "x1"}},
		{name: "with title, dash", dir: "Chapter 5 - Arrival", ok: true, want: ParsedChapter{Chapter: 5, Title: "Arrival"}},
		{name: "with title, colon", dir: "Chapter 5: Arrival", ok: true, want: ParsedChapter{Chapter: 5, Title: "Arrival"}},
		{name: "not this convention: has volume", dir: "Vol.01 Ch.0001 - Title (en) [Group]", ok: false},
		{name: "not this convention: unrelated", dir: "Cover"},
		{name: "not this convention: extras folder", dir: "Chainsaw Man - Vol.01", ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ChapterOnlyParser{}.Parse(c.dir)
			if ok != c.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v", c.dir, ok, c.ok)
			}
			if !ok {
				return
			}
			if got.Chapter != c.want.Chapter || got.Special != c.want.Special || got.Title != c.want.Title {
				t.Fatalf("Parse(%q) = %+v, want %+v", c.dir, got, c.want)
			}
			if got.Volume != nil {
				t.Fatalf("Parse(%q) Volume = %v, want nil", c.dir, *got.Volume)
			}
		})
	}
}
