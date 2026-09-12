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

// version is set at build/release time via -ldflags "-X main.version=...";
// "dev" is what a plain `go build`/`go run` produces.
var version = "dev"

func main() {
	var input, output string
	var batch, quiet, dryRun, showVersion bool

	flag.StringVar(&input, "input", "", "directory to reorganize: a manga's chapters, or (with -batch) a library of manga folders")
	flag.StringVar(&input, "i", "", "shorthand for -input")
	flag.StringVar(&output, "output", "", "directory to write the generated .cbz files into (default: a sibling folder next to -input)")
	flag.StringVar(&output, "o", "", "shorthand for -output")
	flag.BoolVar(&batch, "batch", false, "treat -input as a library folder: process every immediate subfolder as its own manga")
	flag.BoolVar(&quiet, "quiet", false, "only print warnings and errors, not routine progress")
	flag.BoolVar(&quiet, "q", false, "shorthand for -quiet")
	flag.BoolVar(&dryRun, "dry-run", false, "show what would be written without writing any .cbz files")
	flag.BoolVar(&dryRun, "n", false, "shorthand for -dry-run")
	flag.BoolVar(&showVersion, "version", false, "print the version and exit")
	flag.Usage = printUsage
	flag.Parse()

	if showVersion {
		fmt.Println("mangabind", version)
		return
	}

	if input == "" {
		printUsage()
		os.Exit(2)
	}

	if output == "" {
		output = defaultOutputDir(input)
		if !quiet {
			fmt.Printf("mangabind: no -output given, writing to %s\n", output)
		}
	}

	var err error
	if batch {
		err = runBatch(input, output, quiet, dryRun)
	} else {
		_, err = processManga(input, output, quiet, dryRun)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mangabind:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `mangabind reorganizes a chapter-by-chapter manga download into one .cbz per
volume, ready for Kindle Comic Converter or any comic/manga reader.

Usage:
  mangabind -input <dir> [-output <dir>]
  mangabind -input <dir> -batch [-output <dir>]

Examples:
  mangabind -input "D:\Manga\Chainsaw Man"
  mangabind -i "D:\Manga\Chainsaw Man" -o "D:\Volumes"
  mangabind -input "D:\Manga" -batch

Flags:
`)
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, "\nMore info: https://github.com/gustavommcv/mangabind\n")
}

// defaultOutputDir picks a sibling directory next to input, named after it,
// used when -output isn't given. A sibling rather than a subdirectory of
// input matters: writing inside the scanned input tree would make the next
// run see the output folder itself as an unrecognized chapter.
func defaultOutputDir(input string) string {
	return filepath.Join(filepath.Dir(input), filepath.Base(input)+" (mangabind)")
}

// runBatch treats libraryInput as a folder of manga folders, processing each
// immediate subdirectory independently: one manga failing (or producing no
// volumes) is reported but never stops the rest of the batch, the same way
// one bad chapter never stops the rest of a single manga's volumes.
func runBatch(libraryInput, output string, quiet, dryRun bool) error {
	entries, err := os.ReadDir(libraryInput)
	if err != nil {
		return fmt.Errorf("scanning library %s: %w", libraryInput, err)
	}

	var mangaCount, errCount, totalVolumes, totalPages int

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mangaCount++
		mangaPath := filepath.Join(libraryInput, e.Name())
		if !quiet {
			fmt.Printf("== %s ==\n", e.Name())
		}

		summary, err := processManga(mangaPath, output, quiet, dryRun)
		if err != nil {
			errCount++
			fmt.Fprintf(os.Stderr, "mangabind: %s: %v\n", e.Name(), err)
			continue
		}
		totalVolumes += summary.volumes
		totalPages += summary.pages
	}

	fmt.Printf("mangabind: processed %d manga, %d volume(s), %d page(s) total\n", mangaCount, totalVolumes, totalPages)
	if errCount > 0 {
		return fmt.Errorf("%d of %d manga had errors, see above", errCount, mangaCount)
	}
	return nil
}

// mangaSummary is what processManga reports back so runBatch can print a
// final tally without processManga needing to know it's running in a batch.
type mangaSummary struct {
	volumes int
	pages   int
}

func processManga(input, output string, quiet, dryRun bool) (mangaSummary, error) {
	var summary mangaSummary

	entries, skipped, err := scanner.Scan(input)
	if err != nil {
		return summary, fmt.Errorf("scanning %s: %w", input, err)
	}
	if len(entries) == 0 {
		if !quiet {
			fmt.Printf("mangabind: no chapter folders or .cbz files found in %s\n", input)
		}
		return summary, nil
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
			return summary, fmt.Errorf("listing pages in %s: %w", e.Path, err)
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
		if dryRun {
			fmt.Printf("would write %s (%d pages)\n", outPath, len(vol.Pages))
		} else {
			if err := cbz.Write(outPath, vol.Pages); err != nil {
				return summary, fmt.Errorf("writing volume %v: %w", vol.Number, err)
			}
			if !quiet {
				fmt.Printf("wrote %s (%d pages)\n", outPath, len(vol.Pages))
			}
		}
		summary.volumes++
		summary.pages += len(vol.Pages)
	}
	if len(result.Volumes) == 0 {
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

	return summary, nil
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
