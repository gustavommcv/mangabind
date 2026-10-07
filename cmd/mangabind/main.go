// Command mangabind reorganizes a chapter-by-chapter manga download into one .cbz per volume.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gustavommcv/mangabind/internal/cbz"
	"github.com/gustavommcv/mangabind/internal/grouper"
	"github.com/gustavommcv/mangabind/internal/metadata"
	"github.com/gustavommcv/mangabind/internal/parser"
	"github.com/gustavommcv/mangabind/internal/scanner"
)

// version is set at build/release time via -ldflags "-X main.version=...";
// "dev" is what a plain `go build`/`go run` produces.
var version = "dev"

// cliConfig is the parsed result of the command line, kept separate from
// flag.FlagSet so parseFlags is easy to call directly from tests.
type cliConfig struct {
	input, output, metadataFile                       string
	batch, quiet, dryRun, json, combine, progressJSON bool
	showVersion, showProtocol                         bool
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	machineRequested := flagRequested(args, "json") || flagRequested(args, "protocol-version")
	cfg, fs, err := parseFlags(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Help that was asked for is the output, so it goes to stdout (and
			// can be piped into a pager); help that was not asked for is an
			// error, and goes to stderr below.
			printUsage(fs, stdout)
			return 0
		}
		if machineRequested {
			// The report on stdout is the whole answer; stderr stays free for
			// the progress stream a consumer may have asked for.
			report := newMachineReport("configuration", false)
			value := issue("error", "invalid_arguments", "configuration", "The command-line arguments are invalid.")
			value.Diagnostic = err.Error()
			report.addIssue(value)
			_ = writeMachineJSON(stdout, report)
		} else {
			printUsageError(stderr, fs, err)
		}
		return 2
	}

	if cfg.showProtocol {
		if err := writeMachineJSON(stdout, protocolInfo{
			ProtocolVersion: machineProtocolVersion,
			Tool:            "mangabind",
			ToolVersion:     version,
			Capabilities:    []string{"report", "progress-json"},
		}); err != nil {
			printlnTo(stderr, "mangabind: writing protocol information:", err)
			return 1
		}
		return 0
	}

	if cfg.showVersion {
		printlnTo(stdout, "mangabind", version)
		return 0
	}
	if cfg.progressJSON && !cfg.json {
		printlnTo(stderr, "mangabind: -progress-json requires -json")
		return 2
	}

	if cfg.input == "" {
		if cfg.json {
			report := newMachineReport(modeFor(cfg.dryRun), cfg.batch)
			value := issue("error", "missing_input", "configuration", "Choose a manga, CBZ, or library folder to process.")
			value.Recoverable = true
			report.addIssue(value)
			_ = writeMachineJSON(stdout, report)
		} else if len(args) == 0 {
			// Run with nothing at all: show how to use it.
			printUsage(fs, stderr)
		} else {
			printlnTo(stderr, "mangabind: no -input given")
			printlnTo(stderr, "Usage: mangabind -input <dir> [-output <dir>]")
			printlnTo(stderr, hintForHelp)
		}
		return 2
	}
	if cfg.batch && cfg.metadataFile != "" {
		message := "-metadata-file can't be combined with -batch - place a mangabind.json file inside each manga folder instead"
		if cfg.json {
			report := newMachineReport(modeFor(cfg.dryRun), true)
			value := issue("error", "metadata_file_with_batch", "configuration", "A library batch cannot use one shared metadata file.")
			value.Recoverable = true
			value.Diagnostic = message
			report.addIssue(value)
			_ = writeMachineJSON(stdout, report)
		} else {
			printlnTo(stderr, "mangabind:", message)
		}
		return 2
	}

	output := cfg.output
	if output == "" {
		output = defaultOutputDir(cfg.input)
		if !cfg.quiet && !cfg.json {
			printfTo(stderr, "mangabind: no -output given, writing to %s\n", output)
		}
	}

	if cfg.json {
		var progress progressSink
		if cfg.progressJSON {
			progress = newProgressSink(stderr)
		}
		report, err := runMachine(cfg, output, progress)
		if writeErr := writeMachineJSON(stdout, report); writeErr != nil {
			printlnTo(stderr, "mangabind: writing JSON report:", writeErr)
			return 1
		}
		if err != nil {
			return 1
		}
		return 0
	}

	if cfg.batch {
		err = runBatchWithOutput(cfg.input, output, cfg.quiet, cfg.dryRun, cfg.combine, stdout, stderr)
	} else {
		_, err = processMangaWithOutput(cfg.input, output, cfg.metadataFile, cfg.quiet, cfg.dryRun, cfg.combine, stdout, stderr)
	}
	if err != nil {
		printlnTo(stderr, "mangabind:", err)
		return 1
	}
	return 0
}

