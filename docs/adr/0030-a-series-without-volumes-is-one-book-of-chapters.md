# 30. A series without volumes is one book of chapters

Date: 2026-10-25

## Status

Accepted. The owner asked, on 2026-10-25, for a flag that does not group the chapters in volumes, so that a series
that has none can be bound as one book whose table of contents is the chapters alone. It becomes effective with the
pull request that carries it. It is within protocol version 1, and it replaces what Manga2Ink did until now to get the
same result (see Context).

## Context

Mangabind groups chapters into volumes, and a chapter that names no volume is left out until the mapping file gives it
one ([ADR 0003](0003-pluggable-chapter-parsing.md)). For a series that was never published in volumes, or that a reader
wants as one run of chapters, the way to bind it was to say that all its chapters are in one volume, and Manga2Ink did
so with a mapping it wrote for the purpose, and then hid that volume from the table of contents. It works, and it is a
way round the tool: nothing in the command line says "one book of chapters", a person at a terminal has no way to ask
for it without writing that mapping, and the volume `1` that the report then holds is invented by the caller.
[ADR 0006](0006-chapter-directories-for-kcc-toc.md) puts what the app can do in the command line, for whoever uses
the tool alone.

## Decision

- **A new flag, `-no-volumes`, binds every chapter into one book.** Every chapter is put in one volume, whatever its
  name says, whatever volume folder it is in and whatever the mapping file lists it under, and those are not used at
  all: none of them is a conflict. The chapters are in the order of their numbers, as they are in a volume, and a
  chapter that names no volume is in the book too. Chapters with a volume in their name keep it in the report's
  `parser.volume`, since it is what the person wrote.
- **The book is `<manga>.cbz`**, the name of the manga alone, as with `-combine`, and its pages are in the same
  directory per chapter that a volume's are, so the table of contents the converter makes from it is the chapters, in
  order, named as `-chapter-labels` says.
- **The report says it.** The manga report has `without_volumes: true`, and one volume: the whole book, numbered 1
  because a volume has a number, with the path of `<manga>.cbz` and its own `written`. A unit's `effective_volume` is
  that volume, and its `metadata_assignment.status` is `not_applicable`. The issues of a gap and of a chapter that two
  sources claim name no volume, and their messages speak of the book.
- **The versions of a chapter still work.** The mapping file's `chapter_versions` choose among the files that claim a
  chapter, now across the whole series: two volumes that number their chapters again from 1 meet there, and their
  chapters that share a number are a conflict and are left out, as they are within a volume, or are chosen.
- **It cannot be combined with `-combine`**, which nests volumes in one file, and a book with none has nothing to
  nest. The command line is refused with exit code 2 (`no_volumes_with_combine` in a report). It composes with
  `-batch`: each manga is one book.
- **Capability** `no-volumes` is added to the list of `--protocol-version`.

## Consequences

Manga2Ink can ask for "one book of chapters" by name and stops writing a mapping to say so; the tool at a terminal can
do the same. The cost is a number 1 in the report for a book that has no volumes, which `without_volumes` explains,
and that a series that numbers its chapters again in every volume has them in conflict and not renumbered: to bind
those, the volumes are what tells the chapters apart, and `-no-volumes` is not the flag.
