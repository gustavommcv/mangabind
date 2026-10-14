package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// cliConfig is the parsed result of the command line, kept separate from
// flag.FlagSet so parseFlags is easy to call directly from tests.
type cliConfig struct {
	input, output, metadataFile                       string
	batch, quiet, dryRun, json, combine, progressJSON bool
	showVersion, showProtocol                         bool
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
	printTo(w, `
Output: what was or would be written goes to stdout; warnings, notices and
progress go to stderr. With -json, stdout is one JSON report and nothing else.

Exit codes:
  0    done (a folder with nothing to bind is done, with a warning)
  1    something could not be done - see the messages, or the report's issues
  2    the command line could not be used
  130  interrupted with Ctrl-C (or SIGTERM); the unfinished file was removed
`)
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
