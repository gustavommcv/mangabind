# Mangabind

[![CI](https://github.com/gustavommcv/mangabind/actions/workflows/ci.yml/badge.svg)](https://github.com/gustavommcv/mangabind/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gustavommcv/mangabind)](https://github.com/gustavommcv/mangabind/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Mangabind groups manga chapters into volumes. Give it a folder of chapter folders or CBZ files,
and it creates one CBZ per volume, or a single archive for the whole series. Pages stay in chapter
order, and their image contents are preserved.

Open the result in a CBZ reader, or convert it for your e-reader with
[mangapress](https://github.com/gustavommcv/mangapress) or
[Kindle Comic Converter](https://github.com/ciromattia/kcc). For a desktop interface that combines
grouping and conversion, see [Mangabound](https://github.com/gustavommcv/mangabound).

[Download](https://github.com/gustavommcv/mangabind/releases/latest) ·
[Get started](#usage) · [Contribute](CONTRIBUTING.md)

## Install

Download and extract a package from [GitHub Releases](https://github.com/gustavommcv/mangabind/releases/latest),
or use the install script below. Releases include binaries for Windows, macOS, and Linux on x64
and ARM64, plus `checksums.txt` for manual verification. Running a release does not require Go.

**macOS / Linux:**

```sh
curl -fsSL https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.ps1 | iex
```

The scripts install the latest release to `~/.local/bin` on macOS/Linux or
`%LOCALAPPDATA%\Programs\mangabind` on Windows. On macOS/Linux, follow the printed instructions
if the folder is not on your `PATH`. On Windows, restart your terminal after installation.

With Go installed, you can also build and install the latest tagged version:

```sh
go install github.com/gustavommcv/mangabind/cmd/mangabind@latest
```

### Update

Repeat the installation command you used. Check the installed version with `mangabind --version`.

### Uninstall

Delete `~/.local/bin/mangabind` on macOS/Linux, or `%LOCALAPPDATA%\Programs\mangabind` on Windows.
If you used `go install`, delete the executable from `GOBIN`, or from `GOPATH/bin` when `GOBIN`
is unset. You can remove the Windows install folder from your user `Path` in Environment Variables.

## Usage

Point `--input` at a manga folder containing its chapters:

```sh
mangabind --input "Manga/Example Series"
```

Chapter folders and chapter CBZ files can be mixed in the same folder. EPUB and PDF input are
not supported. Without `--output`, the archives go into a sibling folder named
`Example Series (mangabind)`. To choose a destination:

```sh
mangabind --input "Manga/Example Series" --output "Books/Example Series"
```

Use `--dry-run` to preview the grouping before writing archives. Read the report for unrecognized
names, missing volume assignments, gaps, or conflicting chapters.

```sh
mangabind --input "Manga/Example Series" --dry-run
```

Use `--input` for the path and quote paths that contain spaces. Unexpected positional arguments
and unknown flags are rejected with exit code 2, so a trailing `--dry-run` cannot be silently
ignored. A mistyped flag may include a suggestion in the error message.

Help goes to stdout. Errors, warnings, and setup notices such as the selected output folder go
to stderr, keeping them separate from results on stdout.

A volume is written under a temporary name (`.<volume>.<random>.part`) in the output folder and
moved into place only when it is complete, so a run that fails or is interrupted never leaves half a
volume, and never damages one that was there. Press Ctrl-C once to stop: the unfinished file is
removed and the exit code is 130; a second Ctrl-C ends the program at once. See
[the decision](docs/adr/0017-volumes-are-written-whole-or-not-at-all.md).

### Process a library

With `--batch`, each immediate subfolder is treated as a separate manga. A failure in one manga
does not stop the others.

```sh
mangabind --input "Manga" --batch
```

### Combine a series

`--combine` writes one CBZ for the whole manga, with volumes containing chapter folders inside
the archive. The chapter-to-volume assignments stay the same.

```sh
mangabind --input "Manga/Example Series" --combine
```

To convert that structure into an EPUB with volumes and chapters in its table of contents, use
mangapress's `--nested-toc` option.

`-i`, `-o`, and `-n` are shortcuts for `--input`, `--output`, and `--dry-run`.
Use `--quiet` (`-q`) to show only warnings and errors, or `mangabind --help` for all options.

## When chapter names don't carry a volume number

Names such as `Chapter 1` identify a chapter but not its volume. Mangabind reports and skips
chapters without a volume assignment. To supply the missing assignments, save a `mangabind.json`
file in the manga's input folder:

```json
{
  "schema_version": 1,
  "volumes": [
    { "number": "1", "chapters": ["1-7"] },
    { "number": "2", "chapters": ["8-16", "8.5"] }
  ]
}
```

Entries can be individual numbers (`"8.5"`), special chapters (`"21x1"`), or inclusive ranges
(`"1-7"`). A volume number in a chapter's own name takes precedence over the file.

Volume numbers go from 0 to 100000, and a file may list at most 100,000 chapters (a range counts
each chapter in it); a file outside that is refused with an error. A chapter listed under two
different volumes gets one warning for the file, and the last listing is the one that applies.

Use `--metadata-file "path/to/mapping.json"` to select a different file. In batch mode, put a
`mangabind.json` in each manga folder; one shared `--metadata-file` is not supported.

Mangabind uses the `volumes` field for grouping and does not fetch metadata online. Other fields,
including `manga` and `source`, may be stored by tools such as Mangabound without affecting
grouping. An empty `volumes` list leaves filename-based grouping unchanged; `schema_version: 1`
is still required. See [the metadata decision](docs/adr/0010-local-metadata-file.md) for the format.

## Files and links

Mangabind copies pages into new archives. It does not resize, crop, recompress, or choose a cover;
the first page follows chapter and page ordering. Keep generated archives outside the input folder.

A page is an image file: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.bmp`, `.avif`, `.jxl`, `.tif` or
`.tiff`, by its name. Other files inside a chapter (a `notes.txt`, a `credits.html`) are left out
with one warning per chapter, and the leftovers of a file manager or downloader (`.DS_Store`,
`Thumbs.db`, `desktop.ini`, `ComicInfo.xml`, macOS's `__MACOSX/` and `._` files) are left out without
one. See [the page rule](docs/adr/0016-a-page-is-an-image.md).

A symbolic link is followed only if it points to a file inside the manga's input folder. Other
links are skipped with a warning. With `--batch`, a link to a manga folder in the library is not
followed either, and is reported the same way. See
[the link policy](docs/adr/0014-links-stay-inside-the-input.md).

## Machine-readable integration

`--json` writes one report to stdout. Add `--progress-json` for live progress on stderr:

```sh
mangabind --input "Manga/Example Series" --dry-run --json
mangabind --input "Manga/Example Series" --json --progress-json >report.json 2>progress.jsonl
```

Use `mangabind --protocol-version` to check compatibility. The
[protocol reference](docs/machine-protocol-v1.md) defines fields, capabilities, issues, and progress
events. Completion is determined by the final report and exit code, not progress alone.

## Contributing

For a bug report, include your version, command, example chapter names, and the expected grouping.
Small examples are usually enough to reproduce a naming problem. See [CONTRIBUTING.md](CONTRIBUTING.md)
for setup, tests, and adding a parser, or browse the [architecture decisions](docs/adr/README.md).
Report suspected vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE).
