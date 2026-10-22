package parser

import "testing"

func TestAfterPrefixTriesTheNameAndThenEachPlaceAPrefixMayEnd(t *testing.T) {
	var tried []string
	parse := func(s string) (ParsedChapter, bool) {
		tried = append(tried, s)
		return ParsedChapter{}, false
	}

	afterPrefix("[A]b c_d第e　f", parse)

	want := []string{"[A]b c_d第e　f", "b c_d第e　f", "c_d第e　f", "d第e　f", "第e　f", "f"}
	if len(tried) != len(want) {
		t.Fatalf("tried %q, want %q", tried, want)
	}
	for i := range want {
		if tried[i] != want[i] {
			t.Errorf("tried %q, want %q", tried, want)
			break
		}
	}
}

func TestAfterPrefixOfNothingParsesNothing(t *testing.T) {
	if _, ok := afterPrefix("", func(string) (ParsedChapter, bool) { return ParsedChapter{}, true }); ok {
		t.Fatal("an empty name was parsed")
	}
}
