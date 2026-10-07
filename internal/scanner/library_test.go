package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLibraryListsTheFoldersAndOnlyThem(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"Beta", "Alpha", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	manga, links, err := Library(root)

	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".hidden", "Alpha", "Beta"}; !reflect.DeepEqual(manga, want) {
		t.Errorf("manga = %v, want %v, in the order of the folder's names", manga, want)
	}
	if len(links) != 0 {
		t.Errorf("links = %v, want none", links)
	}
}

func TestLibraryDoesNotFollowALinkToAFolderAndSaysSo(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	symlink(t, "Alpha", filepath.Join(root, "InsideLink"))
	symlink(t, outside, filepath.Join(root, "OutsideLink"))
	symlink(t, filepath.Join(root, "gone"), filepath.Join(root, "Ghost"))
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlink(t, "notes.txt", filepath.Join(root, "NotesLink")) // a link to a file: not a manga, no word

	manga, links, err := Library(root)

	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Alpha"}; !reflect.DeepEqual(manga, want) {
		t.Errorf("manga = %v, want only the real folder %v", manga, want)
	}
	got := map[string]string{}
	for _, link := range links {
		got[filepath.Base(link.Path)] = link.Why
	}
	if len(got) != 3 {
		t.Fatalf("links = %v, want InsideLink, OutsideLink and Ghost", links)
	}
	for _, name := range []string{"InsideLink", "OutsideLink"} {
		if !strings.Contains(got[name], "leads to a folder") {
			t.Errorf("%s: %q, want it to say the link leads to a folder", name, got[name])
		}
	}
	if got["Ghost"] != "leads nowhere" {
		t.Errorf("Ghost: %q, want %q", got["Ghost"], "leads nowhere")
	}
}

func TestLibraryOfAFolderThatDoesNotExistIsAnError(t *testing.T) {
	if _, _, err := Library(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("Library() of a missing folder succeeded")
	}
}
