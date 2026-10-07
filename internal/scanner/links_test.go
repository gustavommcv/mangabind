package scanner

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// symlink makes a symbolic link, or skips the test where the system does not
// allow one (Windows without Developer Mode or an elevated shell). The Linux
// and macOS runs make them everywhere.
func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("this system does not allow a symbolic link here: %v", err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// layout is a manga folder with one chapter, and a folder beside it that the
// manga's links are not meant to reach.
type layout struct {
	root    string // the manga folder
	chapter string // a chapter folder inside it
	outside string // a folder next to the manga folder, with a file in it
}

func newLayout(t *testing.T) layout {
	t.Helper()
	base := t.TempDir()
	l := layout{
		root:    filepath.Join(base, "Manga"),
		chapter: filepath.Join(base, "Manga", "Vol.01 Ch.001"),
		outside: filepath.Join(base, "outside"),
	}
	write(t, filepath.Join(l.chapter, "001.png"), "own page")
	write(t, filepath.Join(l.outside, "private.png"), "not part of the manga")
	return l
}

func linkPaths(links []Link) []string {
	var paths []string
	for _, link := range links {
		paths = append(paths, filepath.Base(link.Path))
	}
	return paths
}

func TestPagesFollowALinkToAFileInsideTheManga(t *testing.T) {
	l := newLayout(t)
	write(t, filepath.Join(l.root, "shared", "credits.png"), "shared page")
	symlink(t, filepath.Join(l.root, "shared", "credits.png"), filepath.Join(l.chapter, "002.png"))
	symlink(t, filepath.Join("..", "shared", "credits.png"), filepath.Join(l.chapter, "003.png"))

	pages, _, links, err := Pages(l.root, l.chapter)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"001.png", "002.png", "003.png"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("pages = %v, want %v", pages, want)
	}
	if len(links) != 0 {
		t.Fatalf("links = %v, want none", links)
	}
}

func TestPagesSkipALinkToAFileOutsideTheManga(t *testing.T) {
	l := newLayout(t)
	symlink(t, filepath.Join(l.outside, "private.png"), filepath.Join(l.chapter, "002.png"))
	symlink(t, filepath.Join("..", "..", "outside", "private.png"), filepath.Join(l.chapter, "003.png"))

	pages, _, links, err := Pages(l.root, l.chapter)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"001.png"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("pages = %v, want %v: nothing the links lead to may be listed", pages, want)
	}
	if want := []string{"002.png", "003.png"}; !reflect.DeepEqual(linkPaths(links), want) {
		t.Fatalf("links = %v, want %v", links, want)
	}
	for _, link := range links {
		if link.Why != whyOutside {
			t.Errorf("%s: why = %q, want %q", link.Path, link.Why, whyOutside)
		}
		if filepath.Dir(link.Path) != l.chapter {
			t.Errorf("link path = %q, want it to be in %q", link.Path, l.chapter)
		}
	}
}

func TestPagesFollowEveryLinkOnTheWay(t *testing.T) {
	l := newLayout(t)
	// A link inside the manga to a link that leaves it, and a link through a
	// folder inside the manga that is itself a link to the outside: the real
	// path of both is outside, whatever the first step looks like.
	symlink(t, filepath.Join(l.outside, "private.png"), filepath.Join(l.root, "stepping-stone.png"))
	symlink(t, filepath.Join(l.root, "stepping-stone.png"), filepath.Join(l.chapter, "002.png"))
	symlink(t, l.outside, filepath.Join(l.root, "door"))
	symlink(t, filepath.Join(l.root, "door", "private.png"), filepath.Join(l.chapter, "003.png"))

	pages, _, links, err := Pages(l.root, l.chapter)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"001.png"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("pages = %v, want %v", pages, want)
	}
	if want := []string{"002.png", "003.png"}; !reflect.DeepEqual(linkPaths(links), want) {
		t.Fatalf("links = %v, want %v", links, want)
	}
}

