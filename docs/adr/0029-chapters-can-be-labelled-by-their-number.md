# 29. Chapters can be labelled by their number

Date: 2026-10-24

## Status

Accepted. The owner asked, on 2026-10-24, that the table of contents of a book made from a whole series name its
chapters as they are numbered, also where a chapter has more than one version. It becomes effective with the pull
request that carries it. It adds to [ADR 0006](0006-chapter-directories-for-kcc-toc.md) and is within protocol
version 1.

## Context

The converter takes the name of a chapter in the table of contents from the folder the chapter is in inside the volume
([ADR 0006](0006-chapter-directories-for-kcc-toc.md)), and mangabind named that folder `c003 - Title`, with the
**place** of the chapter in the volume. The place was chosen because it is always unique and always in order, whatever
the chapters are called. It is also not the chapter: in a series with no volumes, where every chapter is in the one
volume, `c007` is the seventh chapter that was found, which is chapter 8 when chapter 3 is missing, and when two
versions of chapter 6 are both kept it is the second version of 6. A table of contents that says the wrong number is
worse than one that says none.

The converter reads the folders of a volume in natural order of their names
([ADR 0019](0019-pages-are-put-in-order-as-kcc-does.md)), so a name that carries the number carries the order only
where the natural order of the names is the reading order, which it is not always: `Ch.21.5` is before `Ch.21x1` by
name and after it in a volume.

## Decision

- **A new flag, `-chapter-labels`, chooses how a chapter is named inside its volume.** `position` is the default and
  is what mangabind has always done. `number` names a chapter by its own number: `Ch.6 - Title`, `Ch.5.5`,
  `Ch.21x1`, with ` (version 1 of 2)` after the title for a chapter kept as one of several
  ([ADR 0028](0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md)). The number is the one in the
  chapter's name, without its zeros.
- **The order is checked, not assumed.** After naming the chapters of a volume by number, mangabind puts the names in
  the natural order the converter will, with the same rules it uses for the pages of a folder, and keeps them only if
  that is the order of the chapters. For a volume where it is not, every name gets its place in front,
  `c003 - Ch.21x1`, which is always in order; the number is still in the name.
- **The report says what each chapter is called.** A volume has `chapter_entries`: the label of each chapter in
  reading order, as the book's table of contents will show it, the absolute path of the unit it came from, and its page
  count. A consumer shows the person exactly which file became which entry.
- **Alternatives are told apart in the default style too**, with the same suffix after `c003 - Title`, since the
  place alone would not tell the second version of a chapter from the next chapter.
- **Capability** `chapter-labels` is added to the list of `--protocol-version`.

## Consequences

Nothing changes for a run that does not ask for `number`, apart from the suffix on alternatives and the report fields.
A book made with `number` has a table of contents of real chapter numbers, with the title when the name had one, and
the order of the chapters is unchanged. The cost is a name that may be long where a title is.