// flagRequested reports whether the boolean flag name is switched on anywhere
// in args, in any of the spellings the flag package accepts (-json, --json,
// -json=true). It is a lenient scan, not a parse: it is what lets a run that
// asked for machine output get a machine-readable answer even when the rest of
// the command line is the very thing that could not be parsed.
func flagRequested(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		trimmed := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		if trimmed == arg {
			continue
		}
		flagName, value, hasValue := strings.Cut(trimmed, "=")
		if flagName != name {
			continue
		}
		if !hasValue {
			return true
		}
		if on, err := strconv.ParseBool(value); err == nil && on {
			return true
		}
	}
	return false
}

func modeFor(dryRun bool) string {
	if dryRun {
		return "plan"
	}
	return "execute"
}

// Human-readable output is best-effort. Machine-readable output uses
// writeMachineJSON, which reports write failures to callers.
func printTo(writer io.Writer, values ...any) {
	_, _ = fmt.Fprint(writer, values...)
}

func printfTo(writer io.Writer, format string, values ...any) {
	_, _ = fmt.Fprintf(writer, format, values...)
}

func printlnTo(writer io.Writer, values ...any) {
	_, _ = fmt.Fprintln(writer, values...)
}

// hintForHelp ends every message about a command line that could not be used.
const hintForHelp = "Try 'mangabind -h' for the examples and the list of flags."

// parseFlags defines and parses mangabind's flags, including the -i/-o/-q/-n
// short aliases. It uses its own FlagSet rather than the package-level
// flag.CommandLine so that it can be called directly (and repeatedly) from
// tests: flag.ContinueOnError makes Parse return errors instead of calling
// os.Exit, and a fresh FlagSet avoids the "flag redefined" panic a second call
// to flag.StringVar on flag.CommandLine would cause.
//
// It prints nothing. The flag package would print its own message and then the
// whole usage for any mistake, and would send it to stderr even when the help
// was asked for; runCLI says what there is to say, and where.
func parseFlags(args []string) (cliConfig, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("mangabind", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var cfg cliConfig
	fs.StringVar(&cfg.input, "input", "", "directory to reorganize: a manga's chapters, or (with -batch) a library of manga folders")
	fs.StringVar(&cfg.input, "i", "", "shorthand for -input")
	fs.StringVar(&cfg.output, "output", "", "directory to write the generated .cbz files into (default: a sibling folder next to -input)")
	fs.StringVar(&cfg.output, "o", "", "shorthand for -output")
	fs.BoolVar(&cfg.batch, "batch", false, "treat -input as a library folder: process every immediate subfolder as its own manga")
	fs.StringVar(&cfg.metadataFile, "metadata-file", "", "local chapter-to-volume mapping to fill in volumes missing from chapter names (not combinable with -batch; see mangabind.json convention)")
	fs.BoolVar(&cfg.quiet, "quiet", false, "only print warnings and errors, not routine progress")
	fs.BoolVar(&cfg.quiet, "q", false, "shorthand for -quiet")
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "show what would be written without writing any .cbz files")
	fs.BoolVar(&cfg.dryRun, "n", false, "shorthand for -dry-run")
	fs.BoolVar(&cfg.json, "json", false, "emit one versioned JSON report on stdout instead of human-readable output")
	fs.BoolVar(&cfg.progressJSON, "progress-json", false, "with -json, emit newline-delimited progress events on stderr while keeping the final report on stdout")
	fs.BoolVar(&cfg.combine, "combine", false, "write the whole manga as one .cbz instead of one per volume, still reporting each volume's own chapters")
	fs.BoolVar(&cfg.showProtocol, "protocol-version", false, "print machine-protocol compatibility information as JSON and exit")
	fs.BoolVar(&cfg.showVersion, "version", false, "print the version and exit")
	fs.Usage = func() {}

	if err := fs.Parse(args); err != nil {
		return cfg, fs, err
	}
	// The flag package stops reading flags at the first argument that is not
	// one, and everything after it, flags included, is left over. Taking that
	// for nothing would let `-dry-run` be dropped without a word, and a real
	// run follow.
	if fs.NArg() > 0 {
		return cfg, fs, &unexpectedArgumentError{args: fs.Args()}
	}
	return cfg, fs, nil
}

