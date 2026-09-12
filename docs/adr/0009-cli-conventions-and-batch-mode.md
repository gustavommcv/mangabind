# 9. Adopt relevant CLI conventions from clig.dev, add batch mode

Date: 2026-09-12

## Status

Accepted

## Context

Mangabind's CLI grew flag-by-flag as each feature landed, without ever being checked against
established CLI design conventions. Separately, a real usage pattern hadn't been addressed:
someone with a whole manga library (a folder of manga folders, each with its own chapters) had to
run Mangabind once per manga by hand.

Audited the current CLI against [clig.dev](https://clig.dev)'s guidelines. Most of it already held
up: flags over positional arguments, `stdout` for primary output vs `stderr` for warnings,
sensible exit codes (0/1/2), a sensible default for `--output`, and - as of ADR 0007 - never
exiting having done nothing without saying so. Concrete gaps found:

- No `--version` flag.
- `-h`/`--help` fell back to Go's bare auto-generated flag listing - no description, no examples.
- No short aliases for the two flags used on every invocation (`--input`/`--output`).
- No way to preview what would happen without writing files (`--dry-run`).
- No way to quiet routine progress output for scripted/repeated use (`--quiet`).
- No way to process more than one manga per invocation (the batch mode gap that motivated this
  audit in the first place).

Guidelines considered and deliberately **not** adopted, because they don't fit a tool this size or
its actual usage pattern:

- `--json`/`--plain` machine-readable output - no evidence anyone scripts around Mangabind's
  output; adding a second output format to keep in sync for a hypothetical consumer is exactly the
  kind of premature scope this project has consistently declined (ADR 0005, ADR 0007, ADR 0008).
- A config file / environment variables - two flags don't need a config layer.
- Man pages, shell completions, analytics - maintenance surface with no demonstrated need yet.
- Colored output - genuinely just cosmetic for this tool; can be added later without touching any
  existing behavior, so there's no cost to deferring it.

## Decision

- Added `--version` (prints a version string injected at release build time via
  `-ldflags -X main.version=...` in `.goreleaser.yml`; plain `go build`/`go run` produce `"dev"`).
- Replaced the default flag-listing help with a real usage message: description, example
  invocations (including the new batch form), then the flag list, then a link to the repo.
- Added `-i`/`-o` as short aliases for `--input`/`--output` (Go's `flag` package has no built-in
  alias mechanism, so this is two `flag.StringVar` calls at the same variable - a small, standard
  workaround).
- Added `--dry-run`/`-n`: runs the full pipeline and prints what each volume would be, without
  calling `cbz.Write`.
- Added `--quiet`/`-q`: suppresses the routine "wrote ..."/progress lines, keeping warnings and
  errors visible (those stay actionable regardless of quiet mode).
- Added `--batch`: treats `--input` as a library folder and processes every immediate subfolder as
  its own manga. One manga failing (or producing zero volumes) is reported and does not stop the
  rest of the batch - the same graceful-degradation stance already applied within a single manga's
  chapters (gaps/conflicts/unparsed don't block the rest of that manga's volumes either). A final
  summary line reports how many manga/volumes/pages were processed; if any manga hit a hard error,
  the process exits non-zero after finishing the rest.

`--batch` is an explicit flag rather than auto-detected, on purpose: a folder of chapter folders
and a folder of manga folders look structurally identical from the outside (both are just "a
folder containing folders") - only the user knows which one they're pointing at, so guessing would
violate the project's standing rule against guessing when there's genuine ambiguity (ADR 0003,
ADR 0005).

## Consequences

The CLI now matches the conventions users of other well-designed CLI tools already expect
(`--version`, real `--help`, `--dry-run`, `--quiet`) without taking on machine-readable output
formats, config files, or other infrastructure this project has no evidence it needs yet. Batch
mode reuses the exact same per-manga pipeline as before (`processManga`, extracted from the old
`run`), so every existing guarantee - gap/conflict detection, chapter-directory-per-KCC-TOC,
never-silent output - applies identically whether Mangabind is pointed at one manga or a whole
library.
