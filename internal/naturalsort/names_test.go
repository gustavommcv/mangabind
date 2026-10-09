package naturalsort

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// groups reads testdata/natsort_vectors.txt: each group is a list of names in
// the order the real natsort library gives them.
func groups(t *testing.T) [][]string {
	t.Helper()
	data, err := os.ReadFile("testdata/natsort_vectors.txt")
	if err != nil {
		t.Fatal(err)
	}
	var out [][]string
	for _, block := range strings.Split(string(data), "---\n")[1:] {
		var names []string
		for _, line := range strings.Split(block, "\n") {
			if line != "" {
				names = append(names, line)
			}
		}
		out = append(out, names)
	}
	return out
}

func TestEachNameComesBeforeTheNextInItsGroup(t *testing.T) {
	all := groups(t)
	if len(all) < 50 {
		t.Fatalf("expected at least 50 groups, read %d", len(all))
	}
	for _, group := range all {
		for i := 0; i+1 < len(group); i++ {
			if !LessName(group[i], group[i+1]) || LessName(group[i+1], group[i]) {
				t.Errorf("%q should come before %q (group %q)", group[i], group[i+1], group)
			}
		}
	}
}

func TestSortingAGroupBackToFrontGivesTheOrderNatsortGives(t *testing.T) {
	for _, group := range groups(t) {
		names := make([]string, len(group))
		for i, name := range group {
			names[len(group)-1-i] = name
		}
		Names(names)
		if !reflect.DeepEqual(names, group) {
			t.Errorf("got %q, want %q", names, group)
		}
	}
}

func sortedNames(names ...string) []string {
	Names(names)
	return names
}

func TestANameComesBeforeTheNamesThatContinueIt(t *testing.T) {
	got := sortedNames("p01-2.png", "p01 (2).png", "p01.png", "p01_b.png")
	want := []string{"p01.png", "p01 (2).png", "p01-2.png", "p01_b.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := sortedNames("cover2.png", "cover.png"), []string{"cover.png", "cover2.png"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestANumberInsideANameIsNotAnExtension(t *testing.T) {
	got := sortedNames("2.png", "1.10.png", "1.5.png", "1.png")
	want := []string{"1.png", "1.5.png", "1.10.png", "2.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDigitsOfEveryScriptAreNumbers(t *testing.T) {
	got := sortedNames("１０.png", "3.png", "２.png", "1.png")
	want := []string{"1.png", "２.png", "3.png", "１０.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	got = sortedNames("第１０話", "第２話", "第１話")
	want = []string{"第１話", "第２話", "第１０話"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	// Digits of two scripts next to each other are one number.
	if LessName("1２.png", "12.png") || LessName("12.png", "1２.png") {
		t.Error("1２ and 12 should be the same number")
	}
}

func TestACharacterThatStandsForADigitIsANumberByItself(t *testing.T) {
	got := sortedNames("x3.png", "x².png", "x1.png")
	want := []string{"x1.png", "x².png", "x3.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	// A circled ten is not a digit, so it sorts after the text "x" and any digit.
	if !LessName("x9.png", "x⑩.png") {
		t.Error("x9 should come before x⑩")
	}
}

func TestOnlyShortExtensionsAreSplitOffAndAtMostTwo(t *testing.T) {
	cases := map[string][]string{
		"a.tar.gz":    {".tar", ".gz"},
		"a.b.c.d.png": {".d", ".png"},
		"a.5.png":     {".png"},
		"a.longer":    nil,
		".png":        nil,
		"Vol. 1":      {". 1"},
		"noext":       nil,
	}
	for name, want := range cases {
		if got := extensions(name); !reflect.DeepEqual(got, want) {
			t.Errorf("extensions(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNumbersOfAnySizeCompareByValue(t *testing.T) {
	big := "p" + strings.Repeat("9", 40) + ".png"
	bigger := "p1" + strings.Repeat("0", 40) + ".png"
	if !LessName(big, bigger) || LessName(bigger, big) {
		t.Error("a 40-digit number should be less than a 41-digit one")
	}
	if LessName("p007.png", "p7.png") || LessName("p7.png", "p007.png") {
		t.Error("p007 and p7 should be the same")
	}
}

func TestCaseIsIgnored(t *testing.T) {
	got := sortedNames("Chapter 10", "chapter 2", "Chapter 1", "b.jpg", "A.jpg")
	want := []string{"A.jpg", "b.jpg", "Chapter 1", "chapter 2", "Chapter 10"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAFilePathComesBeforeAFolderPathAtTheSameLevel(t *testing.T) {
	paths := []string{"Chapter 2/001.png", "zz-credits.png", "Chapter 1/002.png", "cover.png", "Chapter 1/001.png"}
	Paths(paths)
	want := []string{"cover.png", "zz-credits.png", "Chapter 1/001.png", "Chapter 1/002.png", "Chapter 2/001.png"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("got %q, want %q", paths, want)
	}
	if !LessPath("z.png", "a/1.png") || LessPath("a/1.png", "z.png") {
		t.Error("a file should come before a folder even when it sorts after it by name")
	}
	if !LessPath("a", "a/1.png") || LessPath("a/1.png", "a") {
		t.Error("a prefix should come first")
	}
	if LessPath("a/1.png", "a/1.png") {
		t.Error("equal paths are not less")
	}
}

func TestANumberIsNeverComparedWithANumberOfAnotherLevel(t *testing.T) {
	if !LessPath("Ch 2/10.png", "Ch 10/2.png") {
		t.Error("the folder decides first")
	}
}

func TestDecimalValues(t *testing.T) {
	for r, want := range map[rune]int{'0': 0, '9': 9, '０': 0, '９': 9, '٣': 3, '०': 0, '๕': 5, '𝟕': 7} {
		if got, ok := decimalValue(r); !ok || got != want {
			t.Errorf("decimalValue(%q) = %d, %v; want %d", r, got, ok, want)
		}
	}
	for _, r := range []rune{'a', ' ', '.', '話', '²', '①', 'Ⅳ', '½'} {
		if _, ok := decimalValue(r); ok {
			t.Errorf("%q is not a decimal digit", r)
		}
	}
}

func TestEveryDecimalDigitTheUnicodePackageKnowsIsWorthItsPlaceInItsBlock(t *testing.T) {
	// Each block of decimal digits is ten characters in a row; the first is zero.
	for _, rg := range unicode.Nd.R16 {
		if (rg.Hi-rg.Lo+1)%10 != 0 {
			t.Errorf("range %#x-%#x is not whole blocks of ten", rg.Lo, rg.Hi)
		}
	}
	for _, rg := range unicode.Nd.R32 {
		if (rg.Hi-rg.Lo+1)%10 != 0 {
			t.Errorf("range %#x-%#x is not whole blocks of ten", rg.Lo, rg.Hi)
		}
	}
}

func TestSingleValues(t *testing.T) {
	for r, want := range map[rune]int{'²': 2, '³': 3, '¹': 1, '⁰': 0, '⁹': 9, '₀': 0, '①': 1, '⑨': 9, '⓪': 0} {
		if got, ok := singleValue(r); !ok || got != want {
			t.Errorf("singleValue(%q) = %d, %v; want %d", r, got, ok, want)
		}
	}
	for _, r := range []rune{'a', '1', '⑩', '½'} {
		if _, ok := singleValue(r); ok {
			t.Errorf("%q does not stand for a digit", r)
		}
	}
}
