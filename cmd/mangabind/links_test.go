package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// hostileManga makes a manga named Hostile in parent, whose chapter holds one
// page of its own and two links to files in outside, which is not part of it:
// one link by an absolute path and one by a relative path.
func hostileManga(t *testing.T, parent, outside string) string {
	t.Helper()
	manga := filepath.Join(parent, "Hostile")
	chapter := filepath.Join(manga, "Vol.01 Ch.001")
	writeFile(t, filepath.Join(chapter, "001.png"), "its own page")
	writeFile(t, filepath.Join(outside, "private.png"), "SECRET-OUTSIDE-IMAGE")
	writeFile(t, filepath.Join(outside, "id_fake"), "-----BEGIN FAKE KEY-----")
	symlink(t, filepath.Join(outside, "private.png"), filepath.Join(chapter, "002.png"))
	relative, err := filepath.Rel(chapter, filepath.Join(outside, "id_fake"))
	if err != nil {
		t.Fatal(err)
	}
	symlink(t, relative, filepath.Join(chapter, "003.png"))
	return manga
}

// archiveContents reads every entry of a .cbz as text, by entry name.
func archiveContents(t *testing.T, path string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	contents := map[string]string{}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		contents[f.Name] = string(data)
	}
	return contents
}

func TestALinkLeadingOutsideTheInputIsNotPutInTheVolume(t *testing.T) {
	base := t.TempDir()
	manga := hostileManga(t, base, filepath.Join(base, "outside"))
	output := filepath.Join(base, "out")

	report, _, exitCode := runMachineForTest(t, "-input", manga, "-output", output, "-json")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0: the manga still makes a volume", exitCode)
	}
	manga0 := report.Manga[0]
	if manga0.Status != "completed_with_warnings" {
		t.Fatalf("status = %q, want completed_with_warnings", manga0.Status)
	}
	if len(manga0.Volumes) != 1 || manga0.Volumes[0].PageCount != 1 {
		t.Fatalf("volumes = %+v, want one volume of the one page that is the manga's own", manga0.Volumes)
	}
	if manga0.Units[0].PageCount != 1 {
		t.Fatalf("unit page count = %d, want 1", manga0.Units[0].PageCount)
	}

	contents := archiveContents(t, manga0.Volumes[0].OutputPath)
	if len(contents) != 1 {
		t.Fatalf("archive entries = %v, want only the page of its own", contents)
	}
	for name, data := range contents {
		if strings.Contains(data, "SECRET") || strings.Contains(data, "FAKE KEY") {
			t.Fatalf("%s carries a file from outside the input: %q", name, data)
		}
	}

	var links []machineIssue
	for _, issue := range manga0.Issues {
		if issue.Code == "link_skipped" {
			links = append(links, issue)
		}
	}
	if len(links) != 2 {
		t.Fatalf("issues = %+v, want one link_skipped for each of the two links", manga0.Issues)
	}
	for _, issue := range links {
		if issue.Severity != "warning" || issue.Stage != "inspect" || issue.Manga != "Hostile" {
			t.Errorf("issue = %+v, want a warning at inspect for the manga", issue)
		}
		if issue.Chapter == nil || *issue.Chapter != 1 {
			t.Errorf("issue chapter = %v, want chapter 1", issue.Chapter)
		}
		if issue.Volume == nil || *issue.Volume != 1 {
			t.Errorf("issue volume = %v, want volume 1, which the chapter is in", issue.Volume)
		}
		if !strings.HasPrefix(issue.Path, absolutePath(manga)) || !strings.Contains(issue.Message, "leads outside the input folder") {
			t.Errorf("issue = %+v, want the link's own path and why it was skipped", issue)
		}
	}
	if manga0.Summary.Warnings != 2 {
		t.Fatalf("warnings = %d, want 2", manga0.Summary.Warnings)
	}
}

func TestALinkLeadingOutsideIsToldInThePlanToo(t *testing.T) {
	base := t.TempDir()
	manga := hostileManga(t, base, filepath.Join(base, "outside"))
	output := filepath.Join(base, "out")

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", output, "-dry-run", "-json")

	if report.Manga[0].Volumes[0].PageCount != 1 {
		t.Fatalf("planned pages = %d, want the 1 that will be written", report.Manga[0].Volumes[0].PageCount)
	}
	if report.Manga[0].Summary.Warnings != 2 {
		t.Fatalf("warnings = %d, want 2", report.Manga[0].Summary.Warnings)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("a plan wrote %s", output)
	}
}

func TestALinkIsToldToAPersonToo(t *testing.T) {
	base := t.TempDir()
	manga := hostileManga(t, base, filepath.Join(base, "outside"))
	var stdout, stderr bytes.Buffer

	if _, err := processMangaWithOutput(manga, filepath.Join(base, "out"), "", false, false, false, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"002.png", "003.png"} {
		want := "warning: found a link \"" + name + "\" that leads outside the input folder, skipped"
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want it to say %q", stderr.String(), want)
		}
	}
}

