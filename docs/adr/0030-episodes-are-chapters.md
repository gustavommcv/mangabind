# 30. An episode is a chapter

Date: 2026-10-24

## Status

Accepted. The owner asked, on 2026-10-24, why a folder of twelve episodes of a webcomic (`peppercarrot_episode01.cbz`, ...) could not be grouped as a series in Manga2Ink. It becomes effective with the pull request that carries it. It adds a convention to [ADR 0003](0003-pluggable-chapter-parsing.md) and is within protocol version 1.

## Context

Every file of that folder was reported as a name no parser reads (`unparsed_chapter`), so the folder had no chapters to group, no matter what the mapping file said. A webcomic published by episodes (Pepper&Carrot is one) numbers what a manga calls chapters, and the name is the series, then `episode`, then the number.

## Decision

- **`episode` and `ep` are chapter markers**, as `chapter` and `ch` are: `Episode 5`, `Ep. 5`, `ep05`, `Episode 12.5`, `Episode 3x1`, with a title, a language and a group after the number as for a chapter (`Ep.07 - Title (en) [Group]`). An episode has no volume.
- **It may follow the name of the series**, read from the leftmost place a prefix may end, as the Japanese markers are ([ADR 0023](0023-japanese-markers-after-a-prefix.md)): `peppercarrot_episode01`, `Pepper Carrot Episode 7`, `Pepper & Carrot - Episode 07 - The Witch`, `[Group] Series Episode 3`. A prefix that holds a volume or a chapter ends the search, so that what it says is not dropped without a word.
- **The word must be a marker and the number must follow it**: `Epic 5`, `Episodes 5` and `Episode One` are not read, and a marker stuck to the letter before it (`Seriesepisode5`) is not looked for.
- **It is its own parser, `episode`**, tried after the conventions that carry a volume and after `chapter-only`, and the report names it.

## Consequences

A folder of episodes is read as chapters with no volume, which the mapping file, the volumes of an online source or a series made into one book can use as they do for `Chapter 5`. The cost is a name such as `Star Wars Episode 4` read as chapter 4 of nothing: a folder that is a title of a library has chapters inside it and no pages, so it is still read as a library, and one whose name is read wrongly is reported with its chapter number, which a person can see.
