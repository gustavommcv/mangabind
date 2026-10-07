package metadata

import (
	"fmt"
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
