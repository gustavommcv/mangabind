# Contributing to Mangabind

## Setup

Requires Go 1.23+. No other dependencies to install.

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .   # should print nothing; run `gofmt -w .` to fix
```

CI runs the same checks on Windows, macOS, and Linux for every PR, plus
[golangci-lint](https://golangci-lint.run/) (config in `.golangci.yml`) and a `goreleaser --snapshot`
build to catch a broken release config early. Run `golangci-lint run ./...` locally if you have it
installed.

## Testing against real manga

`testdata/` holds small synthetic fixtures (empty or placeholder files, real folder/file *names*)
that are committed and used by `go test`. If you have real manga downloaded via HakuNeko, drop it
under `examples/` at the repo root - that directory is git-ignored (it's copyrighted content) and
is only for manual local verification, never for automated tests.

## Adding support for a new chapter-naming convention

This is the easiest way to contribute and the most valuable one: naming conventions vary a lot
across scan groups and sites (see [docs/adr/0003-pluggable-chapter-parsing.md](docs/adr/0003-pluggable-chapter-parsing.md)
for why).

1. Add a new file under `internal/parser/` implementing the `ChapterNameParser` interface.
2. Register it in the parser registry, ordered from most to least specific.
3. Add table-driven tests with real (or realistic) folder-name examples - no need for actual
   images, just the strings.
4. If a folder name genuinely can't be parsed with confidence, don't guess: return `false` so it
   surfaces in the "unprocessed" report instead of being silently mis-grouped.

## Scope

Mangabind reorganizes files into `.cbz` archives. It does not resize, recompress, crop, or
otherwise touch image content - that's explicitly out of scope (it's KCC's job). PRs that add
image processing will be redirected elsewhere.

Mangabind also doesn't try to identify or reposition a "cover" page - see
[docs/adr/0005-drop-cover-detection.md](docs/adr/0005-drop-cover-detection.md) for why. PRs adding
cover-detection heuristics will be redirected too.

## Language

Code, comments, commit messages, and docs are all in English, to keep the project approachable
for contributors regardless of what manga/language they personally read.
