package parser

import "testing"

func TestJapaneseVolumeParser(t *testing.T) {
	cases := []struct {
		name   string
		dir    string
		ok     bool
		volume float64
	}{
		{name: "第N巻", dir: "第1巻", ok: true, volume: 1},
		{name: "zero padded", dir: "第09巻", ok: true, volume: 9},
		{name: "no 第", dir: "12巻", ok: true, volume: 12},
		{name: "decimal", dir: "第1.5巻", ok: true, volume: 1.5},
		{name: "full-width digits", dir: "第１０巻", ok: true, volume: 10},
		{name: "space inside", dir: "第 3 巻", ok: true, volume: 3},
		{name: "the category, author and title of a scanned book", dir: "(一般コミック) [奥浩哉] いぬやしき 第09巻", ok: true, volume: 9},
		{name: "a bracket lost", dir: "諌山創] 進撃の巨人 第23巻", ok: true, volume: 23},
		{name: "N巻 after a title", dir: "Test 5巻", ok: true, volume: 5},
		{name: "a publisher after it", dir: "いぬやしき 第09巻 (講談社)", ok: true, volume: 9},
		{name: "a publisher and a year after it", dir: "いぬやしき 第09巻 (講談社) [2020]", ok: true, volume: 9},
		{name: "full-width parentheses and corner brackets after it", dir: "いぬやしき 第09巻（講談社）【分冊版】", ok: true, volume: 9},
		{name: "not this convention: a range of volumes", dir: "Series 10-15巻", ok: false},
		{name: "not this convention: a range with 第", dir: "第1-3巻", ok: false},
		{name: "not this convention: the total of a set", dir: "いぬやしき 全10巻", ok: false},
		{name: "not this convention: N巻 stuck to the title", dir: "タイトル2巻", ok: false},
		{name: "not this convention: a title after the volume", dir: "第1巻 おまけ", ok: false},
		{name: "not this convention: a chapter after the volume", dir: "第1巻 第2話", ok: false},
		{name: "not this convention: a prefix that holds another volume", dir: "第1巻 第2巻", ok: false},
		{name: "not this convention: a chapter", dir: "第1話", ok: false},
		{name: "not this convention: no number", dir: "巻", ok: false},
		{name: "not this convention: the English convention", dir: "Vol.01", ok: false},
		{name: "not this convention: empty", dir: "", ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := JapaneseVolumeParser{}.Parse(c.dir)
			if ok != c.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v (got %+v)", c.dir, ok, c.ok, got)
			}
			if !ok {
				return
			}
			if got.Volume == nil || *got.Volume != c.volume {
				t.Fatalf("Parse(%q) Volume = %v, want %v", c.dir, got.Volume, c.volume)
			}
			if !got.VolumeUnit {
				t.Fatalf("Parse(%q) VolumeUnit = false, want true", c.dir)
			}
			if got.Chapter != 1 || got.Title != "" || got.Special != "" || got.Group != "" || got.Lang != "" {
				t.Fatalf("Parse(%q) = %+v, want chapter 1 and nothing else", c.dir, got)
			}
		})
	}
}
