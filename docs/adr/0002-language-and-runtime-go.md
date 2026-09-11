# 2. Use Go as the implementation language

Date: 2026-09-11

## Status

Accepted

## Context

Mangabind needs to be genuinely cross-platform (Windows/macOS/Linux), easy to distribute as a
single binary to non-technical end users, and welcoming to outside open-source contributors. The
core technical work is file/zip manipulation and lightweight image inspection (dimensions/format,
never decoding for resize or recompression - that's explicitly out of scope, see the README).

Three realistic options were considered: Go, TypeScript (Node/Bun), and Python.

- **Single-binary packaging.** Go cross-compiles to a static binary for any OS/arch from a single
  host (`GOOS=... GOARCH=... go build`), with no runtime to bundle. Python's story here is the
  weakest (PyInstaller/Nuitka produce large, slow-starting binaries, sometimes flagged by
  antivirus). Node/Bun have made real progress (Node's Single Executable Applications, `bun build
  --compile`) but it's newer and less battle-tested.
- **Contributor onboarding friction.** `git clone && go build` works with zero extra setup (no
  venv, no node_modules). `gofmt` and `go vet` are built into the toolchain, which removes
  formatting bikeshedding from code review for free.
- **Technical fit.** `.cbz` files are plain zip archives - the `archive/zip` package in Go's
  standard library covers this with no third-party dependency. Cover-detection heuristics only
  need to read image *dimensions*, not decode full images - `image.DecodeConfig` reads just the
  header, which is fast and, again, stdlib-only.
- **Extensibility.** The chapter-naming and cover-detection strategies both need to be pluggable
  (see ADR 0003 and 0004); Go interfaces are a natural, low-ceremony fit for this.

The tradeoff against Go: it's less familiar than Python/JS to some of the scanlation/manga-tooling
community that might otherwise contribute. We judged the lower setup friction (no dependency
manager, enforced formatting) to offset this.

## Decision

Implement Mangabind in Go. Use the standard library wherever it covers the need (`archive/zip`,
`image` for header-only decoding, `flag` for the CLI) before reaching for a third-party
dependency. Use `goreleaser` for cross-platform release builds once we're ready to publish
binaries.

## Consequences

Contributors need a Go toolchain (1.23+) and nothing else. CI runs `gofmt -l`, `go vet`, and
`go test` on a Windows/macOS/Linux matrix. We give up some familiarity with contributors coming
from a Python/JS manga-tooling background, and accept that as a reasonable tradeoff for the
packaging and low-friction-build wins.
