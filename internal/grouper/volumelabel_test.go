package grouper

import "testing"

func TestVolumeLabel(t *testing.T) {
	cases := []struct {
		number float64
		want   string
	}{
		{0, "00"},
		{1, "01"},
		{12, "12"},
		{100, "100"},
		{1.5, "1.5"},
		{0.25, "0.25"},
		{1e14, "100000000000000"},
		// Too large for an int: written out, not converted (it used to give
		// "-9223372036854775808").
		{1e15, "1000000000000000"},
		{1e20, "100000000000000000000"},
		{1e300, "1" + zeros(300)},
	}
	for _, c := range cases {
		if got := VolumeLabel(c.number); got != c.want {
			t.Errorf("VolumeLabel(%v) = %q, want %q", c.number, got, c.want)
		}
	}
}

func zeros(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '0'
	}
	return string(b)
}

func TestVolumeDirNameUsesTheSameLabel(t *testing.T) {
	if got, want := volumeDirName(2, 1e20), "v002 - Vol.100000000000000000000"; got != want {
		t.Errorf("volumeDirName = %q, want %q", got, want)
	}
	if got, want := volumeDirName(1, 3), "v001 - Vol.03"; got != want {
		t.Errorf("volumeDirName = %q, want %q", got, want)
	}
}
