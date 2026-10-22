# 21. Japanese chapter markers: `第N話`, `N話` and `#N`

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for steps 1 to 3 of the plan for the names of Japanese
manga to be done. It becomes effective with the pull request that carries it.

## Context

Chapters of manga that comes from a Japanese source are not named `Chapter 5` or `Ch.005`. The counter of a
chapter in Japanese is `話`, written after the number and usually behind `第`: `第1話`, `第001話 始まり`,
`12話`, `第10.5話`. A downloader that takes the name from the site (Mihon does) and a reader who renames
by hand write the same. The number sign is used too: `#12`.

mangabind 0.7.2 reads a chapter only when its name begins with `Vol`, `Chapter`, `Ch`, `c` or a number, so all
of these were reported as names that could not be parsed and left out of the books. Kavita's parser reads
`第N話`, `N話` and `#N`, and so does the guide to the naming of its scanner.

None of these names carries a volume, as `Chapter 5` does not. The volume comes from the mapping file, or from
the next step (the volume in the name), as for any chapter with no volume.

## Decision

- **A new parser, `JapaneseChapterParser`, reads three shapes:** `第N話`, `N話` (both with a space between the
  parts allowed) and `#N`. `N` is a chapter number as in the other parsers, with an optional decimal part
  (`第10.5話`), in ASCII or full-width digits ([ADR 0020](0020-full-width-digits-and-spaces-in-chapter-names.md)).
- **A title may follow**, after a space (of either width), a dash or a colon (`-`, `:`, `：`), or straight after
  `話` (`第5話旅立ち`). It is returned as written, brackets included (`「約束」`), because it becomes the name of
  the chapter inside the book ([ADR 0006](0006-chapter-directories-for-kcc-toc.md)). After `#N` a title needs a
  separator, so that `#12abc` is not a chapter.
- **It is tried after the conventions in English and before a bare number.** A name that more than one
  convention could read keeps the reading it had.
- **A range is not read.** `第1-3話` would stand for three chapters in one folder; reading it as chapter 1 would
  drop two, so it stays a name that cannot be parsed and is reported.
- **A name without a number (`第話`, `最終話`) is not read either.** Chapters without a number (the last
  chapter, a prologue, an extra) are a separate decision that waits for names from real collections.
- **The marker has to begin the name.** A title before the marker (`[作者] タイトル 第1話`) is not read; that
  belongs to a later step.

## Consequences

A folder or archive named in the Japanese way is a chapter, sorted by its number with the others, and
the mapping file can give it its volume. Names that were reported as not parsed and are now read are the only
change in what is produced; the report, the folder names in the books and the protocol are as before, so this is
within protocol version 1. The parser's name, `japanese-chapter`, is new.
