# 22. Japanese volume markers: `第N巻` and `N巻` in front of a chapter

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for steps 1 to 3 of the plan for the names of Japanese
manga to be done. It becomes effective with the pull request that carries it.

## Context

[ADR 0021](0021-japanese-chapter-markers.md) reads a chapter written `第N話`, `N話` or `#N`. Such a name carries
no volume, so a manga whose folders are named that way needs a mapping file to be grouped into books.

The counter of a volume in Japanese is `巻`, written after the number and usually behind `第`: `第1巻`,
`第01巻`, `3巻`. A collection that has been sorted by volume and by chapter writes both in the name of the
folder: `第1巻 第2話`, `1巻 #3`, `第01巻 - 第12話 始まり`. That name has the same information as
`Vol.01 Ch.002`, which mangabind has always read.

A folder that is named for a volume alone (`第1巻`, with pages in it or with the book itself in it) is something
else: it is a volume, not a chapter, and the model of mangabind (chapters grouped into volumes) has no place for
it. That is a decision about the product, not about a pattern, and it waits for real collections to look at.

## Decision

- **A new parser, `JapaneseVolumeChapterParser`, reads a volume marker followed by a chapter:** `第N巻` or `N巻`,
  then either a separator (a space of either width, a dash or a colon, none of them, or a dash with spaces)
  and a chapter written as ADR 0021 reads it, with its title. The chapter is read by the same code, so the
  two conventions cannot drift apart.
- **The volume may have a decimal part** (`第1.5巻`), as the volume of the other parsers can, and digits and
  spaces may be full-width ([ADR 0020](0020-full-width-digits-and-spaces-in-chapter-names.md)).
- **It is tried before `JapaneseChapterParser`**, so that a name with a volume is not left to a parser that does
  not look for one. A name of either kind cannot be read by the other.
- **A volume alone is not read.** `第1巻`, and `第1巻 おまけ` (a volume followed by something that is not a
  chapter), are reported as names that cannot be parsed, as they were.
- **A range of volumes is not read** (`第1-3巻 第2話`), for the reason a range of chapters is not: it names
  several volumes and the chapter would belong to one of them.
- **The volume marker has to begin the name**, as the chapter marker does. A title before it
  (`[作者] タイトル 第1巻 第2話`) is not read; that belongs to a later step.
- **The volume in a name takes precedence over the mapping file**, as it does for every other convention
  (README, "When chapter names don't carry a volume number").

## Consequences

A collection named `第1巻 第2話` is grouped into its volumes without a mapping file, and one that is named with
chapters only still needs it. Names that were reported as not parsed and are now read are the only change in what
is produced; the report, the folder names in the books and the protocol are as before, so this is within protocol
version 1. The parser's name, `japanese-volume-chapter`, is new. What a folder named for a volume alone should
become is still open.
