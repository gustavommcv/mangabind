package parser

import "testing"

func TestEpisodeParser(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		ok   bool
		want ParsedChapter
	}{
		{name: "Episode N", dir: "Episode 5", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "Ep. N", dir: "Ep. 5", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "epNN", dir: "ep05", ok: true, want: ParsedChapter{Chapter: 5}},
		{name: "decimal", dir: "Episode 12.5", ok: true, want: ParsedChapter{Chapter: 12.5}},
		{name: "special suffix", dir: "Episode 3x1", ok: true, want: ParsedChapter{Chapter: 3, Special: "x1"}},
		{name: "title, language and group", dir: "Ep.07 - A Title (en) [Group]", ok: true, want: ParsedChapter{Chapter: 7, Title: "A Title", Lang: "en", Group: "Group"}},

		// What a downloader or a person puts before it: the name of the series.
		{name: "after an underscore", dir: "peppercarrot_episode01", ok: true, want: ParsedChapter{Chapter: 1}},
		{name: "after a space", dir: "Pepper Carrot Episode 7", ok: true, want: ParsedChapter{Chapter: 7}},
		{name: "after a dash", dir: "Pepper & Carrot - Episode 07 - The Witch", ok: true, want: ParsedChapter{Chapter: 7, Title: "The Witch"}},
		{name: "after a bracket", dir: "[Group] Series Episode 3", ok: true, want: ParsedChapter{Chapter: 3}},
		{name: "full-width digits", dir: "Episode　０５", ok: true, want: ParsedChapter{Chapter: 5}},

		// What it is not.
		{name: "a word that only begins like the marker", dir: "Epic 5", ok: false},
		{name: "the plural", dir: "Episodes 5", ok: false},
		{name: "a number stuck to a letter before it", dir: "Seriesepisode5", ok: false},
		{name: "no number", dir: "Episode One", ok: false},
		{name: "after a prefix that holds a volume", dir: "Vol.01 Episode 3", ok: false},
		{name: "after a prefix that holds a chapter", dir: "Ch.2 Episode 3", ok: false},
		{name: "unrelated", dir: "Cover", ok: false},
		{name: "a chapter is not an episode", dir: "Chapter 5", ok: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := EpisodeParser{}.Parse(c.dir)
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

func TestAnEpisodeIsReadByTheRegistryAfterTheConventionsThatCarryAVolume(t *testing.T) {
	reg := DefaultRegistry()

	got, name, ok := reg.Parse("peppercarrot_episode01")
	if !ok || name != "episode" || got.Chapter != 1 {
		t.Fatalf("Parse = %+v, %q, %v; want chapter 1 by the episode parser", got, name, ok)
	}
	// A name that says a volume and a chapter is still read by the parser that knows both.
	got, name, ok = reg.Parse("Vol.02 Ch.010")
	if !ok || name == "episode" || got.Volume == nil || *got.Volume != 2 {
		t.Fatalf("Parse = %+v, %q, %v; want volume 2 not by the episode parser", got, name, ok)
	}
}
