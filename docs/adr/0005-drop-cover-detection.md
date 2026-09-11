# 5. Drop the separate cover-detection layer

Date: 2026-09-11

## Status

Accepted (supersedes [ADR 0004](0004-cover-detection.md))

## Context

After the grouper and cbz writer were implemented (no cover-detection logic yet), we tested the
real output against the `examples/Chainsaw Man` fixture in Calibre. Calibre - like essentially
every comic/manga reader - simply displays whatever image is first in the archive as the cover.
That observation exposes something ADR 0004 didn't make explicit: **grouper already sorts
chapters by chapter number**, so if a source ships its cover as its own numbered chapter (e.g.
`Ch.0`), it lands as page 1 automatically, with zero extra logic - the "explicit signal" cover
strategies from ADR 0004 would have been entirely redundant with sorting that already exists.

That leaves only the *heuristic* half of ADR 0004 - trying to identify a cover embedded as, say,
the first page of chapter 1 by how it looks compared to the rest. ADR 0004 already flagged that as
the risky part (a false positive corrupts reading order) and gated it off by default. Thinking it
through further: whether a given page "is the cover" is a judgment call that depends entirely on
how the source scanned/labeled the material - it's not something Mangabind can determine
correctly in general, and guessing wrong is worse than doing nothing. That determination belongs
to the source/scan group, not to a tool whose job is reorganizing files into volumes.

## Decision

Drop the `CoverDetector` interface and the planned `internal/cover` package entirely. Mangabind
does not attempt to identify or reposition a cover page. If a source represents its cover as its
own chapter, normal chapter-number sorting already places it first, for free. If a cover is
embedded inside another chapter without a distinguishing name, Mangabind leaves it exactly where
the source put it.

## Consequences

The pipeline stays scanner -> parser -> grouper -> cbz, with no extra stage. This keeps Mangabind
doing one job - reorganizing chapters into volumes without data loss or guessing - rather than
growing into a tool that makes editorial judgments about content it can't actually evaluate.