// unexpectedArgumentError is a word on the command line that is not a flag and
// not the value of one. args is it and everything that followed.
type unexpectedArgumentError struct{ args []string }

func (e *unexpectedArgumentError) Error() string {
	return fmt.Sprintf("unexpected argument %q", e.args[0])
}

// printUsage writes the help: what the tool does, how to call it, examples, and
// every flag.
func printUsage(fs *flag.FlagSet, w io.Writer) {
	printTo(w, `mangabind reorganizes a chapter-by-chapter manga download into one .cbz per
volume, ready for Kindle Comic Converter or any comic/manga reader.

Usage:
  mangabind -input <dir> [-output <dir>]
  mangabind -input <dir> -batch [-output <dir>]

Examples:
  mangabind -input ~/Manga/"Chainsaw Man"
  mangabind -i ~/Manga/"Chainsaw Man" -o ~/Volumes
  mangabind -input ~/Manga -batch
  mangabind -input ~/Manga/"Some Manga" -metadata-file volumes.json
  mangabind -input "D:\Manga\Chainsaw Man"      (Windows)

Flags:
`)
	fs.SetOutput(w)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
	printTo(w, "\nMore info: https://github.com/gustavommcv/mangabind\n")
}

// printUsageError says, in a few lines and on w, what was wrong with the
// command line and where to look: not the whole help, which buries the line the
// person needs.
func printUsageError(w io.Writer, fs *flag.FlagSet, err error) {
	var unexpected *unexpectedArgumentError
	if errors.As(err, &unexpected) {
		printlnTo(w, "mangabind:", err)
		printlnTo(w, "  If it is part of a path with spaces, put the whole path in quotes.")
		for _, arg := range unexpected.args[1:] {
			if strings.HasPrefix(arg, "-") {
				printlnTo(w, "  The flags after it were not read.")
				break
			}
		}
	} else if name, ok := strings.CutPrefix(err.Error(), "flag provided but not defined: -"); ok {
		message := "mangabind: unknown flag -" + name
		if suggestion := closestFlag(fs, name); suggestion != "" {
			message += " (did you mean -" + suggestion + "?)"
		}
		printlnTo(w, message)
	} else {
		printlnTo(w, "mangabind:", err)
	}
	printlnTo(w, hintForHelp)
}

// closestFlag is the flag whose name is nearest to the unknown one, when it is
// near enough to be a slip of the fingers and not a different word: one edit for
// a short name, two for a longer. The one-letter aliases are left out, since
// everything is within one edit of them.
func closestFlag(fs *flag.FlagSet, unknown string) string {
	limit := 1
	if len(unknown) >= 5 {
		limit = 2
	}
	best, bestDistance := "", limit+1
	fs.VisitAll(func(f *flag.Flag) {
		if len(f.Name) < 2 {
			return
		}
		if d := editDistance(unknown, f.Name); d < bestDistance {
			best, bestDistance = f.Name, d
		}
	})
	return best
}

// editDistance is the number of single-character insertions, deletions and
// substitutions that turn a into b.
func editDistance(a, b string) int {
	previous := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current := make([]int, len(b)+1)
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(b)]
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
	return runBatchWithOutput(libraryInput, output, quiet, dryRun, false, os.Stdout, os.Stderr)
}

