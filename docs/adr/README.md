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
