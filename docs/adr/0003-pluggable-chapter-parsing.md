# 3. Pluggable chapter-name parsing instead of one fixed pattern

Date: 2026-09-11

## Status

Accepted

## Context

Chapter folder names are not consistent, even within a single manga. This isn't hypothetical:

- Chapter archives often use configurable naming templates (`%M%`, `%VOL%`, `%CH%`,
  `%C%`), so the same manga
  saved at different times, or by different sources, can produce different folder-name shapes.
- Real collections frequently have chapters "all mixed up" within one manga because different sources
  produced folder names like `Vol 1 Chapter 5`, `Chapter 6`, and `Bonus 1` for the same series.
- The wider scanlation community has its own long-standing naming convention
  (`Title [lang] - c###-###x# (mag/web/v##) [Extra] [Group]{revision}`, see
  [Daiz/manga-naming-scheme](https://github.com/Daiz/manga-naming-scheme)), which uses `x#`/`y#`/`z#`
  suffixes for special/bonus chapters, not just decimals - our chapter-number model needs to allow
  for that, not just `float64` decimals like `12.5`.
- Language shows up as a separate token (e.g. `pt-br`, `zh-hk`, `ja-ro`) that must not
  be confused with title, group, or chapter title during parsing.

Our own `examples/` fixture (Chainsaw Man vol. 1, standard scan naming) confirms the
`Vol.NN Ch.NNNN - Title (lang) [Group]` shape works for one real case, but we should not assume
it's the only one we'll ever see.

Guessing wrong is worse than not guessing: a chapter silently placed in the wrong volume, or
merged in the wrong order, is a corrupted archive that looks fine until someone reads it.

## Decision

Chapter-folder parsing is a chain of strategies behind one interface:

```go
type ParsedChapter struct {
    Volume  *float64 // nil when not present in the name
    Chapter float64
    Special string   // "x1", "y2", ... for bonus/special chapters; empty otherwise
    Title   string
    Lang    string   // informational only; never drives grouping logic
}

type ChapterNameParser interface {
    Parse(dirName string) (ParsedChapter, bool) // bool = parsed with enough confidence to trust
}
```

A registry tries specific parsers first (each one owning one known convention, e.g.
`Vol.NN Ch.NNNN - Title (lang) [Group]`, `Chapter N`, `cNNN`, `Volume N - Ch. NNN`) and falls back
to a generic heuristic regex parser only as a last resort. A folder that no parser can confidently
handle is **not** guessed at - it's surfaced in an "unprocessed" report for the user to resolve,
rather than silently mis-grouped.

Volume grouping only trusts an explicit volume number found in the name by default. Inferring a
missing volume number from neighboring chapters is a real idea for later, but it's a separate,
higher-risk strategy we're deliberately not defaulting to yet.

## Consequences

Adding support for a naming convention we haven't seen yet is: write one small parser + table
tests, register it, done - no risk to existing conventions. This is also intended to be the
easiest kind of first contribution (see CONTRIBUTING.md). The cost is more files/indirection than
a single regex would need; we accept that in exchange for isolation between conventions.
