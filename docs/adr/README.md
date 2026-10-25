# Architecture Decision Records

A log of the significant decisions made on Mangabind, in the order they were made. Each one
explains the context and reasoning at the time, not just the outcome - read the actual ADR when
you need the "why," not just the "what."

| # | Decision |
|---|---|
| [0001](0001-record-architecture-decisions.md) | Keep this log at all, and how |
| [0002](0002-language-and-runtime-go.md) | Implement Mangabind in Go (the version it names is superseded by `go.mod`) |
| [0003](0003-pluggable-chapter-parsing.md) | Chapter-name parsing is a chain of pluggable strategies, not one regex |
| [0004](0004-cover-detection.md) | ~~Conservative, opt-in cover detection~~ - superseded by 0005 |
| [0005](0005-drop-cover-detection.md) | Drop cover detection entirely - not Mangabind's call to make |
| [0006](0006-chapter-directories-for-kcc-toc.md) | Put each chapter in its own directory inside the `.cbz`, for KCC's table of contents |
| [0007](0007-cbz-chapter-support.md) | Support `.cbz` chapters; never exit having done nothing without saying so |
| [0008](0008-simple-binary-releases.md) | Ship raw binaries + install scripts, not Homebrew/Scoop - amended: the scripts check what they download |
| [0009](0009-cli-conventions-and-batch-mode.md) | Adopt clig.dev CLI conventions; add `--batch` mode - partly superseded by 0011 and 0015 |
| [0010](0010-local-metadata-file.md) | Resolve missing volume numbers from a local metadata file, not a network provider - amended: limits and duplicates |
| [0011](0011-versioned-machine-report.md) | Add a versioned JSON planning and execution report; preserve ADR 0009's human CLI behavior - amended: exit code 130 |
| [0012](0012-combine-series-into-one-volume.md) | Add `-combine`: write a whole series as one `.cbz`, still reporting each volume's chapters |
| [0013](0013-opt-in-machine-progress.md) | Add optional structured progress on stderr without changing the final JSON report or human output |
| [0014](0014-links-stay-inside-the-input.md) | Follow a symbolic link only when it leads to a file inside the input; skip and report the rest - amended: a library's links |
| [0015](0015-the-command-line-refuses-what-it-does-not-understand.md) | The command line refuses what it does not understand, and keeps help and notices off the result's stream |
| [0016](0016-a-page-is-an-image.md) | A page is an image, by its name; known junk is left out silently and any other file with one warning per chapter |
| [0017](0017-volumes-are-written-whole-or-not-at-all.md) | A volume is written under a part name and moved into place only when complete; Ctrl-C stops cleanly with exit code 130 |
| [0018](0018-an-entry-cannot-expand-to-more-than-a-page-can-be.md) | An archive entry that declares it would expand to more than a page can be is left out with a warning |
| [0019](0019-pages-are-put-in-order-as-kcc-does.md) | The pages of a chapter are put in the order KCC's natsort gives: stem before extension, digits of every script, case ignored |
| [0020](0020-full-width-digits-and-spaces-in-chapter-names.md) | The digits, the full stop and the space of a chapter name are read in their full-width forms too; titles are returned as written |
| [0021](0021-japanese-chapter-markers.md) | A chapter written `第N話`, `N話` or `#N`, with or without a title, is read; a range, a name with no number and a title before the marker are not |
| [0022](0022-japanese-volume-markers.md) | A chapter that begins with its volume as `第N巻` or `N巻` is read with that volume; a volume alone, a range of volumes and a title before the marker are not |
| [0023](0023-japanese-markers-after-a-prefix.md) | `第N話`, `N話` and the volume markers may follow an author or a title, read from the leftmost place a prefix may end; a prefix that holds a volume is refused |
| [0024](0024-a-volume-named-folder-is-a-volume-of-one-chapter.md) | A folder or .cbz named for a volume alone (`第N巻`, `N巻`) is volume N with one chapter; a range of volumes and a total (`全10巻`) are not read |
| [0025](0025-zip-archives-are-chapters-like-cbz.md) | A `.zip` of images is a chapter or a volume exactly as a `.cbz` is, and reported with kind `cbz`; `.rar` and `.cbr` are named as unsupported, with what to do |
| [0026](0026-an-english-volume-named-unit-is-a-volume-of-one-chapter.md) | A unit named for a volume alone in English (`v01`, `Vol.01`, `Volume 1`, after the series title) is volume N with one chapter; a prefix that holds a chapter ends the search too |
| [0027](0027-volume-folders-hold-chapters.md) | A folder named for a volume alone, with no pages and chapter folders or archives in it, gives its volume to the chapters in it; units are matched by path |
| [0028](0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md) | The mapping file may choose which of the files that claim one chapter to use (`chapter_versions`): one, several kept as alternatives, or none; a file not chosen is `not_chosen`, and the report tells the versions before there is a volume |
| [0029](0029-chapters-can-be-labelled-by-their-number.md) | `-chapter-labels number` names a chapter inside its volume by its own number (`Ch.6 - Title`), in an order that is checked, and the report tells each chapter's label |
| [0030](0030-a-series-without-volumes-is-one-book-of-chapters.md) | `-no-volumes` binds every chapter of a manga into one `.cbz` with no volumes, whatever the names, the folders or the mapping file say; the book's table of contents is the chapters in order |
