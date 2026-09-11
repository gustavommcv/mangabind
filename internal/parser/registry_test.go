package parser

import "testing"

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()

	cases := []struct {
		dir        string
		wantParser string
		wantOK     bool
	}{
		{"Vol.01 Ch.0001 - A Dog and a Chainsaw (en) [Mangastream]", "vol-ch-title", true},
		{"Chapter 5", "chapter-only", true},
		{"total garbage that matches nothing 42", "", false},
	}

	for _, c := range cases {
		_, name, ok := r.Parse(c.dir)
		if ok != c.wantOK {
			t.Errorf("Parse(%q) ok = %v, want %v", c.dir, ok, c.wantOK)
			continue
		}
		if ok && name != c.wantParser {
			t.Errorf("Parse(%q) parser = %q, want %q", c.dir, name, c.wantParser)
		}
	}
}

func TestRegistryTriesMostSpecificFirst(t *testing.T) {
	// A name that both parsers could plausibly touch should resolve via
	// vol-ch-title, since it's registered first and is the more specific
	// convention (it requires an explicit volume number).
	r := DefaultRegistry()
	_, name, ok := r.Parse("Vol.01 Ch.0001 - Title (en) [Group]")
	if !ok || name != "vol-ch-title" {
		t.Fatalf("got parser %q, ok %v; want vol-ch-title, true", name, ok)
	}
}
