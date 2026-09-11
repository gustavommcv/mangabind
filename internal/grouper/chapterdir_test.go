package grouper

import "testing"

func TestChapterDirName(t *testing.T) {
	cases := []struct {
		position int
		title    string
		want     string
	}{
		{1, "A Dog and a Chainsaw", "c001 - A Dog and a Chainsaw"},
		{12, "Interlude", "c012 - Interlude"},
		{1, "", "c001"},
		{2, "Weird/Title\\With Slashes", "c002 - Weird-Title-With Slashes"},
		{3, "  Padded  ", "c003 - Padded"},
	}

	for _, c := range cases {
		if got := chapterDirName(c.position, c.title); got != c.want {
			t.Errorf("chapterDirName(%d, %q) = %q, want %q", c.position, c.title, got, c.want)
		}
	}
}
