# 12. Combine an entire series into one volume, still reporting each volume's chapters

Date: 2026-09-27

## Status

Accepted

## Context

For a very long series (One Piece, 100+ volumes), grouping chapters into one `.cbz` per volume
(ADR 0006) is still too fragmented for someone who wants the whole series as a single continuous
read: that's still 100+ separate files. Mangabound wants an option to bind the entire series into
one file instead, while a downstream tool (mangapress) builds a two-level table of contents inside
it - volume entries as parents, chapter entries nested underneath, the same way ADR 0006's
chapter-directory convention already gives KCC-family tools one TOC entry per chapter.

Mangabind and a consumer like Mangapress are only ever glued together by Mangabound handing one a
file path and reading the other's output - there is no side-channel manifest between the two CLIs,
and this decision doesn't invent one. ADR 0006 already established that a `.cbz`'s own top-level
subdirectories become KCC-family tools' chapter list. The same mechanism, one directory level
deeper, can carry volume boundaries too: `<volume dir>/<chapter dir>/pNNNN.ext` instead of a normal
volume's `<chapter dir>/pNNNN.ext`.

## Decision

- A new `-combine` flag. `grouper.Group` is completely unchanged - volume boundaries are computed
  exactly as they always are. Only how the result is written differs: `grouper.CombineVolumes`
  flattens every volume's already-grouped pages into one ordered list, prefixing each page's
  existing chapter-directory archive name with one more directory for its volume
  (`volumeDirName`, mirroring `chapterDirName`'s own position-prefixed, collision-free naming).
  `cbz.SeriesFileName` names the one output file, e.g. `"Chainsaw Man.cbz"`.
- The machine report gains one new, purely additive field: `manga[].combined_output_path`. Per
  ADR 0011's evolution rule, this needs no protocol version bump. When set, every volume record
  still lists its own `number`, `page_count`, and `chapters` - a consumer can still show "this
  series has N volumes, here's what's in each" - but `written` stays `false` for each one and
  `output_path` points at the one combined file, since no separate per-volume file exists anymore.
- `-combine` composes with `-batch`: each manga in a library still gets its own combined file, not
  one file for the whole library.
- A new issue code, `combined_write_failed`, covers a failed combined write the same way
  `volume_write_failed` already covers a failed per-volume write.

The normative field definitions are updated in
[`docs/machine-protocol-v1.md`](../machine-protocol-v1.md).

## Consequences

Mangabound can offer "bind the whole series as one volume" without Mangabind needing to know
anything about tables of contents, and without Mangapress needing a new communication channel with
Mangabind - it just needs to recognize one more directory level when it's told to expect one.
Existing behavior (no `-combine`) is completely unchanged; every existing test still passes
unmodified.

Known limit, left as it is: Mangabind has no opinion on whether a downstream tool can actually turn
this into a two-level table of contents for every output format - that's entirely a Mangapress
(or other consumer) concern, and outside what this tool reports or guarantees.
