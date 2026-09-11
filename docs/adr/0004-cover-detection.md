# 4. Conservative, opt-in cover detection

Date: 2026-09-11

## Status

Superseded by [ADR 0005](0005-drop-cover-detection.md)

## Context

Official volume covers show up in downloaded manga in several different ways, or not at all:

- as a dedicated chapter-0/"cover"/"extra" folder,
- as an explicitly named page (`cover.jpg`, `000-cover.png`),
- or not included at all.

Our own `examples/` fixture is a concrete case of the last one behaving unexpectedly: page 1 of
chapter 1 is the color *Weekly Shonen Jump magazine* title page for that chapter's serialization,
not the tankobon (volume) cover. A naive "first page of the first chapter is the cover" rule
would misidentify this and force the wrong image to the front of the archive.

A false positive here is worse than doing nothing: it reorders a page inside the resulting
`.cbz`, corrupting the reading order in a way that's easy to miss until someone's actually reading
it.

## Decision

Cover detection is a chain of strategies behind one interface, ordered by confidence, mirroring
the chapter-parser design in ADR 0003:

```go
type CoverDetector interface {
    DetectCover(vol Volume) (page Page, found bool)
}
```

- **On by default:** strategies based on explicit signals only - a folder/chapter named as a
  cover/extra (`Ch.0`, "cover", "extra"), or a page file explicitly named as a cover
  (`cover.jpg`, `000-cover.png`).
- **Off by default, flag-gated:** appearance-based heuristics (e.g. "first page of the first
  chapter looks visually distinct from the rest") - useful in some cases, but risky enough that it
  should be an explicit opt-in, not silent default behavior.

When no strategy finds a cover with enough confidence, the volume is assembled with no forced
cover page - natural chapter/page order is left untouched. This is the correct behavior for our
own fixture today.

## Consequences

Most volumes without an explicit cover signal will simply not get a "cover" treatment, even if a
human could visually tell one page apart from the rest. That's an intentional accuracy-over-recall
tradeoff. Users who want the heuristic behavior can opt in via a flag once that strategy exists.
