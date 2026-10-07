# 16. A page is an image

Date: 2026-10-07

## Status

Accepted.

## Context

A chapter folder was read with `os.ReadDir` and every file in it was a page; a `.cbz` chapter was read the same way, every entry that is not a folder. mangabind goes by name and never opens a file, so a page was whatever the folder held. An audit of the released binary ([`docs/audit-2026-10.md`](../audit-2026-10.md), finding 2) packed seven "pages" from a folder with three images and four other files:

```
c001/p0001.hidden  c001/p0002.jpg  c001/p0003.jpg  c001/p0004.JPG
c001/p0005.xml     c001/p0006.db   c001/p0007.txt
```

A name that starts with `.` sorts before `001`, so a `.DS_Store` (which macOS writes into every folder Finder opens) became the first page of its chapter; a `.cbz` made on macOS brings a whole `__MACOSX/` tree of `._001.jpg` sidecars, plus `ComicInfo.xml` and `Thumbs.db`, and each one was a page too. The reports counted them (`page_count`), so a consumer showed the wrong number. A reader that opens the `.cbz` shows the junk as pages or refuses the book; mangapress reads on and skips them with its `skipped_non_images` warning, which is why the final books were right and the counts were not.

The scanner already knew the idea: at the top of the manga folder `.DS_Store`, `Thumbs.db` and `desktop.ini` are junk, and an unrecognised file is reported rather than dropped (ADR 0007). It was applied only there.

## Decision

- **A page is an image, by its name**, compared without regard to case: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.bmp`, `.avif`, `.jxl`, `.tif`, `.tiff`. The list is wider than what mangapress reads (jpg, jpeg, png, gif, bmp, webp), on purpose: a `.cbz` is for any comic reader, and one that opens avif, jxl or tiff should get them. mangapress skips the ones it does not read, with its own warning.
- **The rule is the same for a chapter folder and for a `.cbz` chapter**, and an archive entry is judged by the last element of its path with either separator (archives made on Windows use backslashes).
- **Known junk is left out without a word**: `.DS_Store`, `Thumbs.db`, `desktop.ini` and `ComicInfo.xml` (any case), any file whose name starts with `._`, and anything under a `__MACOSX` folder. It is never a page and nobody wants to be told about it a hundred times.
- **Any other file is left out with a warning**, so that an unusual image extension is never dropped unseen. The warning is `unsupported_page_files` at the `inspect` stage, **one per chapter**, not one per file: a downloader that puts a credits page in every chapter would otherwise produce a hundred lines of the same complaint. It names the chapter, its volume and number, and in `related_paths` every file left out of a folder (for a `.cbz`, where there is no file to give, the entries are in `diagnostic`). On the terminal it is one `warning:` line that names up to three of the files and says how many more there are, and `-quiet` does not hide it.
- **Links are judged first.** A link that does not lead to a file inside the input is still `link_skipped` (ADR 0014), whatever its name; one that does is then judged by its own name.
- **The count follows.** `page_count`, the plan, the progress counters and the volumes' pages are the images. A chapter left with no images is an empty chapter, as before; if it held other files they are named in the warning as well.
- The decision is made on names only. mangabind still never opens a page, so a file with an image name that is not an image goes through, as before, and a file with a wrong extension is left out.

## Consequences

Volumes made from a folder that holds junk are different from before: shorter by exactly the files that were never pages, and the first page is the first image. The page numbers inside a chapter (`p0001`, `p0002`, …) are given after the filtering, so they have no gaps. The new code is within protocol version 1, which ADR 0011 allows: consumers must ignore codes they do not know, and one that does not know it still sees an ordinary warning with a message. Mangabound shows only the codes it has listed, so it will not show this one until it is added there.

A folder with files that are images but are not named like one (no extension, an extension from a rare format) now loses them, with a warning that names them. That is the cost of a rule that does not guess, and the warning is how a person finds out; the list is a small file to extend.
