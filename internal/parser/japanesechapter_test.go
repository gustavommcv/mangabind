package parser

import "testing"

func TestJapaneseChapterParser(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		ok   bool
		want ParsedChapter
	}{
		{name: "第N話", dir: "第1話", ok: true, want: ParsedChapter{Chapter: 1}},
		{name: "zero padded", dir: "第001話", ok: true, want: ParsedChapter{Chapter: 1}},
		{name: "no 第", dir: "12話", ok: true, want: ParsedChapter{Chapter: 12}},
		{name: "decimal", dir: "第10.5話", ok: true, want: ParsedChapter{Chapter: 10.5}},
		{name: "full-width digits", dir: "第１２話", ok: true, want: ParsedChapter{Chapter: 12}},
		{name: "full-width zero padding and decimal", dir: "第００１０．５話", ok: true, want: ParsedChapter{Chapter: 10.5}},
		{name: "space inside", dir: "第 3 話", ok: true, want: ParsedChapter{Chapter: 3}},
		{name: "title after a space", dir: "第1話 始まり", ok: true, want: ParsedChapter{Chapter: 1, Title: "始まり"}},
		{name: "title after an ideographic space", dir: "第2話　出会い", ok: true, want: ParsedChapter{Chapter: 2, Title: "出会い"}},
		{name: "title after a dash", dir: "第3話 - 別れ", ok: true, want: ParsedChapter{Chapter: 3, Title: "別れ"}},
		{name: "title after a full-width colon", dir: "第4話：再会", ok: true, want: ParsedChapter{Chapter: 4, Title: "再会"}},
		{name: "title right after the counter", dir: "第5話旅立ち", ok: true, want: ParsedChapter{Chapter: 5, Title: "旅立ち"}},
		{name: "title keeps its own brackets", dir: "第6話「約束」", ok: true, want: ParsedChapter{Chapter: 6, Title: "「約束」"}},
		{name: "title with trailing spaces", dir: "第7話 夜明け  ", ok: true, want: ParsedChapter{Chapter: 7, Title: "夜明け"}},
		{name: "#N", dir: "#3", ok: true, want: ParsedChapter{Chapter: 3}},
		{name: "#N zero padded", dir: "#001", ok: true, want: ParsedChapter{Chapter: 1}},
		{name: "#N with a decimal", dir: "#12.5", ok: true, want: ParsedChapter{Chapter: 12.5}},
		{name: "#N full-width", dir: "#１２", ok: true, want: ParsedChapter{Chapter: 12}},
		{name: "#N with a title", dir: "#12 Title", ok: true, want: ParsedChapter{Chapter: 12, Title: "Title"}},
		{name: "#N with a title after a dash", dir: "#12 - 始まり", ok: true, want: ParsedChapter{Chapter: 12, Title: "始まり"}},
		{name: "not this convention: a range", dir: "第1-3話", ok: false},
		{name: "not this convention: no number", dir: "第話", ok: false},
		{name: "not this convention: a volume", dir: "第1巻", ok: false},
		{name: "after an author and a title", dir: "[作者] タイトル 第1話", ok: true, want: ParsedChapter{Chapter: 1}},
		{name: "after a real name, with an ideographic space", dir: "[ハレム]ナナとカオル ～高校生のSMごっこ～\u3000第10話", ok: true, want: ParsedChapter{Chapter: 10}},
		{name: "after a title, no space before 第", dir: "タイトル第3話 旅立ち", ok: true, want: ParsedChapter{Chapter: 3, Title: "旅立ち"}},
		{name: "after a bracket, no space", dir: "[ハレム]第4話", ok: true, want: ParsedChapter{Chapter: 4}},
		{name: "after a scanlator and an underscore", dir: "Raw_第1話 始まり", ok: true, want: ParsedChapter{Chapter: 1, Title: "始まり"}},
		{name: "after a title, N話 without 第", dir: "ナナとカオル 12話", ok: true, want: ParsedChapter{Chapter: 12}},
		{name: "after a title that has a number in it", dir: "Re:ゼロ 2期 7話", ok: true, want: ParsedChapter{Chapter: 7}},
		{name: "the leftmost marker is the chapter", dir: "タイトル 第1話 第2話", ok: true, want: ParsedChapter{Chapter: 1, Title: "第2話"}},
		{name: "not this convention: N話 stuck to the title", dir: "タイトル12話", ok: false},
		{name: "not this convention: a hash after a title", dir: "Batman #12", ok: false},
		{name: "not this convention: a range after a title", dir: "タイトル 第1-3話", ok: false},
		{name: "not this convention: a volume after a title", dir: "[作者] タイトル 第1巻", ok: false},
		{name: "not this convention: an English volume before it", dir: "Vol.01 第1話", ok: false},
		{name: "not this convention: a short English volume before it", dir: "v01 12話", ok: false},
		{name: "not this convention: a range of volumes before it", dir: "タイトル 第1-3巻 第2話", ok: false},
		{name: "not this convention: a range of chapters before it", dir: "第1-3話 第5話", ok: false},
		{name: "not this convention: an English chapter before it", dir: "Ch.5 第6話", ok: false},
		{name: "not this convention: a full-width volume before it", dir: "タイトル 第１巻 第２話 ", ok: false},
		{name: "not this convention: a hash with letters stuck to it", dir: "#12abc", ok: false},
		{name: "not this convention: a hash alone", dir: "#", ok: false},
		{name: "not this convention: the English convention", dir: "Chapter 5", ok: false},
		{name: "not this convention: a bare number", dir: "5", ok: false},
		{name: "not this convention: empty", dir: "", ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := JapaneseChapterParser{}.Parse(c.dir)
			if ok != c.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v (got %+v)", c.dir, ok, c.ok, got)
			}
			if !ok {
				return
			}
			if got.Chapter != c.want.Chapter || got.Title != c.want.Title || got.Special != "" || got.Group != "" || got.Lang != "" {
				t.Fatalf("Parse(%q) = %+v, want %+v", c.dir, got, c.want)
			}
			if got.Volume != nil {
				t.Fatalf("Parse(%q) Volume = %v, want nil", c.dir, *got.Volume)
			}
		})
	}
}
