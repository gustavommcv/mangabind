package metadata

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
)

func volumesFile(volumes ...string) string {
	return fmt.Sprintf(`{"schema_version": 1, "volumes": [%s]}`, strings.Join(volumes, ","))
}

func volume(number string, chapters ...string) string {
	quoted := make([]string, len(chapters))
	for i, c := range chapters {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	return fmt.Sprintf(`{"number": %q, "chapters": [%s]}`, number, strings.Join(quoted, ","))
}

func TestLoadRefusesAVolumeNumberThatCannotBeAVolume(t *testing.T) {
	for _, number := range []string{"NaN", "nan", "Inf", "+Inf", "-Inf", "infinity", "-3", "-0.5", "1e300", "100001", "100000.5", "abc", ""} {
		_, err := Load(writeFile(t, volumesFile(volume(number, "1"))))
		if err == nil {
			t.Errorf("volume %q was accepted", number)
			continue
		}
		if !strings.Contains(err.Error(), "invalid number") {
			t.Errorf("volume %q: error = %v, want it to say the number is invalid", number, err)
		}
	}
}

func TestLoadAcceptsVolumeNumbersAtTheEdges(t *testing.T) {
	for _, number := range []string{"0", "0.5", "1", "12.5", "1e2", "100000"} {
		if _, err := Load(writeFile(t, volumesFile(volume(number, "1")))); err != nil {
			t.Errorf("volume %q was refused: %v", number, err)
		}
	}
}

func TestARangeThatIsTooLargeIsRefusedBeforeAnythingIsAllocatedForIt(t *testing.T) {
	// This line used to ask for 216 GB and end the program with an
	// out-of-memory crash.
	path := writeFile(t, volumesFile(volume("1", "1-9000000000")))
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	_, err := Load(path)

	runtime.ReadMemStats(&after)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("error = %v, want one that says the range is too large", err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 32<<20 {
		t.Errorf("refusing the range allocated %d MiB", allocated>>20)
	}
}

func TestAFileMayListAsManyChaptersAsTheCapAndNoMore(t *testing.T) {
	cases := []struct {
		name    string
		volumes []string
		ok      bool
	}{
		{"exactly the cap", []string{volume("1", "1-100000")}, true},
		{"one more than the cap", []string{volume("1", "1-100001")}, false},
		{"two ranges that add up to more", []string{volume("1", "1-60000"), volume("2", "60001-120000")}, false},
		{"a chapter after the cap is used up", []string{volume("1", "1-100000", "100001")}, false},
		{"a range that overflows an int when counted", []string{volume("1", "0-9223372036854775807")}, false},
	}
	for _, c := range cases {
		_, err := Load(writeFile(t, volumesFile(c.volumes...)))
		if (err == nil) != c.ok {
			t.Errorf("%s: error = %v, want ok = %v", c.name, err, c.ok)
		}
	}
}

func TestARangeEndingAtTheLargestIntHasOnlyItsDeclaredChapters(t *testing.T) {
	cases := []struct {
		name          string
		lo, hi        int
		budget, count int
	}{
		{"zero", 0, 0, 1, 1},
		{"ordinary range", 3, 5, 3, 3},
		{"one chapter at the int limit", math.MaxInt, math.MaxInt, 1, 1},
		{"two chapters ending at the int limit", math.MaxInt - 1, math.MaxInt, 2, 2},
		{"range just below the int limit", math.MaxInt - 3, math.MaxInt - 1, 4, 3},
		{"the full budget ending at the int limit", math.MaxInt - maxChapters + 1, math.MaxInt, maxChapters, maxChapters},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			token := fmt.Sprintf("%d-%d", c.lo, c.hi)
			keys, err := expandChapterToken(token, c.budget)
			if err != nil {
				t.Fatal(err)
			}
			if len(keys) != c.count {
				t.Fatalf("range %q expanded to %d chapters, want %d", token, len(keys), c.count)
			}
			for i, key := range keys {
				want := chapterKey{chapter: float64(c.lo + i)}
				if key != want {
					t.Fatalf("chapter %d = %+v, want %+v", i, key, want)
				}
			}
		})
	}
}

func TestARangeAtTheIntLimitStillUsesTheChapterBudget(t *testing.T) {
	for _, c := range []struct {
		count, budget int
	}{{1, 0}, {2, 1}, {maxChapters + 1, maxChapters}} {
		token := fmt.Sprintf("%d-%d", math.MaxInt-c.count+1, math.MaxInt)
		keys, err := expandChapterToken(token, c.budget)
		if err == nil || !strings.Contains(err.Error(), "too large") || keys != nil {
			t.Errorf("range %q with budget %d: keys = %v, error = %v, want refusal before expansion", token, c.budget, keys, err)
		}
	}
}

func TestAChapterListedUnderTwoVolumesIsReportedAndTheLastListingWins(t *testing.T) {
	path := writeFile(t, volumesFile(
		volume("1", "1-10"),
		volume("2", "5-6", "5"), // chapter 5 twice in volume 2 is not a conflict
		volume("3", "6", "8x1"),
		volume("4", "8x1"),
	))

	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []Duplicate{
		{Chapter: 5, Earlier: 1, Later: 2},
		{Chapter: 6, Earlier: 1, Later: 2},
		{Chapter: 6, Earlier: 2, Later: 3},
		{Chapter: 8, Special: "x1", Earlier: 3, Later: 4},
	}
	got := m.Duplicates()
	if len(got) != len(want) {
		t.Fatalf("duplicates = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("duplicate %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	for _, c := range []struct {
		chapter float64
		special string
		volume  float64
	}{{5, "", 2}, {6, "", 3}, {8, "x1", 4}, {1, "", 1}} {
		if v, ok := m.Lookup(c.chapter, c.special); !ok || v != c.volume {
			t.Errorf("Lookup(%v, %q) = %v, %v, want volume %v: the last listing", c.chapter, c.special, v, ok, c.volume)
		}
	}
}

func TestListingAChapterTwiceInTheSameVolumeIsNotADuplicate(t *testing.T) {
	m, err := Load(writeFile(t, volumesFile(volume("1", "1-3", "2", "1-3"))))
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Duplicates(); len(got) != 0 {
		t.Errorf("duplicates = %+v, want none", got)
	}
}
