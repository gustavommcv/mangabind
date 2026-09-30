package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mangabind.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAndLookup(t *testing.T) {
	path := writeFile(t, `{
		"schema_version": 1,
		"manga": {"title": "Example Manga"},
		"volumes": [
			{"number": "1", "chapters": ["1-7"]},
			{"number": "2", "chapters": ["8-16", "8.5", "21x1"]}
		],
		"source": {"provider": "external", "id": "abc123"}
	}`)

	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		chapter float64
		special string
		want    float64
	}{
		{1, "", 1},
		{7, "", 1},
		{8, "", 2},
		{16, "", 2},
		{8.5, "", 2},
		{21, "x1", 2},
	}
	for _, c := range cases {
		got, ok := m.Lookup(c.chapter, c.special)
		if !ok {
			t.Errorf("Lookup(%v, %q): not found, want volume %v", c.chapter, c.special, c.want)
			continue
		}
		if got != c.want {
			t.Errorf("Lookup(%v, %q) = %v, want %v", c.chapter, c.special, got, c.want)
		}
	}

	if _, ok := m.Lookup(17, ""); ok {
		t.Error("Lookup(17, \"\") should not be found - not covered by the file")
	}
	if _, ok := m.Lookup(21, ""); ok {
		t.Error("Lookup(21, \"\") should not be found - only 21x1 is listed, not plain 21")
	}
}

// Other tools keep their own facts about a manga in this file (Mangabound
// stores manga.author and manga.language, see the README), so keys this
// package has never heard of must not change how the file is read.
func TestLoadIgnoresKeysItDoesNotKnow(t *testing.T) {
	path := writeFile(t, `{
		"schema_version": 1,
		"manga": {"title": "Example Manga", "author": "Some Author", "language": "en"},
		"volumes": [{"number": "1", "chapters": ["1-7"]}],
		"another_tool": {"anything": true}
	}`)

	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := m.Lookup(3, ""); !ok || got != 1 {
		t.Errorf("Lookup(3, \"\") = %v, %v, want volume 1", got, ok)
	}

	// Only those keys and no volumes is a valid file that maps nothing.
	empty, err := Load(writeFile(t, `{
		"schema_version": 1,
		"manga": {"author": "Some Author", "language": "pt-br"},
		"volumes": []
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := empty.Lookup(1, ""); ok {
		t.Error("a file with no volumes should not map any chapter")
	}
}

func TestLoadRejectsUnsupportedSchemaVersion(t *testing.T) {
	path := writeFile(t, `{"schema_version": 2, "volumes": []}`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for an unsupported schema_version")
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	path := writeFile(t, `{not valid json`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestLoadRejectsInvalidVolumeNumber(t *testing.T) {
	path := writeFile(t, `{"schema_version": 1, "volumes": [{"number": "one", "chapters": ["1"]}]}`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for a non-numeric volume number")
	}
}

func TestLoadRejectsInvalidChapterToken(t *testing.T) {
	cases := []string{"abc", "1-2-3", "5-1", "-1", "1x"}
	for _, tok := range cases {
		t.Run(tok, func(t *testing.T) {
			path := writeFile(t, `{"schema_version": 1, "volumes": [{"number": "1", "chapters": ["`+tok+`"]}]}`)
			if _, err := Load(path); err == nil {
				t.Fatalf("expected an error for invalid chapter token %q", tok)
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json")); err == nil {
		t.Fatal("expected an error loading a nonexistent file")
	}
}
