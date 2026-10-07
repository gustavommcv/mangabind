# 15. The command line refuses what it does not understand, and keeps help and notices off the result's stream

Date: 2026-10-07

## Status

Accepted. Applies and completes [ADR 0009](0009-cli-conventions-and-batch-mode.md); that ADR is marked as partly superseded.

## Context

An audit of mangabind against clig.dev and its own ADRs ([`docs/audit-2026-10.md`](../audit-2026-10.md), findings 1, 8, 11, 12 and 13) ran the released binary and found five things about the command line, all reproduced:

- The `flag` package stops reading flags at the first argument that is not one, and mangabind never looked at what was left. `mangabind -input Manga extra -dry-run -output out2` ignored `-dry-run` and `-output`, exited 0, and wrote two real `.cbz` files into the default output folder. The same happens when a shell hands over a path with an unquoted space, or a glob that matches more than one folder. ADR 0009 says `-dry-run` shows what would be written without writing; here the flag was dropped without a word and the run it was meant to prevent went ahead.
- `-json=true` was not recognised as asking for machine output, so a command line that could not be parsed got a human message and no report, against ADR 0011 ("when machine mode was requested, a non-zero exit still produces a parseable report whenever the process can do so").
- `-h` printed the help on stderr, so `mangabind -h | less` showed nothing, and an unknown flag printed Go's one-line message followed by the whole 44-line help, which buries the line the person needs.
- Three notices that are not the result (`no -output given, writing to …`, `using metadata file …`, `no chapter folders or .cbz files found …`) went to stdout, next to the `wrote …` lines that are, so `mangabind -input x | tee list.txt` captured them. ADR 0009's own rule is `stdout` for the result and `stderr` for the rest.
- The "no chapter folders" line was hidden by `-quiet`, although it is the one thing that says nothing was done.

## Decision

- **A leftover argument is an error.** After the flags are read, any word that is left is refused with exit code 2 and a message that names it, says that a path with spaces needs quotes, and, when a later word begins with `-`, says that the flags after it were not read. Nothing is written. mangabind takes no positional arguments, so there is no reading in which the word is meant.
- **Machine output is recognised wherever it is asked for.** Before the command line is parsed, a lenient scan looks for `-json`, `--json`, `-json=true` (any value `strconv.ParseBool` reads as true) or `--protocol-version`, up to a bare `--`. If one is found and the command line cannot be used, stdout holds one report with an `invalid_arguments` issue, as before, and stderr stays empty: with `--progress-json` it is reserved for progress. `-json=false` is not a request for machine output.
- **Help that is asked for is the output.** `-h`, `-help` and `--help` print the help to stdout and exit 0.
- **A mistake gets a short message on stderr and exit code 2**: what was wrong, for an unknown flag the nearest flag when it is within one edit (two for a name of five letters or more) as `(did you mean -input?)`, and one line pointing to `mangabind -h`. The whole help is not printed. Running with no arguments at all is the one case that still prints it (to stderr, exit 2), since a person who has just typed the name wants the examples.
- **Notices are not the result.** `no -output given, writing to …` and `using metadata file …` go to stderr. Stdout is the lines that say what was or would be written, the batch headings and the batch tally.
- **An empty folder is a warning that `-quiet` does not hide.** `warning: no chapter folders or .cbz files found in …` goes to stderr whatever the verbosity. The exit code stays 0: nothing to do is not a failure, the JSON report already says it (`no_chapters_found`, a warning), and making it an error would change what a manga with no chapters means in a `-batch` run. ADR 0007's rule, never to exit having done nothing without saying so, is what this keeps.
- **The examples in the help start with a POSIX path**, and the Windows one comes last.

## Consequences

A script that passed a stray word and was silently obeyed now fails with exit 2, which is the point. Mangabound passes only flags, so it is not affected. A script that read a notice from stdout reads it from stderr now; no consumer of that is known, and the machine report never carried them. The protocol is unchanged: `invalid_arguments` already existed, and the document says in which cases it is used. Because `parseFlags` prints nothing and `runCLI` decides what is said and where, each of these rules is a test on `runCLI` (flags before and after a stray argument, the spellings of `-json`, help on stdout, the length of the messages, the suggestion, the streams), which is the layer the audit found least covered.

Not adopted: a positional input (`mangabind <folder>`). ADR 0009 prefers flags, and it would be a new way to call the tool, not a repair of this one.
