# 23. Japanese markers after a prefix

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for the next steps of the plan for the names of Japanese
manga to be done on the best evidence found. It becomes effective with the pull request that carries it. It
amends [ADR 0021](0021-japanese-chapter-markers.md) and [ADR 0022](0022-japanese-volume-markers.md), which
left a title before the marker for a later step.

## Context

A chapter or a volume that comes from a Japanese source seldom carries only its marker. The names in the tests
of Kavita's parser, which were reported by people who use it, show the shapes:

- `[ハレム]ナナとカオル ～高校生のSMごっこ～　第10話`: an author in brackets, the title, an ideographic space and
  the chapter.
- `(一般コミック) [奥浩哉] いぬやしき 第09巻`: the category the collection uses, the author, the title and the
  volume.
- `諌山創] 進撃の巨人 第23巻`: the same with a bracket lost.
- `Test 5巻`, `Series 10-15巻`: a title and a volume, with and without a range.

Mihon, which downloads from many sources, names a chapter's folder `<scanlator>_<name of the chapter>`, so a
name from a Japanese source reads `Raw_第1話 始まり`.

Every parser of mangabind reads a name from its first character, so all of these were reported as names that
cannot be parsed. Because the marker is at the end of the prefix in all of them, a prefix can be skipped if
it is skipped only where a prefix ends.

## Decision

- **`第N話`, `N話`, `第N巻 第N話` and `N巻 第N話` may come after a prefix.** The name is read from its start and
  then from each place a prefix may end, from the left: after a space (of either width), after a closing bracket
  (`] 】 ) ） 」 』 ］ 〕 》`) and after an underscore, and before any `第`, which begins a marker and nothing else.
  The first place that reads is the answer, so the leftmost marker is the chapter and what follows it is its title.
  The prefix is not kept: the series is the folder the chapters are in, and the title of a chapter is what comes
  after its number.
- **A marker stuck to the letter before it is not looked for** (`タイトル12話`): a number in a title is not a
  chapter. `第` is the exception because nothing but a marker begins with it, and Japanese titles run into it
  without a space (`タイトル第3話`).
- **`#N` stays at the start of the name.** A hash after a title (`Batman #12`) is an issue of something that is
  not this convention's to read.
- **A prefix that holds a volume ends the search.** `Vol.01 第1話`, `v01 12話` and `タイトル 第1-3巻 第2話` are not read:
  the marker would be taken for a chapter with no volume and the volume dropped without a word, where a name that
  is not read is reported. A volume in the name is then either read with its chapter (ADR 0022) or not at all.
- **The English parsers are not changed.** A prefix before `Ch.` or `Vol.` (`Title - c001 (v01) [Group]`) is
  another convention, with its own risks (a number in the title), and is not part of this plan.

## Consequences

A folder named as a Japanese collection names its files is read, and the report, the folder names inside the
books and the protocol are as before, so this is within protocol version 1. Names that were reported as not parsed
and are now read are the only change in what is produced. A name where the marker is stuck to a Latin or kana
character, or the volume of which cannot be read, is still reported.
