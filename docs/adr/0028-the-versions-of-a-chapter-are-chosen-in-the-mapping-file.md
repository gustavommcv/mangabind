# 28. The versions of a chapter are chosen in the mapping file

Date: 2026-10-24

## Status

Accepted. The owner asked, on 2026-10-24, that a person using Manga2Ink be able to choose between two files of the
same chapter without moving, renaming or deleting any of them. It becomes effective with the pull request that
carries it. It adds to [ADR 0005](0005-drop-cover-detection.md)'s rule that nothing is guessed and to
[ADR 0010](0010-local-metadata-file.md)'s mapping file, and it is within protocol version 1
([ADR 0011](0011-versioned-machine-report.md)).

## Context

A folder of chapters downloaded from more than one source often holds the same chapter twice: from two scan groups,
in two languages, or once more after a re-release. Two files that claim one chapter number in one volume are a
conflict, and mangabind leaves **both** out of the book, as it has always done, because it has no way to know which is
the right one. The only way out was to move or delete a file in the person's folder, and a program that is asked to
read a collection has no business asking for that.

Two things made it worse for a consumer. A conflict exists only inside a volume, so for a series with no volumes the
files that claim a chapter were reported as chapters with no volume, and nobody could be asked which to use until a
mapping gave them one. And the consumer had no way to say which: whatever it decided, mangabind still read the whole
folder and found both files.

## Decision

- **The mapping file may carry the choice.** A new optional list, `chapter_versions`, has one entry for a chapter:
  `{"chapter": "6", "use": ["Ch.0006 (en) [A].cbz"]}`. `chapter` is one chapter token (`"6"`, `"6.5"`, `"21x1"`; a
  range is not one chapter), and `use` the files to use, each as its path below the manga folder, written with `/`
  (`Volume 1/Ch.0006 (en) [A].cbz`; a `\` is read as `/`). One file is the choice of a version; several are all kept,
  as alternatives of the chapter; none leaves the chapter out. The file is the one consumers already write, so the
  folder of the person is never touched, and a consumer that writes its own temporary mapping can keep the choices
  wherever it likes.
- **A file that is not chosen is `not_chosen`.** A new disposition of a unit: it was read, and left out because the
  mapping file chose others. It is not a conflict. A chapter that was left out of the book entirely on purpose is
  still a gap in its volume, which the gap warning has always said of any missing chapter.
- **Alternatives are kept side by side and told apart.** The files that were all chosen are bound one after the other,
  in the order they were found, and each is labelled with its place: `(version 1 of 2)` after its title. They are
  not a conflict. Their chapter number is still theirs, which is what the labels by number of
  [ADR 0029](0029-chapters-can-be-labelled-by-their-number.md) keep.
- **A choice is about a chapter that several files claim, and nothing else.** A chapter claimed by one file has
  nothing to decide and is bound whatever the choice says: a choice made before a duplicate was removed does not take
  the chapter away. A choice that names none of the files that claim a chapter is not used, and is told by a new
  warning, `chapter_choice_unmatched`; the files are then a conflict, as they were. Two choices for one chapter, or a
  chapter that is a range, make the file unusable (`metadata_load_failed`), as any other fault in it.
- **The report tells the versions before there is a volume.** Each manga report has `chapter_versions`: one entry for
  every chapter that more than one file claims, with the chapter, the special suffix, the effective volume when
  there is one, the absolute paths of the files in the order they were found, and `resolution`, `chosen` or
  `unresolved`. The files are those with one chapter number, one special suffix and one effective volume (none, for
  files that have none), which is the group they would be a conflict in once they have a volume. A consumer can ask
  the question first and write the mapping after.
- **The report tells what a unit looks like.** Each unit has `first_page`, the name of the first image in reading
  order (a file of the folder, or an entry of the archive), so that a consumer can show the unit by its cover
  without listing it again.
- **Capabilities.** `chapter-versions` (the choices, `not_chosen`, `chapter_versions` of the report) and
  `unit-first-page` are added to the list of `--protocol-version`, so that a consumer checks for them and not for a
  release.

## Consequences

A series with two versions of a chapter can be bound with one of them, with both, or without the chapter, and the
person's folder is as it was. The default has not changed: with no choice, the files are a conflict and are left
out, with the warning that told it. Nothing in a run that has no `chapter_versions` is different, except that the
report has the new fields.

The choice is made by the path of a file, so a file that is renamed is a file the choice does not know, which is the
case the unused-choice warning is for. A consumer that remembers choices has to remember them by something that
survives its own use of the folder; that is its business.