func TestPagesSkipALinkThatLeadsToNoFile(t *testing.T) {
	l := newLayout(t)
	if err := os.MkdirAll(filepath.Join(l.root, "a folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	symlink(t, filepath.Join(l.root, "nothing-here.png"), filepath.Join(l.chapter, "002.png"))
	symlink(t, filepath.Join(l.root, "a folder"), filepath.Join(l.chapter, "003.png"))
	symlink(t, l.outside, filepath.Join(l.chapter, "004.png"))

	pages, _, links, err := Pages(l.root, l.chapter)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"001.png"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("pages = %v, want %v", pages, want)
	}
	why := map[string]string{}
	for _, link := range links {
		why[filepath.Base(link.Path)] = link.Why
	}
	// A folder outside is said to lead outside, which is the more useful thing to know.
	want := map[string]string{"002.png": whyNotAFile, "003.png": whyNotAFile, "004.png": whyOutside}
	if !reflect.DeepEqual(why, want) {
		t.Fatalf("why = %v, want %v", why, want)
	}
}

func TestPagesCompareWithTheRealPathOfTheManga(t *testing.T) {
	l := newLayout(t)
	write(t, filepath.Join(l.root, "shared", "credits.png"), "shared page")
	symlink(t, filepath.Join(l.root, "shared", "credits.png"), filepath.Join(l.chapter, "002.png"))
	symlink(t, filepath.Join(l.outside, "private.png"), filepath.Join(l.chapter, "003.png"))
	// The manga is reached through a link of its own, as when its folder was
	// given by the name of a shortcut.
	shortcut := filepath.Join(filepath.Dir(l.root), "shortcut")
	symlink(t, l.root, shortcut)

	pages, _, links, err := Pages(shortcut, filepath.Join(shortcut, filepath.Base(l.chapter)))
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"001.png", "002.png"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("pages = %v, want %v", pages, want)
	}
	if want := []string{"003.png"}; !reflect.DeepEqual(linkPaths(links), want) {
		t.Fatalf("links = %v, want %v", links, want)
	}
}

func TestPagesSayRootCouldNotBeChecked(t *testing.T) {
	l := newLayout(t)
	symlink(t, filepath.Join(l.root, "Vol.01 Ch.001", "001.png"), filepath.Join(l.chapter, "002.png"))

	_, _, links, err := Pages(filepath.Join(l.root, "gone"), l.chapter)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 || links[0].Why != whyNoInput {
		t.Fatalf("links = %v, want the one link, unchecked", links)
	}
}

func TestScanFollowsAnArchiveLinkInsideTheMangaOnly(t *testing.T) {
	l := newLayout(t)
	inside := filepath.Join(l.root, "archives", "chapter.cbz")
	outside := filepath.Join(l.outside, "stolen.cbz")
	for _, path := range []string{inside, outside} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		if _, err := zw.Create("01.jpg"); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	symlink(t, inside, filepath.Join(l.root, "Vol.02 Ch.002.cbz"))
	symlink(t, outside, filepath.Join(l.root, "Vol.02 Ch.003.cbz"))
	symlink(t, l.outside, filepath.Join(l.root, "Vol.02 Ch.004.cbz"))

	entries, skipped, links, err := Scan(l.root)
	if err != nil {
		t.Fatal(err)
	}

	var archives []string
	for _, e := range entries {
		if e.IsArchive {
			archives = append(archives, e.Name)
		}
	}
	if want := []string{"Vol.02 Ch.002"}; !reflect.DeepEqual(archives, want) {
		t.Fatalf("archives = %v, want %v", archives, want)
	}
	if want := []string{"Vol.02 Ch.003.cbz", "Vol.02 Ch.004.cbz"}; !reflect.DeepEqual(linkPaths(links), want) {
		t.Fatalf("links = %v, want %v", links, want)
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped = %v: a link is told of as a link, not as an unrecognized file", skipped)
	}
}

func TestScanLeavesALinkThatIsNotAnArchiveAsItWas(t *testing.T) {
	l := newLayout(t)
	symlink(t, filepath.Join(l.outside, "private.png"), filepath.Join(l.root, "cover.png"))
	symlink(t, l.outside, filepath.Join(l.root, "Vol.03 Ch.005"))

	entries, skipped, links, err := Scan(l.root)
	if err != nil {
		t.Fatal(err)
	}

	// Neither is a chapter, as before: they are unrecognized files, not links
	// that were followed or refused.
	if want := []string{"Vol.01 Ch.001"}; len(entries) != 1 || entries[0].Name != want[0] {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	if want := []string{"Vol.03 Ch.005", "cover.png"}; !reflect.DeepEqual(skipped, want) {
		t.Fatalf("skipped = %v, want %v", skipped, want)
	}
	if len(links) != 0 {
		t.Fatalf("links = %v, want none", links)
	}
}

func TestInside(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "library", "Manga")
	cases := map[string]bool{
		filepath.Join(root, "a.png"):           true,
		filepath.Join(root, "Vol.01", "a.png"): true,
		root:                                   true,
		filepath.Join(string(filepath.Separator), "library"):                   false,
		filepath.Join(string(filepath.Separator), "library", "Manga2"):         false,
		filepath.Join(string(filepath.Separator), "library", "Other", "a.png"): false,
		filepath.Join(root, "..", "Other", "a.png"):                            false,
	}
	for path, want := range cases {
		if got := inside(root, path); got != want {
			t.Errorf("inside(%q, %q) = %v, want %v", root, path, got, want)
		}
	}
	// Relative to nothing in common, as two Windows drives are.
	if inside(root, "relative.png") {
		t.Error("a relative path has nothing in common with an absolute root")
	}
}
