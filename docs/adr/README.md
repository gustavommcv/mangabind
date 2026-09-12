# Architecture Decision Records

A log of the significant decisions made on Mangabind, in the order they were made. Each one
explains the context and reasoning at the time, not just the outcome - read the actual ADR when
you need the "why," not just the "what."

| # | Decision |
|---|---|
| [0001](0001-record-architecture-decisions.md) | Keep this log at all, and how |
| [0002](0002-language-and-runtime-go.md) | Implement Mangabind in Go |
| [0003](0003-pluggable-chapter-parsing.md) | Chapter-name parsing is a chain of pluggable strategies, not one regex |
| [0004](0004-cover-detection.md) | ~~Conservative, opt-in cover detection~~ - superseded by 0005 |
| [0005](0005-drop-cover-detection.md) | Drop cover detection entirely - not Mangabind's call to make |
| [0006](0006-chapter-directories-for-kcc-toc.md) | Put each chapter in its own directory inside the `.cbz`, for KCC's table of contents |
| [0007](0007-cbz-chapter-support.md) | Support `.cbz` chapters; never exit having done nothing without saying so |
| [0008](0008-simple-binary-releases.md) | Ship raw binaries + install scripts, not Homebrew/Scoop |
| [0009](0009-cli-conventions-and-batch-mode.md) | Adopt clig.dev CLI conventions; add `--batch` mode |
