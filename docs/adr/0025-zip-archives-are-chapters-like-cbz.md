# 25. A `.zip` of images is a chapter or a volume like a `.cbz`

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for `.zip` to be supported and for `.rar` to be left out for
now and noted. It becomes effective with the pull request that carries it. It extends
[ADR 0007](0007-cbz-chapter-support.md), which gave `.cbz` the same standing as a folder of images.

## Context

A collection scanned from the printed books ("自炊"), and many that are bought or shared without DRM, is kept as
one archive per volume, and the common archive is a `.zip`, often with a folder of the pages inside. mangabind
took a `.cbz` and nothing else: a `.zip` was reported as an unrecognized file and left out, so the collection
that is kept the commonest way came out as no books.

A `.cbz` is a ZIP file that comic readers know by that name. The reader that mangabind uses for it,
`archive/zip` of the standard library, opens a `.zip` as it opens a `.cbz`; nothing in the way entries are
listed, ordered, limited (ADR 0018) or copied depends on the extension. A `.rar` is another matter: it needs a
decoder, and the project has no third-party dependency.

## Decision

- **A `.zip` is read exactly as a `.cbz` is,** whatever the case of the extension: as a chapter archive, with
  its name read as a chapter or a volume, its pages in the order of ADR 0019, its entries limited by ADR 0018,
  and a conflict with a `.cbz` or a folder of the same chapter.
- **The report does not change.** The `kind` of such a unit is `"cbz"`: the field says that the unit is an
  archive of images, which a `.zip` is, and a new value would break a reader that knows the two it was given
  (protocol version 1, [docs/machine-protocol-v1.md](../machine-protocol-v1.md)). The output is still a `.cbz`.
- **`.rar` is not read, for now, and the message says what to do.** `.rar` and `.cbr` (a RAR by another name)
  are reported as `found "…" - .rar archives aren't supported (only folders of images, .cbz or .zip; extract it or
  save it as .zip), skipped`. Reading them needs a decoder, which is a dependency, and is left until it is asked
  for again. `.7z` and `.cb7` are as before, unrecognized files.
- **The messages that list the formats say `.zip` too:** `.epub` and `.pdf` chapters "aren't supported (only
  folders of images, .cbz or .zip)".

## Consequences

A collection kept as a `.zip` per volume or per chapter is bound, and one that mixes folders, `.cbz` and `.zip`
is bound as one. The report, the names inside the books and the protocol are as before, so this is within
protocol version 1. A `.zip` that is not an archive of images (a document, a text file) is a chapter with no
pages and is reported as one, as a `.cbz` of the same would be. The scanner does not look inside a `.zip` to see
whether it holds a manga folder instead of pages: it lists every image entry, at any depth, as it does for a `.cbz`.
