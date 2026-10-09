# 19. The pages of a chapter are put in order as KCC puts them

Date: 2026-10-09

## Status

Accepted by the owner on 2026-10-09. It becomes effective with the pull request that carries it.

## Context

Mangabind orders the pages of a chapter itself, and the volume it writes renames them `p0001`, `p0002`
and so on, so the order it chooses is the order of the book. The book is meant to be read as the same
pages Kindle Comic Converter (KCC) would give from the same folder, and the converter this project
publishes, mangapress, is held to KCC's behavior (its ADR 0013).

A comparison of mangapress with KCC 12.0.0 on generated books found that mangapress ordered names
differently from KCC in two ways, and that mangabind, which used the same rule, had them too:

- **Extension.** KCC compares a name without its extension first: `p01.png`, `p01 (2).png`, `p01-2.png`,
  and `1.png`, `1.5.png`, `1.10.png`, `2.png`. Mangabind compared the whole name, so `p01 (2).png` came
  before `p01.png`, and `1.5.png` before `1.png`, by a space or a dot sorting before `.png`.
- **Digits.** KCC counts the digits of every script as numbers (`１０` is ten). Mangabind counted only
  ASCII digits, so `１０.png` came before `２.png`.

Inserted pages (`12.5.png`) and copies (`page (2).jpg`) are the names where it shows. Mangabind also
compared case: `B.jpg` before `a.jpg`; KCC ignores it.

## Decision

- **The pages of a chapter, in a folder or in a `.cbz`, are put in the order of KCC's `natsort` library:**
  a name is split into its stem and up to two short extensions, none that starts with a digit, and the stem
  is compared first; a number is a run of decimal digits of any script, or one character that stands for a
  digit (`²`, `①`), of any size; case is ignored; the text between numbers is compared by code point.
- **In a `.cbz`, a path is compared component by component**, and where one has a file and the other a
  folder at the same level the file comes first, so a folder's own pages come before the folders inside it,
  as in KCC and in mangapress.
- **The order is checked against the real library**, not against a reading of it: `internal/naturalsort/testdata/natsort_vectors.txt`
  is the order natsort 8.4.0 gives for 1,657 names, produced by `tools/parity/natsort_vectors.py` in the mangapress
  repository, and a test sorts every group back to that order.
- **What is not changed:** the order of chapter units (`Less`), which are read for their numbers and sorted
  by them; the order of the lists of skipped and oversized names, which are only reported.
- **What stays different from KCC, by its nature:** natsort follows the locale of the machine for the text
  between numbers, so the order of punctuation and accents varies there. Mangabind takes the one answer that
  does not (code point order, which is what natsort gives in the C locale). Case is folded with lower-casing, not
  Python's full case folding (`ß`).

## Consequences

A chapter whose pages are named `1.png`, `1.5.png`, `2.png`, or with `(2)` copies, comes out in the order a
person reading the folder would expect and KCC would give; before, those pages were out of place. A chapter
whose names have no such pattern (`001.jpg`, `002.jpg`, `Page 1.png`) comes out in the same order as before,
apart from case: a page `Z.png` no longer comes before `a.png`. A `.cbz` with some pages beside a folder
of others lists the loose ones first. The new code is within protocol version 1.
