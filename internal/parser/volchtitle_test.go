package parser

import "testing"

func TestVolChTitleParser(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		dir  string
		ok   bool
		want ParsedChapter
	}{
		{
			name: "full",
			dir:  "Vol.01 Ch.0001 - A Dog and a Chainsaw (en) [Mangastream]",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 1, Title: "A Dog and a Chainsaw", Lang: "en", Group: "Mangastream"},
		},
		{
			name: "apostrophe in title",
			dir:  "Vol.01 Ch.0002 - Pochita's Whereabouts (en) [Mangastream]",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 2, Title: "Pochita's Whereabouts", Lang: "en", Group: "Mangastream"},
		},
		{
			name: "decimal chapter",
			dir:  "Vol.12 Ch.0105.5 - Interlude (en) [Group]",
			ok:   true,
			want: ParsedChapter{Volume: f(12), Chapter: 105.5, Title: "Interlude", Lang: "en", Group: "Group"},
		},
		{
			name: "special suffix",
			dir:  "Vol.03 Ch.0021x1 - Bonus Story (en) [Group]",
			ok:   true,
			want: ParsedChapter{Volume: f(3), Chapter: 21, Special: "x1", Title: "Bonus Story", Lang: "en", Group: "Group"},
		},
		{
			name: "no group",
			dir:  "Vol.01 Ch.0001 - Title (en)",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 1, Title: "Title", Lang: "en"},
		},
		{
			name: "no lang or group",
			dir:  "Vol.01 Ch.0001 - Title",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 1, Title: "Title"},
		},
		{
			name: "language with region code",
			dir:  "Vol.01 Ch.0001 - Title (pt-br) [Group]",
			ok:   true,
			want: ParsedChapter{Volume: f(1), Chapter: 1, Title: "Title", Lang: "pt-br", Group: "Group"},
		},
		{
			name: "not this convention: chapter only",
			dir:  "Chapter 5",
			ok:   false,
		},
		{
			name: "not this convention: unrelated folder",
			dir:  "Chainsaw Man - Vol.01",
			ok:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := VolChTitleParser{}.Parse(c.dir)
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
