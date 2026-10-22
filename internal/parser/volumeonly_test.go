package parser

import "testing"

func TestVolumeOnlyParser(t *testing.T) {
	cases := []struct {
		name   string
		dir    string
		ok     bool
		volume float64
	}{
		{name: "v01", dir: "v01", ok: true, volume: 1},
		{name: "Vol.01", dir: "Vol.01", ok: true, volume: 1},
		{name: "Volume 1", dir: "Volume 1", ok: true, volume: 1},
		{name: "vol 02", dir: "vol 02", ok: true, volume: 2},
		{name: "upper case", dir: "VOLUME 12", ok: true, volume: 12},
		{name: "decimal", dir: "v03.5", ok: true, volume: 3.5},
		{name: "full-width digits", dir: "Vol.０１", ok: true, volume: 1},
		{name: "the folder mangadex-downloader makes for a volume", dir: "Volume. 1", ok: true, volume: 1},
		{name: "the name of a volume whose number has a dot", dir: "Volume. 12", ok: true, volume: 12},
		{name: "not this convention: the folder of chapters with no volume", dir: "No Volume", ok: false},
		{name: "Kavita's documented name", dir: "Series Name v01", ok: true, volume: 1},
		{name: "after a title and a dash", dir: "Love Hina - Volume 01 [Scans]", ok: true, volume: 1},
		{name: "after a comma, with a year and a publisher", dir: "Battle Royale, v01 (2000) [TokyoPop] [Manga-Sketchbook]", ok: true, volume: 1},
		{name: "with an edition in braces before it", dir: "Dance in the Vampire Bund {Special Edition} v03.5 (2019) (Digital) (KG Manga)", ok: true, volume: 3.5},
		{name: "with an edition before it", dir: "Highschool of the Dead - Full Color Edition v02 [Uasaha] (Yen Press)", ok: true, volume: 2},
		{name: "after an underscore", dir: "Series_v04", ok: true, volume: 4},
		{name: "not this convention: a title after the volume", dir: "Samurai Jack Vol. 01 - The threads of Time", ok: false},
		{name: "not this convention: a chapter after the volume", dir: "Series v02 c003 - The Invisible Guardian (2) [JS Scans]", ok: false},
		{name: "not this convention: a chapter before the volume", dir: "Series Ch.5 v01", ok: false},
		{name: "not this convention: a second volume in the prefix", dir: "Pokémon Adventures v04 - Yellow v01 (AnHeroGold-Empire)", ok: false},
		{name: "not this convention: the volume in brackets", dir: "Title - c001 (v01) [Group]", ok: false},
		{name: "not this convention: v stuck to a letter", dir: "Dev5", ok: false},
		{name: "not this convention: a v inside a word", dir: "Marvel 5", ok: false},
		{name: "not this convention: no number", dir: "Volume", ok: false},
		{name: "not this convention: a bare number", dir: "5", ok: false},
		{name: "not this convention: empty", dir: "", ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := VolumeOnlyParser{}.Parse(c.dir)
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
