package parser

import "testing"

func TestBareNumberParser(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		ok   bool
		want float64
	}{
		{name: "zero padded", dir: "001", ok: true, want: 1},
		{name: "no padding", dir: "12", ok: true, want: 12},
		{name: "decimal", dir: "012.5", ok: true, want: 12.5},
		{name: "zero", dir: "000", ok: true, want: 0},
		{name: "not this convention: has text", dir: "Chapter 5", ok: false},
		{name: "not this convention: has group tag", dir: "005 [Group]", ok: false},
		{name: "not this convention: empty", dir: "", ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := BareNumberParser{}.Parse(c.dir)
			if ok != c.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v", c.dir, ok, c.ok)
			}
			if ok && got.Chapter != c.want {
				t.Fatalf("Parse(%q) Chapter = %v, want %v", c.dir, got.Chapter, c.want)
			}
			if ok && got.Volume != nil {
				t.Fatalf("Parse(%q) Volume = %v, want nil", c.dir, *got.Volume)
			}
		})
	}
}
