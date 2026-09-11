package main

import (
	"path/filepath"
	"testing"
)

func TestDefaultOutputDir(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{
			input: filepath.Join("C:", "Users", "me", "Manga", "Chainsaw Man"),
			want:  filepath.Join("C:", "Users", "me", "Manga", "Chainsaw Man (mangabind)"),
		},
		{
			input: filepath.Join("manga", "One Piece"),
			want:  filepath.Join("manga", "One Piece (mangabind)"),
		},
	}
	for _, c := range cases {
		if got := defaultOutputDir(c.input); got != c.want {
			t.Errorf("defaultOutputDir(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
