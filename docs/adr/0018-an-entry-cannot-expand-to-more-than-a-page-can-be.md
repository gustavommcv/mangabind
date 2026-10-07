# 18. An archive entry cannot expand to more than a page can be

Date: 2026-10-07

## Status

Accepted.

## Context

A `.cbz` chapter is a zip, and mangabind copies its pages into the volume without looking at them: through `io.Copy`, into an archive whose entries are stored, not compressed. An audit of the released binary ([`docs/audit-2026-10.md`](../audit-2026-10.md), finding 6) made a 407,791-byte `.cbz` whose one entry was 400 MiB of zeros, deflated; mangabind wrote a **419,430,542-byte volume** from it in a second, with no warning, and nothing in the program limited what an entry could expand to. A deflate stream can expand a little over a thousand to one, so a few megabytes of archive can fill a disk.

[ADR 0014](0014-links-stay-inside-the-input.md) already takes the threat model as given: the people most exposed are those who extract or download a manga archive made by someone else. A bomb is the same case through another door. The audit asked whether mangabind should defend against it at all or leave it to the choice of sources, and the owner decided that it should, with generous ceilings.

## Decision

- **An image entry of a `.cbz` chapter is left out when, by the sizes its own header declares, it is too large to be a page:** it expands to more than **256 MiB**, or it expands to more than **1000 times** what it is stored as and is more than **16 MiB**. A page of a comic is a few megabytes and a scan at print resolution a few tens; nothing is a quarter of a gigabyte. Real images do not compress a thousand to one, and the small entries that do (a blank bitmap) are below the 16 MiB floor, where they can do no harm. The check costs nothing: it reads the sizes already in the archive's directory.
- **The declared size is the real one as far as this matters**, because `archive/zip` refuses to read more from an entry than its header declares, so a header cannot understate what it expands to. The reader returns an error as soon as more comes out than was declared, so a header that understates makes the copy fail: it is reported as a failed volume (the next volume is still written), not as a bomb, and a test builds such an archive.
- **The entry is left out, not an error, and it is said:** one warning per chapter, `page_entries_too_large` at the `inspect` stage, naming the chapter and its volume and number. On the terminal it names up to three entries with how large they are and what they are stored as (`"002.jpg" (24.0 MiB, from 24.5 KiB)`); in the report the diagnostic lists them with exact sizes. `-quiet` does not hide it. A chapter left with no pages is an empty chapter, as before.
- **Entries that are not images are not affected**: they are already left out by [ADR 0016](0016-a-page-is-an-image.md), for what they are, and never read.
- **The writer asks the scanner.** When an archive has two entries of one name and one of them is left out, the nth page that asks for the name must still get the nth entry that was listed, so the writer indexes only the entries `scanner.IsPageEntry` accepts.
- **Plain files in a chapter folder are not limited.** Nothing expands there: a large file costs what it costs to copy, and the person put it there.
- The limits are constants in `internal/scanner`. They are not options: a person who really has a page larger than a quarter of a gigabyte is not making comics, and one more flag is one more thing to document for the case nobody has.

## Consequences

The 400 KB archive that became a 419 MB volume now becomes a volume without that entry and a warning that says why. The limit is on one entry, not on the sum: an archive of many entries each just under the ceiling can still be large, which is bounded by the size of the archive the person chose, not by a header. A legitimate entry above the ceiling is left out with a warning that names its size, which is how a person finds out; the constants are the place to change if that ever happens. The new code is within protocol version 1.
