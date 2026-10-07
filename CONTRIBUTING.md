# Contributing to Mangabind

## Setup

Requires Go 1.26+ (the minimum is the `go` line of `go.mod`, and CI follows it). No other dependencies to install.

```bash
go build ./...
go test -race ./...   # -race needs a C compiler; plain `go test ./...` works without it
go vet ./...
gofmt -l .            # should print nothing; run `gofmt -w .` to fix
go mod tidy -diff     # should print nothing: go.mod and go.sum are as `go mod tidy` leaves them
```

CI runs the same checks on Windows, macOS, and Linux for every PR (the race detector everywhere but
Windows), plus [golangci-lint](https://golangci-lint.run/) (config in `.golangci.yml`), `govulncheck`
(also once a week, since an advisory can land without a change here) and a `goreleaser --snapshot`
build to catch a broken release config early. Run `golangci-lint run ./...` locally if you have it
installed.

Everything the workflows use is pinned: each action to a commit (the tag it stands for is in the
comment beside it) and each tool to a version. Dependabot proposes the updates, one pull request a
week for the actions.

## Releasing

A release is a `vX.Y.Z` tag on a commit that is on `main`. The Release workflow runs the same checks
as CI and `govulncheck` on that commit, refuses a tag whose commit is not on `main`, and only then
builds and publishes the archives with goreleaser.

## Testing against real manga

`testdata/` holds small synthetic fixtures (empty or placeholder files, real folder/file *names*)
that are committed and used by `go test`. If you have real manga folders or CBZ files, drop them
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

Mangabind resolves missing volume numbers from a local metadata file only (`internal/metadata`) -
it never talks to the network. PRs adding an external metadata API (or any other) client, a `MetadataProvider`
abstraction, or manga/work identification will be redirected; see
[docs/adr/0010-local-metadata-file.md](docs/adr/0010-local-metadata-file.md) for why. A tool that
*generates* a metadata file from an external source is welcome as its own separate project.

## Language

Code, comments, commit messages, and docs are all in English, to keep the project approachable
for contributors regardless of what manga/language they personally read.
