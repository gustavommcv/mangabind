// Command mangabind reorganizes a chapter-by-chapter manga download into one .cbz per volume.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gustavommcv/mangabind/internal/cbz"
	"github.com/gustavommcv/mangabind/internal/grouper"
	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

func main() {
	input := flag.String("input", "", "directory containing one folder or .cbz file per downloaded chapter")
	output := flag.String("output", "", "directory to write the generated .cbz files into")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: mangabind --input <dir> --output <dir>")
		os.Exit(2)
	}

	if err := run(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, "mangabind:", err)
		os.Exit(1)
	}
}

func run(input, output string) error {
	entries, skipped, err := scanner.Scan(input)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", input, err)
	}
	if len(entries) == 0 {
		fmt.Printf("mangabind: no chapter folders or .cbz files found in %s\n", input)
	}
	for _, name := range skipped {
		fmt.Fprintln(os.Stderr, "warning:", unsupportedFileMessage(name))
	}

	reg := parser.DefaultRegistry()
	var chapters []grouper.Chapter
	var unparsed []string

	for _, e := range entries {
		parsed, _, ok := reg.Parse(e.Name)
		if !ok {
			unparsed = append(unparsed, e.Name)
			continue
		}

		var pages []string
		if e.IsArchive {
			pages, err = scanner.PagesInArchive(e.Path)
		} else {
			pages, err = scanner.Pages(e.Path)
		}
		if err != nil {
			return fmt.Errorf("listing pages in %s: %w", e.Path, err)
		}
		if len(pages) == 0 {
			unparsed = append(unparsed, e.Name)
			continue
		}

		chapters = append(chapters, grouper.Chapter{Parsed: parsed, Path: e.Path, Pages: pages, IsArchive: e.IsArchive})
	}

	result := grouper.Group(chapters, unparsed)
	mangaName := filepath.Base(input)

	for _, vol := range result.Volumes {
		outPath := filepath.Join(output, cbz.VolumeFileName(mangaName, vol.Number))
		if err := cbz.Write(outPath, vol.Pages); err != nil {
			return fmt.Errorf("writing volume %v: %w", vol.Number, err)
		}
		fmt.Printf("wrote %s (%d pages)\n", outPath, len(vol.Pages))
	}
	if len(entries) > 0 && len(result.Volumes) == 0 {
		fmt.Println("mangabind: no volumes produced - see warnings below")
	}

	for _, name := range result.Unparsed {
		fmt.Fprintf(os.Stderr, "warning: could not parse chapter name, skipped: %q\n", name)
	}
	for _, c := range result.Unassigned {
		fmt.Fprintf(os.Stderr, "warning: chapter has no volume number, skipped: %q\n", filepath.Base(c.Path))
	}
	for _, g := range result.Gaps {
		fmt.Fprintf(os.Stderr, "warning: volume %v is missing chapter(s) between %v and %v\n", g.Volume, g.After, g.Before)
	}
	for _, c := range result.Conflicts {
		fmt.Fprintf(os.Stderr, "warning: volume %v chapter %v%s has %d conflicting sources, all skipped:\n",
			c.Volume, c.Chapter, c.Special, len(c.Sources))
		for _, s := range c.Sources {
			fmt.Fprintf(os.Stderr, "  - %q\n", filepath.Base(s))
		}
	}

	return nil
}

// unsupportedFileMessage explains why a file found alongside chapter
// folders/.cbz files wasn't treated as a chapter - naming the two HakuNeko
// download formats Mangabind doesn't support explicitly, so the user knows
// this isn't a bug, rather than getting no feedback at all.
func unsupportedFileMessage(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".epub":
		return fmt.Sprintf("found %q - .epub chapters aren't supported (only folders of images or .cbz), skipped", name)
	case ".pdf":
		return fmt.Sprintf("found %q - .pdf chapters aren't supported (only folders of images or .cbz), skipped", name)
	default:
		return fmt.Sprintf("found unrecognized file, skipped: %q", name)
	}
}
