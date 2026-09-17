# 7. Support .cbz chapters, reject .epub/.pdf chapters loudly, never go silent

Date: 2026-09-11

## Status

Accepted

## Context

A user testing Mangabind hit a case where the tool ran, exited successfully, and printed
*nothing at all* - no volume written, no warning, nothing. The cause: manga collections may contain
various chapter formats - "Folder with Images" (Mangabind's assumed input so far), or
"Comic Book Archive (*.cbz)", "E-Book Publication (*.epub)", "Portable Document Format (*.pdf)".
They had .cbz files, so the input directory contained one `.cbz` file per chapter instead of
one subfolder per chapter. `scanner.Scan` only ever looked at directories
(`if !e.IsDir() { continue }`), so every file was silently skipped, every downstream stage
received an empty list, and nothing in `cmd/mangabind` ever printed - not even an error, since an
empty result isn't an error condition in any of the existing code paths.

Two separate problems, both worth fixing:

1. Mangabind must never exit having done nothing without saying so.
2. Of the four formats, is it worth supporting all of them?

Looking at the other two: a `.cbz` is just a zip of raw scan images, the same problem Mangabind
already solves, in a different container - reading it needs nothing beyond `archive/zip`, already
a dependency via the cbz writer. An `.epub` or `.pdf` chapter, on the other hand, is already a
*finished* reading document (an EPUB carries its own XHTML wrapper pages, OPF manifest/spine, and
navigation; a PDF has its own page/content-stream structure). Correctly merging several of those
per-chapter documents into one coherent volume means re-implementing a meaningful chunk of an
EPUB or PDF assembler - reconciling internal IDs, manifests, and navigation across files - which
is a fundamentally different and much larger problem than reorganizing raw images. It also isn't
really Mangabind's problem to solve: an `.epub` or `.pdf` has already completed the "turn scans into a reading document" step Mangabind is supposed
to hand off to KCC. Supporting it would mean either adding real third-party dependencies (no PDF
support exists in Go's standard library) or reimplementing EPUB internals - both push against the
project's stated goals: stay single-purpose, keep the dependency list at zero (ADR 0002).

## Decision

- `scanner.Scan` now classifies each entry under the input root as a chapter folder, a `.cbz`
  chapter archive, or (for anything else besides a short list of known OS junk files like
  `.DS_Store`) an unsupported file reported back to the caller instead of silently dropped.
- `.cbz` chapters are read via `archive/zip` (`scanner.PagesInArchive`) exactly like folders are
  read via `os.ReadDir` (`scanner.Pages`); `grouper.Chapter` and `grouper.Page` gained an
  `IsArchive`/`SourceInArchive` distinction so the cbz writer can copy a page's bytes straight from
  inside a source `.cbz` into the output volume, with no extraction to disk.
- `.epub` and `.pdf` chapters are **not** supported, but a file with either extension found in the
  input directory now produces an explicit warning naming the format and explaining that only
  folders of images or `.cbz` are supported - not just a generic "unrecognized file" message.
- `cmd/mangabind` now always prints an outcome: if scanning finds nothing at all, or if entries
  were found but zero volumes came out of grouping, it says so explicitly instead of exiting quiet.

## Consequences

Mangabind now supports both folder and `.cbz` raw chapter formats. Someone who inputs `.epub`
or `.pdf` gets a clear, specific reason why nothing happened rather than silence - and someone
whose input directory turns out to be entirely empty or unusable gets the same. `.cbz`-per-chapter
required no new dependency; `.epub`/`.pdf` support remains deliberately out of scope.