func TestAChapterOfNothingButLinksOutsideIsLeftOutAsEmpty(t *testing.T) {
	base := t.TempDir()
	manga := filepath.Join(base, "Hostile")
	outside := filepath.Join(base, "outside")
	writeFile(t, filepath.Join(outside, "private.png"), "SECRET-OUTSIDE-IMAGE")
	writeFile(t, filepath.Join(manga, "Vol.01 Ch.001", "001.png"), "its own page")
	if err := os.MkdirAll(filepath.Join(manga, "Vol.01 Ch.002"), 0o755); err != nil {
		t.Fatal(err)
	}
	symlink(t, filepath.Join(outside, "private.png"), filepath.Join(manga, "Vol.01 Ch.002", "001.png"))

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", filepath.Join(base, "out"), "-json")

	codes := map[string]int{}
	for _, issue := range report.Manga[0].Issues {
		codes[issue.Code]++
	}
	if codes["link_skipped"] != 1 || codes["empty_chapter"] != 1 {
		t.Fatalf("issue codes = %v, want the link and the chapter it left empty, each once", codes)
	}
	if report.Manga[0].Volumes[0].PageCount != 1 {
		t.Fatalf("pages = %d, want only the page of chapter 1", report.Manga[0].Volumes[0].PageCount)
	}
}

func TestAnArchiveLinkLeadingOutsideIsNotAChapter(t *testing.T) {
	base := t.TempDir()
	manga := filepath.Join(base, "Hostile")
	outside := filepath.Join(base, "outside")
	writeFile(t, filepath.Join(manga, "Vol.01 Ch.001", "001.png"), "its own page")
	outsideArchive := filepath.Join(outside, "stolen.cbz")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(outsideArchive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("01.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("SECRET-OUTSIDE-IMAGE")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	symlink(t, outsideArchive, filepath.Join(manga, "Vol.01 Ch.002.cbz"))

	report, _, _ := runMachineForTest(t, "-input", manga, "-output", filepath.Join(base, "out"), "-json")

	manga0 := report.Manga[0]
	if len(manga0.Units) != 1 || manga0.Volumes[0].PageCount != 1 {
		t.Fatalf("units = %+v, volumes = %+v, want only the folder chapter", manga0.Units, manga0.Volumes)
	}
	var link *machineIssue
	for i := range manga0.Issues {
		if manga0.Issues[i].Code == "link_skipped" {
			link = &manga0.Issues[i]
		}
	}
	if link == nil || link.Chapter != nil || filepath.Base(link.Path) != "Vol.01 Ch.002.cbz" {
		t.Fatalf("issues = %+v, want a link_skipped for the archive, in no chapter", manga0.Issues)
	}
}

func TestALibraryBindsTheOtherMangaWhenOneHasALinkOutside(t *testing.T) {
	base := t.TempDir()
	library := filepath.Join(base, "Library")
	hostileManga(t, library, filepath.Join(base, "outside"))
	makeChapter(t, filepath.Join(library, "Fine"), "Vol.01 Ch.0001 - Alpha (en) [Group]", 2)

	report, _, exitCode := runMachineForTest(t, "-input", library, "-batch", "-output", filepath.Join(base, "out"), "-json")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	pages := map[string]int{}
	warnings := map[string]int{}
	for _, m := range report.Manga {
		for _, v := range m.Volumes {
			pages[m.Name] += v.PageCount
		}
		warnings[m.Name] = m.Summary.Warnings
	}
	if pages["Fine"] != 2 || pages["Hostile"] != 1 {
		t.Fatalf("pages = %v, want 2 for Fine and 1 for Hostile", pages)
	}
	if warnings["Fine"] != 0 || warnings["Hostile"] != 2 {
		t.Fatalf("warnings = %v, want none for Fine and the two links for Hostile", warnings)
	}
}

func TestALinkInsideTheMangaIsStillFollowed(t *testing.T) {
	base := t.TempDir()
	manga := filepath.Join(base, "Linked")
	chapter := filepath.Join(manga, "Vol.01 Ch.001")
	writeFile(t, filepath.Join(chapter, "001.png"), "its own page")
	writeFile(t, filepath.Join(manga, "shared", "credits.png"), "a page shared by two chapters")
	symlink(t, filepath.Join(manga, "shared", "credits.png"), filepath.Join(chapter, "002.png"))

	report, _, exitCode := runMachineForTest(t, "-input", manga, "-output", filepath.Join(base, "out"), "-json")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	// The folder the shared page is kept in is not a chapter, and is told of as
	// such; the link itself is not told of at all.
	for _, issue := range report.Manga[0].Issues {
		if issue.Code == "link_skipped" {
			t.Fatalf("issues = %+v, want no link skipped: it stays inside the manga", report.Manga[0].Issues)
		}
	}
	contents := archiveContents(t, report.Manga[0].Volumes[0].OutputPath)
	if len(contents) != 2 {
		t.Fatalf("archive entries = %v, want both pages", contents)
	}
}
