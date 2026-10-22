# 26. A unit named for a volume alone in English is a volume of one chapter

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for the English `Vol.01` and `v01` to be read if it made
sense. It becomes effective with the pull request that carries it. It extends
[ADR 0024](0024-a-volume-named-folder-is-a-volume-of-one-chapter.md), which left the English names as they were,
and it makes the check of [ADR 0023](0023-japanese-markers-after-a-prefix.md) wider.

## Context

It makes sense, because the reason for ADR 0024 is not Japanese: a collection kept as one unit per volume came out
as no books. Kavita documents exactly this shape for English collections: its guide to the naming of manga shows the
volumes as `Series Name v01.cbz`, `Series Name v02.cbz`, and its tests have `Love Hina - Volume 01 [Scans]`,
`Battle Royale, v01 (2000) [TokyoPop] [Manga-Sketchbook]` and
`Highschool of the Dead - Full Color Edition v02 [Uasaha] (Yen Press)`. mangabind read `Vol.01 Ch.001` and none of
these.

Unlike a chapter, such a name always carries the title of the series before the volume, so a name read only from its
first character would read none of them.

## Decision

- **`VolumeOnlyParser` reads `v01`, `Vol.01`, `Vol 1`, `Volume 1`, in any case, as volume N with one chapter**
  numbered 1 and without a title, as the Japanese `第N巻` is read. The volume may have a decimal part (`v03.5`),
  and the groups in parentheses or brackets that a collection puts after it (a year, a publisher, a scanner) are
  allowed.
- **It is read after a prefix, as ADR 0023 says of the Japanese markers:** from the start of the name and then
  from each place a prefix may end (after a space, a closing bracket or an underscore), leftmost first. A `v` stuck
  to a letter (`Dev5`, `Marvel 5`) is not a volume.
- **A name that says more than a volume is not read.** A title after it (`Samurai Jack Vol. 01 - The threads of
  Time`), a chapter after it (`Series v02 c003 - The Invisible Guardian`) and a volume in brackets inside another
  convention (`Title - c001 (v01) [Group]`, whose chapter has no parser here) stay reported as names that cannot be
  parsed.
- **The prefix check also refuses a chapter.** A prefix that holds a volume ended the search for the Japanese
  markers (ADR 0023); it now also ends it when it holds a chapter (`Ch.5`, `Chapter 5`, `第5話`), for the same
  reason: reading the name from the marker would drop what the prefix says. `第1-3話 第5話` is no longer read as
  chapter 5.

## Consequences

A collection kept with one folder, `.cbz` or `.zip` per volume, named as Kavita documents it, is bound into the books
of its volumes. Names that were reported as not parsed and are now read are the only change in what is produced,
apart from the narrower reading of a name whose prefix holds a chapter; the report, the names inside the books and
the protocol are as before, so this is within protocol version 1. The parser's name, `volume-only`, is new.
