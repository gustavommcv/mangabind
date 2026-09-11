// Command mangabind reorganizes a chapter-by-chapter manga download into one .cbz per volume.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gustavommcv/Mangabind/internal/cbz"
	"github.com/gustavommcv/Mangabind/internal/grouper"
	"github.com/gustavommcv/Mangabind/internal/parser"
	"github.com/gustavommcv/Mangabind/internal/scanner"
)

func main() {
	input := flag.String("input", "", "directory containing one folder per downloaded chapter")
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
	dirs, err := scanner.Scan(input)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", input, err)
	}

	reg := parser.DefaultRegistry()
	var chapters []grouper.Chapter
	var unparsed []string

	for _, d := range dirs {
		parsed, _, ok := reg.Parse(d.Name)
		if !ok {
			unparsed = append(unparsed, d.Name)
			continue
		}

		pages, err := scanner.Pages(d.Path)
		if err != nil {
			return fmt.Errorf("listing pages in %s: %w", d.Path, err)
		}
		if len(pages) == 0 {
			unparsed = append(unparsed, d.Name)
			continue
		}

		chapters = append(chapters, grouper.Chapter{Parsed: parsed, Dir: d.Path, Pages: pages})
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

	for _, name := range result.Unparsed {
		fmt.Fprintf(os.Stderr, "warning: could not parse chapter folder name, skipped: %q\n", name)
	}
	for _, c := range result.Unassigned {
		fmt.Fprintf(os.Stderr, "warning: chapter has no volume number, skipped: %q\n", filepath.Base(c.Dir))
	}
	for _, g := range result.Gaps {
		fmt.Fprintf(os.Stderr, "warning: volume %v is missing chapter(s) between %v and %v\n", g.Volume, g.After, g.Before)
	}
	for _, c := range result.Conflicts {
		fmt.Fprintf(os.Stderr, "warning: volume %v chapter %v%s has %d conflicting sources, all skipped:\n",
			c.Volume, c.Chapter, c.Special, len(c.Dirs))
		for _, d := range c.Dirs {
			fmt.Fprintf(os.Stderr, "  - %q\n", filepath.Base(d))
		}
	}

	return nil
}
