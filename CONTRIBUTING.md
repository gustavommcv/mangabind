# Contributing to Mangabind

Bug reports, chapter-name examples, documentation fixes, and code contributions are welcome.
For bugs, include the version, command, expected result, and actual output. Replace personal paths
and share the smallest example that reproduces the problem.

## Setup and checks

Use the Go version required by [go.mod](go.mod), currently Go 1.26 or later. The project has no
third-party runtime dependencies.

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
go mod tidy -diff
```

Formatting and module checks should print nothing. `gofmt -w .` applies formatting. With a C
compiler installed, also run `go test -race ./...` to check for data races. To run the CLI
directly from the checkout:

```sh
go run ./cmd/mangabind --help
```

[CI](.github/workflows/ci.yml) builds and tests on Windows, macOS, and Linux, with race detection
on macOS and Linux. It also runs golangci-lint and a GoReleaser snapshot build. A separate
[vulnerability check](.github/workflows/govulncheck.yml) runs on PRs, pushes to `main`, and weekly.
Actions and tools are pinned; Dependabot proposes weekly action updates. Use the workflow files
for the current versions and commands.

## Releasing

A `vX.Y.Z` tag triggers [the release workflow](.github/workflows/release.yml). It checks that the
tagged commit is on `main` and runs CI and vulnerability checks before GoReleaser builds and
publishes the archives. Obtain maintainer approval before creating a release tag.

## Installers

`install.sh` and `install.ps1` are tested like code, with no network: `sh scripts/test-installers.sh`
and `pwsh -File scripts/test-installers.ps1` build a real mangabind, pack it as a release does, and
run the installer against it, including a damaged archive that must be refused. The
[Installers workflow](.github/workflows/installers.yml) runs both when either script changes, and
once a week runs the installers against the real latest release.

## Tests and fixtures

Add regression tests for changed behavior, including errors that callers need to handle.
Use small synthetic fixtures in `testdata/`; parser tests usually need only chapter-name strings.
Keep real manga used for local checks in the ignored `examples/` directory. Do not commit manga
pages or make automated tests depend on a private collection.

### Add a chapter-naming convention

1. Implement `ChapterNameParser` in `internal/parser/`.
2. Register the parser from most to least specific.
3. Add table-driven tests for matching names, near misses, and existing conventions it could overlap.
4. Return `false` when a name cannot be recognized confidently, so another parser can try it.

See [ADR 0003](docs/adr/0003-pluggable-chapter-parsing.md) for the parser design.

## Project scope

Mangabind groups chapters and writes CBZ archives. Image conversion belongs in a separate tool,
such as mangapress or KCC. Covers follow page ordering; there is no cover-detection step.

Volume assignments can come from names or a local metadata file. Tools that retrieve online
metadata can write that file; Mangabind itself stays offline. See the decisions on
[covers](docs/adr/0005-drop-cover-detection.md) and
[metadata](docs/adr/0010-local-metadata-file.md).

Changes to JSON reports or progress events must preserve the
[machine protocol](docs/machine-protocol-v1.md), or explicitly version a breaking change.

## Pull requests

Keep each PR focused and explain the change and its verification. Write code comments,
documentation, commit messages, and PR descriptions in English. Check instructions against the
current CLI and workflows. Preserve accepted ADRs and audit evidence; record a changed decision
in a new ADR when it needs one.

Before merging, verify the remote checks for the exact PR commit and wait for the maintainer's
approval. Passing local checks does not replace a successful remote run. If the run cannot be
verified, report the commit and the visibility problem rather than treating the work as complete.
Report suspected vulnerabilities privately through [SECURITY.md](SECURITY.md).

The [architecture decisions](docs/adr/README.md) explain the design. The
[October 2026 audit](docs/audit-2026-10.md) records findings at its stated snapshot; check linked
follow-ups and current code before picking up a finding.
