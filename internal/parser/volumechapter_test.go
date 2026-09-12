package parser

import "testing"

func TestVolumeChapterParser(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		dir  string
		ok   bool
		want ParsedChapter
	}{
		{
			name: "spelled out, no title",
			dir:  "Vol 1 Chapter 5",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 5},
		},
		{
			name: "Volume + abbreviated Ch. with dash",
			dir:  "Volume 1 - Ch. 005",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 5},
		},
		{
			name: "full form with title, lang, group",
			dir:  "Volume 01 Chapter 5 - Title Here (en) [Group]",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 5, Title: "Title Here", Lang: "en", Group: "Group"},
		},
		{
			name: "decimal chapter",
			dir:  "Vol 3 Chapter 12.5",
			ok:   true,
			want: ParsedChapter{Volume: f(3), Chapter: 12.5},
		},
		{
			name: "special suffix",
			dir:  "Vol 2 Ch 21x1",
			ok:   true,
			want: ParsedChapter{Volume: f(2), Chapter: 21, Special: "x1"},
		},
		{
			name: "colon title separator",
			dir:  "Volume 1 Chapter 5: Arrival",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 5, Title: "Arrival"},
		},
		{
			name: "also matches the abbreviated dotted form (harmless overlap)",
			dir:  "Vol.01 Ch.0001 - Title (en) [Group]",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 1, Title: "Title", Lang: "en", Group: "Group"},
		},
		{
			name: "not this convention: chapter only, no volume",
			dir:  "Chapter 5",
			ok:   false,
		},
		{
			name: "not this convention: unrelated",
			dir:  "Cover",
			ok:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := VolumeChapterParser{}.Parse(c.dir)
			if ok != c.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v", c.dir, ok, c.ok)
			}
			if !ok {
				return
			}
			if got.Chapter != c.want.Chapter || got.Special != c.want.Special ||
				got.Title != c.want.Title || got.Lang != c.want.Lang || got.Group != c.want.Group {
				t.Fatalf("Parse(%q) = %+v, want %+v", c.dir, got, c.want)
			}
			if (got.Volume == nil) != (c.want.Volume == nil) {
				t.Fatalf("Parse(%q) Volume = %v, want %v", c.dir, got.Volume, c.want.Volume)
			}
			if got.Volume != nil && *got.Volume != *c.want.Volume {
				t.Fatalf("Parse(%q) Volume = %v, want %v", c.dir, *got.Volume, *c.want.Volume)
			}
		})
	}
}