func runBatchWithOutput(libraryInput, output string, quiet, dryRun, combine bool, stdout, stderr io.Writer) error {
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
			printfTo(stdout, "== %s ==\n", e.Name())
		}

		// No explicit override in batch mode - each manga only picks up its
		// own mangabind.json convention file, if it has one.
		summary, err := processMangaWithOutput(mangaPath, output, "", quiet, dryRun, combine, stdout, stderr)
		if err != nil {
			errCount++
			printfTo(stderr, "mangabind: %s: %v\n", e.Name(), err)
			continue
		}
		totalVolumes += summary.volumes
		totalPages += summary.pages
	}

	printfTo(stdout, "mangabind: processed %d manga, %d volume(s), %d page(s) total\n", mangaCount, totalVolumes, totalPages)
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

// metadataConventionFile is the name Mangabind looks for inside a manga's
// own input folder when -metadata-file isn't given explicitly - see
// docs/adr/0010-local-metadata-file.md.
const metadataConventionFile = "mangabind.json"

// resolveMetadataFile picks which metadata file (if any) applies to input:
// an explicit override always wins; otherwise Mangabind looks for its
// naming convention right inside the manga's own folder. Returning "" means
// no metadata file applies - not an error, just nothing to load.
func resolveMetadataFile(input, explicit string) string {
	if explicit != "" {
		return explicit
	}
	candidate := filepath.Join(input, metadataConventionFile)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func processManga(input, output, metadataFile string, quiet, dryRun bool) (mangaSummary, error) {
	return processMangaWithOutput(input, output, metadataFile, quiet, dryRun, false, os.Stdout, os.Stderr)
}

func processMangaWithOutput(input, output, metadataFile string, quiet, dryRun, combine bool, stdout, stderr io.Writer) (mangaSummary, error) {
	summary, _, err := processMangaDetailed(input, output, metadataFile, quiet, dryRun, combine, true, stdout, stderr, nil)
	return summary, err
}

func processMangaDetailed(input, output, metadataFile string, quiet, dryRun, combine, human bool, stdout, stderr io.Writer, progress progressSink) (mangaSummary, mangaReport, error) {
	var summary mangaSummary
	mangaName := filepath.Base(filepath.Clean(input))
	report := newMangaReport(mangaName, absolutePath(input))
	if progress != nil {
		progress(machineProgress{Stage: "inspect", State: "started", Manga: mangaName})
	}

	var metaMap *metadata.Map
	if mf := resolveMetadataFile(input, metadataFile); mf != "" {
		report.MetadataFile = absolutePath(mf)
		m, err := metadata.Load(mf)
		if err != nil {
			wrapped := fmt.Errorf("loading metadata file %s: %w", mf, err)
			value := issue("error", "metadata_load_failed", "metadata", "Couldn't load the chapter-to-volume metadata file.")
			value.Manga = mangaName
			value.Path = absolutePath(mf)
			value.Recoverable = true
			value.Diagnostic = wrapped.Error()
			report.addIssue(value)
			finalizeMangaReport(&report, summary)
			return summary, report, wrapped
		}
		metaMap = m
		if human && !quiet {
			printfTo(stderr, "mangabind: using metadata file %s\n", mf)
		}
	}

	entries, skipped, links, err := scanner.Scan(input)
	if err != nil {
		wrapped := fmt.Errorf("scanning %s: %w", input, err)
		value := issue("error", "input_scan_failed", "inspect", "Couldn't inspect the manga folder.")
		value.Manga = mangaName
		value.Path = absolutePath(input)
		value.Recoverable = true
		value.Diagnostic = wrapped.Error()
		report.addIssue(value)
		finalizeMangaReport(&report, summary)
		return summary, report, wrapped
	}
	if len(entries) == 0 {
		if human {
			// A warning, so -quiet does not hide that nothing was done.
			printfTo(stderr, "warning: no chapter folders or .cbz files found in %s\n", input)
		}
		value := issue("warning", "no_chapters_found", "inspect", "No chapter folders or CBZ files were found.")
		value.Manga = mangaName
		value.Path = absolutePath(input)
		report.addIssue(value)
		finalizeMangaReport(&report, summary)
		return summary, report, nil
	}
	reportLinks(&report, mangaName, links, nil, human, stderr)
	for _, name := range skipped {
		message := unsupportedFileMessage(name)
		if human {
			printlnTo(stderr, "warning:", message)
		}
		value := issue("warning", "unsupported_input_file", "inspect", message)
		value.Manga = mangaName
		value.Path = absolutePath(filepath.Join(input, name))
		report.addIssue(value)
	}

	reg := parser.DefaultRegistry()
	var chapters []grouper.Chapter
	var unparsed []string

	for _, e := range entries {
		parsed, parserName, ok := reg.Parse(e.Name)
		unit := unitReport{
			Name:               e.Name,
			Path:               absolutePath(e.Path),
			Kind:               "folder",
			Disposition:        "candidate",
			MetadataAssignment: metadataAssignmentReport{Status: "not_requested"},
		}
		if e.IsArchive {
			unit.Kind = "cbz"
		}
		if !ok {
			unit.Parser = parserReport{Matched: false}
			unit.MetadataAssignment.Status = "not_applicable"
			unit.Disposition = "unparsed"
			report.Units = append(report.Units, unit)
			unparsed = append(unparsed, e.Name)
			continue
		}

		chapterNumber := parsed.Chapter
		unit.Parser = parserReport{
			Matched:  true,
			Name:     parserName,
			Volume:   copyFloat(parsed.Volume),
			Chapter:  &chapterNumber,
			Special:  parsed.Special,
			Title:    parsed.Title,
			Group:    parsed.Group,
			Language: parsed.Lang,
		}
		if metaMap != nil {
			unit.MetadataAssignment.Status = "not_found"
		}

		if metaMap != nil {
			if metaVol, found := metaMap.Lookup(parsed.Chapter, parsed.Special); found {
				metadataVolume := metaVol
				unit.MetadataAssignment.MetadataVolume = &metadataVolume
				switch {
				case parsed.Volume == nil:
					v := metaVol
					parsed.Volume = &v
					unit.MetadataAssignment.Status = "applied"
				case *parsed.Volume == metaVol:
					unit.MetadataAssignment.Status = "confirmed"
				case *parsed.Volume != metaVol:
					unit.MetadataAssignment.Status = "ignored_conflict"
					if human {
						printfTo(stderr,
							"warning: chapter %v%s: name says volume %v, metadata file says volume %v - keeping the name\n",
							parsed.Chapter, parsed.Special, *parsed.Volume, metaVol)
					}
					message := fmt.Sprintf("Chapter %v%s keeps volume %v from its name instead of metadata volume %v.", parsed.Chapter, parsed.Special, *parsed.Volume, metaVol)
					value := issue("warning", "metadata_volume_conflict", "metadata", message)
					value.Manga = mangaName
					value.Volume = copyFloat(parsed.Volume)
					value.Chapter = copyFloat(&parsed.Chapter)
					value.Special = parsed.Special
					value.Path = absolutePath(e.Path)
					report.addIssue(value)
				}
			}
		}
		unit.EffectiveVolume = copyFloat(parsed.Volume)

		var pages []string
		if e.IsArchive {
			pages, err = scanner.PagesInArchive(e.Path)
		} else {
			var pageLinks []scanner.Link
			pages, pageLinks, err = scanner.Pages(input, e.Path)
			reportLinks(&report, mangaName, pageLinks, &parsed, human, stderr)
		}
		if err != nil {
			wrapped := fmt.Errorf("listing pages in %s: %w", e.Path, err)
			unit.Disposition = "failed"
			report.Units = append(report.Units, unit)
			value := issue("error", "page_listing_failed", "inspect", fmt.Sprintf("Couldn't list pages for %q.", e.Name))
			value.Manga = mangaName
			value.Path = absolutePath(e.Path)
			value.Recoverable = true
			value.Diagnostic = wrapped.Error()
			report.addIssue(value)
			finalizeMangaReport(&report, summary)
			return summary, report, wrapped
		}
		unit.PageCount = len(pages)
		if len(pages) == 0 {
			unit.Disposition = "empty"
			report.Units = append(report.Units, unit)
			unparsed = append(unparsed, e.Name)
			continue
		}

		report.Units = append(report.Units, unit)
		chapters = append(chapters, grouper.Chapter{Parsed: parsed, Path: e.Path, Pages: pages, IsArchive: e.IsArchive})
	}

	result := grouper.Group(chapters, unparsed)
	markUnitDispositions(report.Units, result)
	if progress != nil {
		progress(machineProgress{Stage: "inspect", State: "completed", Manga: mangaName})
	}
	var totalPages int
	for _, volume := range result.Volumes {
		totalPages += len(volume.Pages)
	}
	completedPages := 0
	emitWrite := func(state string, index int, volume float64, completed int) {
		if progress != nil {
			progress(machineProgress{
				Stage: "write", State: state, Manga: mangaName,
				VolumeIndex: index + 1, VolumeCount: len(result.Volumes), VolumeNumber: &volume,
				CompletedPages: &completed, TotalPages: totalPages,
			})
		}
	}

	if combine {
		// One .cbz for the whole manga instead of one per volume - see
		// docs/adr/0012-combine-series-into-one-volume.md. Each volume still
		// gets its own report entry (chapters, page count, number), but its
		// pages physically live inside the one combined file; grouper.Group
		// above is untouched, only how the result is written differs.
		outPath := filepath.Join(output, cbz.SeriesFileName(mangaName))
		report.CombinedOutputPath = absolutePath(outPath)
		for _, vol := range result.Volumes {
			report.Volumes = append(report.Volumes, volumeReport{
				Number:     vol.Number,
				OutputPath: absolutePath(outPath),
				PageCount:  len(vol.Pages),
				Chapters:   chapterNamesForVolume(report.Units, vol.Number),
				Written:    false,
			})
			summary.volumes++
			summary.pages += len(vol.Pages)
		}
		if dryRun {
			if human {
				printfTo(stdout, "would write %s (%d pages, %d volumes)\n", outPath, summary.pages, summary.volumes)
			}
		} else {
			combined := grouper.CombineVolumes(result.Volumes)
			if len(result.Volumes) > 0 {
				emitWrite("started", 0, result.Volumes[0].Number, 0)
			}
			volumeIndex := 0
			volumeEnd := 0
			if len(result.Volumes) > 0 {
				volumeEnd = len(result.Volumes[0].Pages)
			}
			var onPage func(int)
			if progress != nil {
				onPage = func(completed int) {
					for volumeIndex+1 < len(result.Volumes) && completed > volumeEnd {
						volumeIndex++
						volumeEnd += len(result.Volumes[volumeIndex].Pages)
					}
					if len(result.Volumes) > 0 {
						emitWrite("advanced", volumeIndex, result.Volumes[volumeIndex].Number, completed)
					}
				}
			}
			if err := cbz.WriteWithProgress(outPath, combined, onPage); err != nil {
				wrapped := fmt.Errorf("writing combined series: %w", err)
				value := issue("error", "combined_write_failed", "write", "Couldn't write the combined series.")
				value.Manga = mangaName
				value.Path = absolutePath(outPath)
				value.Recoverable = true
				value.Diagnostic = wrapped.Error()
				report.addIssue(value)
				finalizeMangaReport(&report, summary)
				return summary, report, wrapped
			}
			if len(result.Volumes) > 0 {
				last := len(result.Volumes) - 1
				emitWrite("completed", last, result.Volumes[last].Number, totalPages)
			}
			if human && !quiet {
				printfTo(stdout, "wrote %s (%d pages, %d volumes)\n", outPath, summary.pages, summary.volumes)
			}
		}
	} else {
		for index, vol := range result.Volumes {
			outPath := filepath.Join(output, cbz.VolumeFileName(mangaName, vol.Number))
			volume := volumeReport{
				Number:     vol.Number,
				OutputPath: absolutePath(outPath),
				PageCount:  len(vol.Pages),
				Chapters:   chapterNamesForVolume(report.Units, vol.Number),
				Written:    false,
			}
			if dryRun {
				if human {
					printfTo(stdout, "would write %s (%d pages)\n", outPath, len(vol.Pages))
				}
			} else {
				emitWrite("started", index, vol.Number, completedPages)
				var onPage func(int)
				if progress != nil {
					onPage = func(completed int) {
						emitWrite("advanced", index, vol.Number, completedPages+completed)
					}
				}
				if err := cbz.WriteWithProgress(outPath, vol.Pages, onPage); err != nil {
					wrapped := fmt.Errorf("writing volume %v: %w", vol.Number, err)
					report.Volumes = append(report.Volumes, volume)
					value := issue("error", "volume_write_failed", "write", fmt.Sprintf("Couldn't write volume %v.", vol.Number))
					value.Manga = mangaName
					value.Volume = copyFloat(&vol.Number)
					value.Path = absolutePath(outPath)
					value.Recoverable = true
					value.Diagnostic = wrapped.Error()
					report.addIssue(value)
					finalizeMangaReport(&report, summary)
					return summary, report, wrapped
				}
				completedPages += len(vol.Pages)
				emitWrite("completed", index, vol.Number, completedPages)
				volume.Written = true
				if human && !quiet {
					printfTo(stdout, "wrote %s (%d pages)\n", outPath, len(vol.Pages))
				}
			}
			report.Volumes = append(report.Volumes, volume)
			summary.volumes++
			summary.pages += len(vol.Pages)
		}
	}
	if len(result.Volumes) == 0 {
		if human {
			printlnTo(stdout, "mangabind: no volumes produced - see warnings below")
		}
		value := issue("warning", "no_volumes_produced", "group", "No volumes could be produced from the inspected chapters.")
		value.Manga = mangaName
		report.addIssue(value)
	}

	for _, line := range summarizeNames("could not parse chapter name, skipped", result.Unparsed, "") {
		if human {
			printlnTo(stderr, "warning:", line)
		}
	}
	for _, name := range result.Unparsed {
		value := issue("warning", "unparsed_chapter", "parse", fmt.Sprintf("Couldn't parse chapter name %q; it was skipped.", name))
		value.Manga = mangaName
		value.Path = absolutePath(filepath.Join(input, name))
		if unit := findUnit(report.Units, name); unit != nil && unit.Parser.Matched {
			value.Code = "empty_chapter"
			value.Stage = "inspect"
			value.Message = fmt.Sprintf("Chapter %q contains no pages; it was skipped.", name)
			value.Path = unit.Path
		}
		report.addIssue(value)
	}
	unassignedNames := make([]string, len(result.Unassigned))
	for i, c := range result.Unassigned {
		unassignedNames[i] = filepath.Base(c.Path)
	}
	for _, line := range summarizeNames("chapter has no volume number, skipped", unassignedNames, "see -metadata-file") {
		if human {
			printlnTo(stderr, "warning:", line)
		}
	}
	for _, c := range result.Unassigned {
		chapter := c.Parsed.Chapter
		value := issue("warning", "unassigned_chapter", "group", fmt.Sprintf("Chapter %v%s has no volume assignment and was skipped.", chapter, c.Parsed.Special))
		value.Manga = mangaName
		value.Chapter = &chapter
		value.Special = c.Parsed.Special
		value.Path = absolutePath(c.Path)
		report.addIssue(value)
	}
	for _, g := range result.Gaps {
		if human {
			printfTo(stderr, "warning: volume %v is missing chapter(s) between %v and %v\n", g.Volume, g.After, g.Before)
		}
		value := issue("warning", "chapter_gap", "group", fmt.Sprintf("Volume %v is missing chapters between %v and %v.", g.Volume, g.After, g.Before))
		value.Manga = mangaName
		value.Volume = copyFloat(&g.Volume)
		report.addIssue(value)
	}
	for _, c := range result.Conflicts {
		if human {
			printfTo(stderr, "warning: volume %v chapter %v%s has %d conflicting sources, all skipped:\n",
				c.Volume, c.Chapter, c.Special, len(c.Sources))
			for _, s := range c.Sources {
				printfTo(stderr, "  - %q\n", filepath.Base(s))
			}
		}
		chapter := c.Chapter
		value := issue("warning", "chapter_conflict", "group", fmt.Sprintf("Volume %v chapter %v%s has %d conflicting sources; all were skipped.", c.Volume, c.Chapter, c.Special, len(c.Sources)))
		value.Manga = mangaName
		value.Volume = copyFloat(&c.Volume)
		value.Chapter = &chapter
		value.Special = c.Special
		for _, source := range c.Sources {
			value.RelatedPaths = append(value.RelatedPaths, absolutePath(source))
		}
		report.addIssue(value)
	}

	finalizeMangaReport(&report, summary)
	return summary, report, nil
}

func absolutePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}

func copyFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func findUnit(units []unitReport, name string) *unitReport {
	for i := range units {
		if units[i].Name == name {
			return &units[i]
		}
	}
	return nil
}

func markUnitDispositions(units []unitReport, result grouper.Result) {
	for i := range units {
		if units[i].Disposition == "candidate" {
			units[i].Disposition = "included"
		}
	}
	for _, chapter := range result.Unassigned {
		if unit := findUnit(units, filepath.Base(chapter.Path)); unit != nil {
			unit.Disposition = "unassigned"
		}
	}
	for _, conflict := range result.Conflicts {
		for _, source := range conflict.Sources {
			if unit := findUnit(units, filepath.Base(source)); unit != nil {
				unit.Disposition = "conflict"
			}
		}
	}
}

func chapterNamesForVolume(units []unitReport, volume float64) []string {
	var chapters []unitReport
	for _, unit := range units {
		if unit.Disposition == "included" && unit.EffectiveVolume != nil && *unit.EffectiveVolume == volume {
			chapters = append(chapters, unit)
		}
	}
	sort.Slice(chapters, func(i, j int) bool {
		left, right := chapters[i].Parser, chapters[j].Parser
		if left.Chapter != nil && right.Chapter != nil && *left.Chapter != *right.Chapter {
			return *left.Chapter < *right.Chapter
		}
		return left.Special < right.Special
	})
	names := make([]string, len(chapters))
	for i, chapter := range chapters {
		names[i] = chapter.Name
	}
	return names
}

