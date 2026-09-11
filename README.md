# Mangabind

Mangabind reorganizes a chapter-by-chapter manga download (e.g. from
[HakuNeko](https://github.com/manga-download/hakuneko)) into one `.cbz` per volume, ready to hand
off to [Kindle Comic Converter](https://github.com/ciromattia/kcc) or any other reader/converter.

```
HakuNeko (downloads chapter by chapter)
    -> Mangabind (groups chapters into volumes, preserves the cover, writes .cbz)
    -> KCC (image processing: resize, format conversion, ...)
    -> your reader (KOReader, Kindle, ...)
```

## The problem

HakuNeko downloads each chapter into its own folder, with pages numbered from `001` in every
folder. Two things make turning that into per-volume archives non-trivial:

- **Name collisions** - every chapter folder restarts page numbering, so folders can't just be
  merged.
- **Inconsistent naming** - a single manga is often scanned by different groups over time, each
  using a different folder-naming convention (`Vol.01 Ch.0001 - Title (en) [Group]`, `Chapter 1`,
  `c001`, ...). Detection has to handle a mix of conventions within one input folder.

Mangabind also detects and preserves the official volume cover when one is present (as a
dedicated chapter-0/cover folder or file), placing it as the first page of the resulting `.cbz`.

**Out of scope:** any image processing (resize, recompression, cropping, color conversion) - that
is KCC's job, the next step in the pipeline. Mangabind only reorganizes files into `.cbz`
containers; it never modifies, moves, or deletes your original downloaded files.

## Status

Early development - see [docs/adr](docs/adr) for the design decisions made so far.

## Install

```bash
go install github.com/gustavommcv/Mangabind/cmd/mangabind@latest
```

## Usage

```bash
mangabind --input /path/to/downloaded/manga --output /path/to/output
```

## How detection works

Chapter-folder parsing and cover detection are both implemented as pluggable strategies rather
than a single fixed regex, because naming conventions vary even within one manga. See
[docs/adr/0003-pluggable-chapter-parsing.md](docs/adr/0003-pluggable-chapter-parsing.md) and
[docs/adr/0004-cover-detection.md](docs/adr/0004-cover-detection.md) for the reasoning, and
[CONTRIBUTING.md](CONTRIBUTING.md) for how to add a parser for a naming convention Mangabind
doesn't yet recognize.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
