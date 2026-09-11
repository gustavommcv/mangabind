package naturalsort

import (
	"reflect"
	"testing"
)

func TestLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"2.jpg", "10.jpg", true},
		{"10.jpg", "2.jpg", false},
		{"page2", "page10", true},
		{"Ch.0001", "Ch.0002", true},
		{"Ch.0010", "Ch.0002", false},
		{"Ch.2", "Ch.10", true},
		{"a", "a", false},
		{"a", "b", true},
		{"file", "file2", true},
		{"file10a", "file10b", true},
		{"001.jpg", "01.jpg", false}, // equal numeric value, "001" == "01" -> falls through to length tiebreak
	}

	for _, c := range cases {
		if got := Less(c.a, c.b); got != c.want {
			t.Errorf("Less(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestStrings(t *testing.T) {
	in := []string{"10.jpg", "1.jpg", "2.jpg", "20.jpg", "3.jpg"}
	want := []string{"1.jpg", "2.jpg", "3.jpg", "10.jpg", "20.jpg"}

	Strings(in)

	if !reflect.DeepEqual(in, want) {
		t.Errorf("Strings() = %v, want %v", in, want)
	}
}