// maxIndividualWarnings caps how many names of the same warning category get
// printed one by one before summarizeNames collapses the rest into a single
// line - without it, a manga with hundreds of chapters and no volume
// information anywhere would print one warning per chapter.
const maxIndividualWarnings = 10

// summarizeNames turns a list of same-category names into warning lines:
// each one individually if there are few enough, otherwise the first
// maxIndividualWarnings plus one line summarizing the rest (with an
// optional hint pointing at the fix).
func summarizeNames(label string, names []string, hint string) []string {
	if len(names) == 0 {
		return nil
	}
	if len(names) <= maxIndividualWarnings {
		lines := make([]string, len(names))
		for i, name := range names {
			lines[i] = fmt.Sprintf("%s: %q", label, name)
		}
		return lines
	}

	lines := make([]string, 0, maxIndividualWarnings+1)
	for _, name := range names[:maxIndividualWarnings] {
		lines = append(lines, fmt.Sprintf("%s: %q", label, name))
	}
	rest := fmt.Sprintf("... and %d more", len(names)-maxIndividualWarnings)
	if hint != "" {
		rest += " (" + hint + ")"
	}
	return append(lines, rest)
}

// reportLinks tells of the links the scanner did not follow: a warning for
// each, naming the link and what is wrong with it, and nothing of it goes into
// a volume (docs/adr/0014-links-stay-inside-the-input.md). chapter is the
// chapter a page link was found in, and nil for a link found beside the
// chapters; the issue names its volume when it has one, so that a consumer
// can tell which volume lacks the page.
func reportLinks(report *mangaReport, mangaName string, links []scanner.Link, chapter *parser.ParsedChapter, human bool, stderr io.Writer) {
	for _, link := range links {
		message := fmt.Sprintf("found a link %q that %s, skipped", filepath.Base(link.Path), link.Why)
		if human {
			printlnTo(stderr, "warning:", message)
		}
		value := issue("warning", "link_skipped", "inspect", message)
		value.Manga = mangaName
		value.Path = absolutePath(link.Path)
		if chapter != nil {
			value.Volume = copyFloat(chapter.Volume)
			value.Chapter = copyFloat(&chapter.Chapter)
			value.Special = chapter.Special
		}
		report.addIssue(value)
	}
}

// unsupportedFileMessage explains why a file found alongside chapter
// folders/.cbz files wasn't treated as a chapter - naming the two
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
