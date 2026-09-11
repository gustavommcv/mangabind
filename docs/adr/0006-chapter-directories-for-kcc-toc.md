# 6. Put each chapter in its own directory inside the .cbz

Date: 2026-09-11

## Status

Accepted

## Context

Running a Mangabind-generated `.cbz` through KCC (Kindle Comic Converter) to produce an EPUB
produced a table of contents with a single entry for the whole volume - no per-chapter
navigation - even though the cover displayed correctly in KOReader.

The cause: Mangabind's archive layout was completely flat (`c001_p0001.jpg`, `c001_p0002.jpg`,
`c002_p0001.jpg`, ... all in the zip root). Researching KCC's source confirmed how it actually
builds chapters and a TOC:

- `buildEPUB()` in
  [comic2ebook.py](https://github.com/ciromattia/kcc/blob/master/kindlecomicconverter/comic2ebook.py)
  walks the extracted image directory with `os.walk()`; every distinct subdirectory it encounters
  becomes a separate chapter.
- The chapter's title in the generated NCX/nav comes from that subdirectory's name
  (`chapternames[os.path.basename(folder)]`), unless a `ComicInfo.xml` bookmark overrides it -
  which we don't use.
- Archive extraction
  ([comicarchive.py](https://github.com/ciromattia/kcc/blob/master/kindlecomicconverter/comicarchive.py))
  shells out to `tar`/`7z`/`unar`/`unrar`, passing the target directory straight through with no
  flattening - a `.cbz`'s internal folder structure survives extraction intact.
- A real user's [reported issue](https://github.com/ciromattia/kcc/issues/662) confirms the
  community already relies on exactly this convention (`Vol1/ch1/`, `Vol1/ch2/`, ...) to get
  chapter-separated EPUBs; the only failure mode there is a loose cover file sitting outside any
  chapter directory, which doesn't apply to us since we don't produce a loose cover file at all
  (see ADR 0005).

A flat `.cbz` isn't wrong for reading page-by-page (Calibre, YACReader, KOReader's plain CBZ view
all just sort by full path either way), but it throws away information Mangabind already has -
which pages belong to which chapter - that KCC needs to build real navigation.

## Decision

`grouper.Group` now places each chapter's pages under a directory named `cNNN - Title` (`NNN` is
the chapter's position within the volume, `Title` is the parsed chapter title, sanitized to strip
characters that would create unintended nesting). Archive names become e.g.
`c001 - A Dog and a Chainsaw/p0001.jpg` instead of `c001_p0001.jpg`.

## Consequences

Converting a Mangabind volume through KCC now produces one TOC entry per chapter, titled from the
source chapter's own title. Plain `.cbz`/`.cbr` readers are unaffected - they already sort by full
path, and a directory separator sorts the same as before relative to other entries in the same
chapter. No new external dependency was needed: `archive/zip` already supports directory-shaped
entry names with no special handling.
