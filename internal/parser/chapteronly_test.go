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

		// What the MangaDex downloaders write when a chapter has no volume: the name of a chapter without a
		// title ends at its language, and a title is followed by the language and the group.
		{name: "language only", dir: "Ch.0001 (en)", ok: true, want: ParsedChapter{Chapter: 1, Lang: "en"}},
		{name: "language only, spelled out", dir: "Chapter 12 (ja)", ok: true, want: ParsedChapter{Chapter: 12, Lang: "ja"}},
		{name: "language with a region", dir: "Ch.0007 (pt-br)", ok: true, want: ParsedChapter{Chapter: 7, Lang: "pt-br"}},
		{name: "language of three letters", dir: "Ch.5 (fil)", ok: true, want: ParsedChapter{Chapter: 5, Lang: "fil"}},
		{name: "language written in capitals", dir: "Ch.5 (EN)", ok: true, want: ParsedChapter{Chapter: 5, Lang: "EN"}},
		{name: "decimal chapter and language", dir: "Ch.0105.5 (en)", ok: true, want: ParsedChapter{Chapter: 105.5, Lang: "en"}},
		{name: "special suffix and language", dir: "Ch.0021x1 (en)", ok: true, want: ParsedChapter{Chapter: 21, Special: "x1", Lang: "en"}},
		{name: "title and language", dir: "Ch.0006 - A Perfect Day for Koala (en)", ok: true, want: ParsedChapter{Chapter: 6, Title: "A Perfect Day for Koala", Lang: "en"}},
		{
			name: "title, language and a group with brackets in its name",
			dir:  "Ch.0011 - My lady is fond of her old butler!？ (en) [sirtetris (Tarek)]",
			ok:   true,
			want: ParsedChapter{Chapter: 11, Title: "My lady is fond of her old butler!？", Lang: "en", Group: "sirtetris (Tarek)"},
		},
		{
			name: "title, language and a group",
			dir:  "Ch.0015 - Have you seen Ubuntu Touch!？ (en) [prussian blue hedgehog scans]",
			ok:   true,
			want: ParsedChapter{Chapter: 15, Title: "Have you seen Ubuntu Touch!？", Lang: "en", Group: "prussian blue hedgehog scans"},
		},
		{name: "group without a language", dir: "Ch.5 [Scans]", ok: true, want: ParsedChapter{Chapter: 5, Group: "Scans"}},
		{name: "title and group without a language", dir: "Chapter 5 - Arrival [Scans]", ok: true, want: ParsedChapter{Chapter: 5, Title: "Arrival", Group: "Scans"}},
		{
			name: "a title that ends in a note keeps it, and the language comes after",
			dir:  "Ch.0016 - Miss Ichinomiya꞉ Battles without Honor and Humanity! (+2 short Extras) (en)",
			ok:   true,
			want: ParsedChapter{Chapter: 16, Title: "Miss Ichinomiya꞉ Battles without Honor and Humanity! (+2 short Extras)", Lang: "en"},
		},
		{name: "a bracket that is not a language stays in the title", dir: "Chapter 5 - Arrival (Part 2)", ok: true, want: ParsedChapter{Chapter: 5, Title: "Arrival (Part 2)"}},
		{name: "a longer word in brackets stays in the title", dir: "Chapter 5 - Arrival (Final Chapter)", ok: true, want: ParsedChapter{Chapter: 5, Title: "Arrival (Final Chapter)"}},
		{name: "a word of five letters is not a language", dir: "Chapter 5 - Arrival (Final)", ok: true, want: ParsedChapter{Chapter: 5, Title: "Arrival (Final)"}},
		{name: "no title and a bracket that is not a language", dir: "Chapter 5 (Final)", ok: false},
		{name: "a word after the number with no dash", dir: "Chapter 5 Arrival", ok: false},
		{name: "a language before the number", dir: "(en) Ch.5", ok: false},
		{name: "a language that is not closed", dir: "Ch.5 (en", ok: false},
		{name: "a chapter with a volume is not this convention, whatever follows", dir: "Vol.01 Ch.0001 (en)", ok: false},
		{name: "a word that only begins like the marker", dir: "Chronicles 5 (en)", ok: false},
		{name: "full-width digits", dir: "Chapter　５", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "full-width zero padding", dir: "c００５", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "full-width decimal", dir: "Ch.１２．５", ok: true, want: ParsedChapter{Chapter: 12.5}},
		{name: "full-width special suffix", dir: "Chapter 21x１", ok: true, want: ParsedChapter{Chapter: 21, Special: "x1"}},
		{name: "title after an ideographic space", dir: "Chapter 5\u3000-\u3000Ｔｉｔｌｅ", ok: true, want: ParsedChapter{Chapter: 5, Title: "Ｔｉｔｌｅ"}},
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
			if got.Chapter != c.want.Chapter || got.Special != c.want.Special || got.Title != c.want.Title ||
				got.Lang != c.want.Lang || got.Group != c.want.Group {
				t.Fatalf("Parse(%q) = %+v, want %+v", c.dir, got, c.want)
			}
			if got.Volume != nil {
				t.Fatalf("Parse(%q) Volume = %v, want nil", c.dir, *got.Volume)
			}
		})
	}
}
