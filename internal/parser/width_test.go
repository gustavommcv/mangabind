package parser

import "testing"

func TestFoldWidth(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"12.5", "12.5"},
		{"０１２３４５６７８９", "0123456789"},
		{"１２．５", "12.5"},
		{"x１", "x1"},
		// Only digits and the full stop are folded: letters, the space and punctuation are text.
		{"Ｖｏｌ　１！", "Ｖｏｌ　1！"},
	}
	for _, c := range cases {
		if got := foldWidth(c.in); got != c.want {
			t.Errorf("foldWidth(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseNumber(t *testing.T) {
	for in, want := range map[string]float64{"5": 5, "０５": 5, "12.5": 12.5, "１２．５": 12.5, "0１2.５": 12.5} {
		got, err := parseNumber(in)
		if err != nil || got != want {
			t.Errorf("parseNumber(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := parseNumber("Ａ"); err == nil {
		t.Error("parseNumber(Ａ) read a letter as a number")
	}
}

func TestWiden(t *testing.T) {
	got := widen(`^\d+\s*x`)
	if want := `^[0-9０-９]+[\s　]*x`; got != want {
		t.Fatalf("widen = %q, want %q", got, want)
	}
}
