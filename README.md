# Mangabind

[![CI](https://github.com/gustavommcv/mangabind/actions/workflows/ci.yml/badge.svg)](https://github.com/gustavommcv/mangabind/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gustavommcv/mangabind)](https://github.com/gustavommcv/mangabind/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Mangabind reorganizes a chapter-by-chapter manga download (e.g. from
[HakuNeko](https://github.com/manga-download/hakuneko)) into one `.cbz` per volume, ready to hand
off to [Kindle Comic Converter](https://github.com/ciromattia/kcc) or any other reader/converter.

```
HakuNeko (downloads chapter by chapter)
    -> Mangabind (groups chapters into volumes, writes .cbz)
    -> KCC (image processing: resize, format conversion, ...)
    -> your reader (KOReader, Kindle, ...)
```

## The problem

HakuNeko downloads each chapter as its own unit - either a folder of loose images, or (if you pick
that download format) a `.cbz` file - with pages numbered from `001` in every one. Two things make
turning that into per-volume archives non-trivial:

- **Name collisions** - every chapter restarts page numbering, so chapters can't just be merged.
- **Inconsistent naming** - a single manga is often scanned by different groups over time, each
  using a different naming convention (`Vol.01 Ch.0001 - Title (en) [Group]`, `Chapter 1`, `c001`,
  ...). Detection has to handle a mix of conventions within one input folder.

Mangabind accepts a mix of chapter folders and `.cbz` chapter files in the same input directory.
HakuNeko's other two download formats, `.epub` and `.pdf`, aren't supported - those are already
finished reading documents rather than raw scans, and merging them correctly would mean
re-implementing a meaningful part of an EPUB/PDF assembler rather than reorganizing files (see
[docs/adr/0007-cbz-chapter-support.md](docs/adr/0007-cbz-chapter-support.md)). Mangabind reports
any `.epub`/`.pdf` chapter it finds instead of silently ignoring it.

**Out of scope:** any image processing (resize, recompression, cropping, color conversion) - that
is KCC's job, the next step in the pipeline. Mangabind also doesn't try to identify or reposition
a "cover" page - if a source ships its cover as its own chapter (e.g. `Ch.0`), normal
chapter-number ordering already places it first; readers display whatever page ends up first
regardless. Deciding what counts as a cover is the source/scan group's call, not Mangabind's (see
[docs/adr/0005-drop-cover-detection.md](docs/adr/0005-drop-cover-detection.md)). Mangabind only
reorganizes files into `.cbz` containers; it never modifies, moves, or deletes your original
downloaded files.

## Status

Early development - see [docs/adr](docs/adr/README.md) for the design decisions made so far.

## Install

**macOS/Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.ps1 | iex
```

Both scripts download the right binary for your OS/architecture from the
[latest release](https://github.com/gustavommcv/mangabind/releases/latest) and put it on your
PATH - no need to install Go. Prebuilt binaries and checksums for every release are also available
there directly, if you'd rather install manually.

Already have Go and want the dev version instead:

```bash
go install github.com/gustavommcv/mangabind/cmd/mangabind@latest
```

### Update

Run the same install command again - it always fetches the latest release and overwrites the
existing binary in place. `mangabind --version` tells you what you currently have installed.

### Uninstall

Mangabind is a single self-contained binary; there's no installer state to clean up beyond it.

- **macOS/Linux:** `rm ~/.local/bin/mangabind`
- **Windows:** delete `%LOCALAPPDATA%\Programs\mangabind\mangabind.exe`. The installer added that
  folder to your user `PATH`; if you'd rather remove that entry too, it's under Settings > System >
  About > Advanced system settings > Environment Variables > `Path` (User variables).
- **`go install`:** `rm $(go env GOPATH)/bin/mangabind` (or `%GOPATH%\bin\mangabind.exe` on Windows).

## Usage

```bash
mangabind --input /path/to/downloaded/manga [--output /path/to/output]
```

`--output` is optional. If you don't pass it, Mangabind writes to a sibling folder next to
`--input`, named `<input folder> (mangabind)` - never inside `--input` itself, since that would
make the next run see the output folder as a bogus chapter.

Got a whole library instead of just one manga - a folder full of manga folders, each with their
own chapters? Add `--batch` and point `--input` at the library folder; every immediate subfolder
is processed as its own manga, independently (one manga having issues never stops the rest):

```bash
mangabind --input /path/to/manga/library --batch
```

Other flags: `-i`/`-o` are shorthands for `--input`/`--output`; `--dry-run` (`-n`) shows what would
be written without writing anything; `--quiet` (`-q`) suppresses routine progress output, keeping
only warnings/errors; `--version` prints the version. Run `mangabind --help` for the full list.

### Machine-readable integration

`--json` adds a versioned machine report without changing the existing human output mode. Combine it
with `--dry-run` to inspect every chapter, parser result, metadata assignment, warning, conflict,
gap, and intended output without writing files; omit `--dry-run` to execute and report which outputs
were written:

```bash
mangabind --input "/path/to/manga" --output "/path/to/volumes" --dry-run --json
mangabind --input "/path/to/manga" --output "/path/to/volumes" --json
```

`mangabind --protocol-version` returns the compatibility handshake used by GUI consumers. See the
[machine protocol v1 specification](docs/machine-protocol-v1.md) and
[ADR 0011](docs/adr/0011-versioned-machine-report.md). Scripts must check `protocol_version` rather
than infer compatibility from the release version.

## When chapter names don't carry a volume number

Some sources only name chapters `Chapter 1`, `Chapter 2`, ... with no volume information at all -
Mangabind has no way to know which volume each one belongs to, so those chapters are reported and
skipped rather than guessed into the wrong place. Fix that with a small local metadata file:

```json
{
  "schema_version": 1,
  "volumes": [
    { "number": "1", "chapters": ["1-7"] },
    { "number": "2", "chapters": ["8-16", "8.5"] }
  ]
}
```

Save it as `mangabind.json` inside the manga's own input folder and Mangabind picks it up
automatically (or pass `--metadata-file path/to/file.json` explicitly - not combinable with
`--batch`, since each manga in a library needs its own file). `chapters` entries can be a single
number (`"8.5"`, or `"21x1"` for a bonus/special chapter) or an inclusive range (`"1-7"`). A volume
number already present in a chapter's own name always wins; the file only fills in what's missing.
Mangabind never fetches this data itself - see
[docs/adr/0010-local-metadata-file.md](docs/adr/0010-local-metadata-file.md) for why, and where
generating this file from a source like MangaDex should live instead.

## How detection works

Chapter-folder parsing is implemented as a chain of pluggable strategies rather than a single
fixed regex, because naming conventions vary even within one manga. See
[docs/adr/0003-pluggable-chapter-parsing.md](docs/adr/0003-pluggable-chapter-parsing.md) for the
reasoning, and [CONTRIBUTING.md](CONTRIBUTING.md) for how to add a parser for a naming convention
Mangabind doesn't yet recognize.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
