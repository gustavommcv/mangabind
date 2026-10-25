# Mangabind machine protocol version 1

This document is the normative contract for Mangabind's machine-readable interface.

## Compatibility handshake

```text
mangabind --protocol-version
```

The command exits 0 and writes one JSON object to stdout:

```json
{
  "protocol_version": 1,
  "tool": "mangabind",
  "tool_version": "0.7.0",
  "capabilities": ["report", "progress-json", "chapter-versions", "chapter-labels", "unit-first-page", "no-volumes"]
}
```

`tool_version` is the release's own version (`0.7.0` here is an example), and `dev` for an unversioned local build. Manga2Ink must require an exact supported
`protocol_version` and independently verify the pinned release version and executable checksum.

`capabilities` lists optional features a consumer may rely on: `report` is the versioned JSON report
this document describes, and `progress-json` is the opt-in progress side channel described below.
Releases before 0.6.0 advertise only `report`; a consumer should check for the capability, not
compare release versions. The others are:

- `chapter-versions`: the mapping file's `chapter_versions` choices, the unit disposition `not_chosen`, the
  manga report's `chapter_versions` and the `chapter_choice_unmatched` issue
  ([ADR 0028](adr/0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md))
- `chapter-labels`: the `-chapter-labels` flag and the volume's `chapter_entries`
  ([ADR 0029](adr/0029-chapters-can-be-labelled-by-their-number.md))
- `unit-first-page`: the unit's `first_page`
- `no-volumes`: the `-no-volumes` flag and the manga report's `without_volumes`
  ([ADR 0030](adr/0030-a-series-without-volumes-is-one-book-of-chapters.md))

## Exit codes

The exit code and the report agree, and a consumer should read both: the report says what was done,
the exit code says whether anything was not.

| Exit code | Meaning |
|---|---|
| 0 | The run completed. Its `status` is `completed` or `completed_with_warnings`; nothing to bind is a warning, not a failure. |
| 1 | Something could not be done: the report's `status` is `failed`, and its error issues say what. |
| 2 | The command line could not be used (`invalid_arguments`, `missing_input`, `metadata_file_with_batch`, `no_volumes_with_combine`, `--progress-json` without `--json`). |
| 130 | The run was interrupted by Ctrl-C or `SIGTERM` (`interrupted`). |

When machine output was asked for, the report is on stdout for every one of these that the process
could write, and it is the only thing there.

## Planning and execution

```text
mangabind --input <folder> --output <folder> --dry-run --json
mangabind --input <folder> --output <folder> --json
mangabind --input <library> --output <folder> --batch --dry-run --json
mangabind --input <folder> --output <folder> --combine --json
```

Stdout is exactly one JSON report. Planning mode reports `mode: "plan"` and all volume records have
`written: false`. Execution reports `mode: "execute"`; `written` becomes true only after that CBZ
was successfully closed. Human output is unchanged when `--json` is absent.

### Optional progress side channel

`--progress-json` requires `--json`. It leaves the single stdout report unchanged and emits one
JSON object per line on stderr while work is underway. For example, a standalone script can run
`mangabind --input <folder> --output <folder> --json --progress-json >report.json 2>progress.jsonl`.
The handshake advertises `"progress-json"` alongside `"report"` when this mode is available.

Every progress event contains `protocol_version: 1`, `tool: "mangabind"`, `tool_version`,
`kind: "progress"`, `stage`, `state`, and `manga`. `stage` is `"inspect"` or `"write"`.
Inspection has `"started"` and `"completed"` states. Write events use `"started"`, `"advanced"`,
and `"completed"` and include the 1-based `volume_index`, `volume_count`, numeric `volume_number`,
`completed_pages`, and `total_pages`. The page counters refer to this manga, even in batch mode;
they reset for the next manga. `advanced` means a page was copied into the archive, not that the
archive was closed. Only `completed` follows a successful close. In `--combine`, progress identifies
the source volume currently being copied into the single output archive. `--dry-run` emits no write
events. The final stdout report and exit code, not progress, determine whether the command succeeded. When a run is interrupted, no `completed` event follows for the volume that was being written. When a volume fails to write, the counters keep the pages that were copied into it and go on with the next volume: they never go back.

With `--progress-json`, stderr is reserved for progress during normal execution. A failure to
serialize the final report may still produce a plain diagnostic there. Consumers must treat that
case as a failed invocation, not as a progress event. Without `--progress-json`, the existing
stderr behavior is unchanged.

A command line that cannot be used still gets a report: when `--json` or `--protocol-version` appears
anywhere on it, in any spelling the flag package reads (`-json`, `--json`, `--json=true`), stdout holds
one report with an `invalid_arguments` issue and the exit code is 2. Nothing is written to stderr then,
so the stream stays free for progress.

The top-level object contains:

- `protocol_version`, `tool`, `tool_version`, and `kind` (`"report"`)
- `mode`: `"plan"` or `"execute"`
- `status`: `"completed"`, `"completed_with_warnings"`, or `"failed"`
- absolute `input_path` and `output_path`
- `batch`: whether library-batch semantics were used
- `manga`: ordered per-manga reports; one entry for a non-batch invocation
- `issues`: invocation-level issues, such as an unreadable library folder
- `summary`: manga, volume, page, warning, and error totals

Every manga report contains its name, absolute input and optional metadata paths, status, ordered
`units`, ordered intended `volumes`, structured `issues`, and totals. When `-combine` was used,
`combined_output_path` is also present (see "Volumes" below) - see
[ADR 0012](adr/0012-combine-series-into-one-volume.md).

### Units

Every discovered chapter folder or chapter CBZ has one unit record:

- `kind`: `"folder"` or `"cbz"` (an archive of images: a `.cbz` or a `.zip`, which are the same format)
- `parser`: whether a parser matched, its stable parser name, and the original parsed volume,
  chapter, special suffix, title, group, and language when present
- `metadata_assignment.status`: `"not_applicable"`, `"not_requested"`, `"not_found"`, `"applied"`,
  `"confirmed"`, or `"ignored_conflict"`; `metadata_volume` is present when metadata matched
- `effective_volume`: the assignment used for grouping, when one exists
- `volume_folder`: the absolute path of the volume folder the unit was found in, when it was found in one
  (`<manga>/第1巻/第2話`); a folder named for a volume alone holds the chapters in it, and a chapter whose name
  carries no volume takes the folder's as its `effective_volume`
  ([ADR 0027](adr/0027-volume-folders-hold-chapters.md))
- `page_count`: the images in the unit. A page is an image by its name (`.jpg`, `.jpeg`, `.png`,
  `.gif`, `.webp`, `.bmp`, `.avif`, `.jxl`, `.tif`, `.tiff`); see
  [ADR 0016](adr/0016-a-page-is-an-image.md)
- `first_page`: the first image of the unit in reading order, as the name of a file in a folder or of an entry of an
  archive; absent for a unit with no pages
- `disposition`: `"included"`, `"unassigned"`, `"unparsed"`, `"empty"`, `"conflict"`, `"not_chosen"`, or `"failed"`.
  `"not_chosen"` is a unit that claims a chapter several units claim, which the mapping file's `chapter_versions` chose
  against ([ADR 0028](adr/0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md))

The parser's original `volume` is never rewritten when metadata fills a missing assignment;
`effective_volume` records the result. This distinction lets a consumer show exactly where an
assignment came from.

### Chapter versions

The manga report's `chapter_versions` has one entry for every chapter that more than one unit claims, whether or not
a volume would make them a conflict (an empty list when there is none):

- `chapter` and `special`: the chapter number and its special suffix (`x1`), when there is one
- `volume`: the effective volume of the units, when they have one. The units of a group share their chapter, special
  suffix and effective volume
- `units`: the absolute paths of the units in the group, in the order they were found, chosen or not
- `resolution`: `"chosen"` when the mapping file chose among them (`chapter_versions` in the file), and
  `"unresolved"` when it did not

The mapping file chooses with `"chapter_versions": [{"chapter": "6", "use": ["Ch.0006 (en) [A].cbz"]}]`: one unit is
the choice of a version, several are all kept as alternatives, none leaves the chapter out. A unit is named by its
path below the manga folder, written with `/`. A chapter that one unit claims is not decided by a choice. See
[ADR 0028](adr/0028-the-versions-of-a-chapter-are-chosen-in-the-mapping-file.md).

### Volumes

Every intended output has its numeric volume, absolute output path, page count, ordered source unit
names, and `written` state. A failed write remains present with `written: false`.

When `-combine` was used, the manga report's `combined_output_path` is set instead of each volume
having its own written file: every volume record still describes its own `number`, `page_count`,
and `chapters`, but `output_path` points at that one combined file and `written` stays `false` for
each individual entry - the write succeeded or failed as a whole, reflected by whether
`combined_output_path`'s file actually exists and by the absence of a `combined_write_failed`
issue. See [ADR 0012](adr/0012-combine-series-into-one-volume.md).

When `-no-volumes` was used, the manga report has `without_volumes: true` and one volume: the whole book, numbered 1
because a volume has a number, written to `<manga>.cbz` (no volume in its name) and with `written` as for any volume.
Every unit has that volume as its `effective_volume` and `metadata_assignment.status` `not_applicable`; what its name
says stays in `parser.volume`. Issues about a chapter or a gap name no volume. See
[ADR 0030](adr/0030-a-series-without-volumes-is-one-book-of-chapters.md).

A volume's `chapter_entries` lists its chapters in reading order, each with the `label` the chapter has in the book
(the name of its directory in the volume, which the converter shows in the table of contents), the absolute `path`
of the unit it came from, and its `page_count`. The label is `c003 - Title` by default, and `Ch.6 - Title` with
`-chapter-labels number`; a chapter kept as one of several alternatives has ` (version 1 of 2)` after its title. See
[ADR 0029](adr/0029-chapters-can-be-labelled-by-their-number.md).

### Issues

Issues always contain:

- `tool`: `"mangabind"`
- `severity`: `"warning"` or `"error"`
- stable `code` and `stage`
- `recoverable`
- a user-facing `message`

Optional context is `manga`, `volume`, `chapter`, `special`, `path`, and `related_paths`.
`diagnostic` may contain technical detail for an explicitly expanded diagnostics view. Consumers
must branch on `code`, not English message text.

When no chapter units are found, `no_chapters_found` accompanies any inspection warnings about
unsupported files or skipped links; it does not replace them. The run still exits 0 with
`status: "completed_with_warnings"`, and produces no volumes.

Version 1 codes are:

| Code | Stage | Meaning |
|---|---|---|
| `invalid_arguments` | `configuration` | A flag could not be parsed, or an argument that is not a flag was left over |
| `missing_input` | `configuration` | No input was provided |
| `metadata_file_with_batch` | `configuration` | One shared metadata override was requested for batch mode |
| `no_volumes_with_combine` | `configuration` | `-no-volumes` and `-combine` were asked for together |
| `library_scan_failed` | `inspect` | The batch root could not be read |
| `input_scan_failed` | `inspect` | A manga input could not be read |
| `unsupported_input_file` | `inspect` | A sibling file is not a supported chapter unit |
| `unsupported_page_files` | `inspect` | A chapter holds files that are not images, apart from known junk (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `ComicInfo.xml`, `__MACOSX/` and `._` files); they were left out. One issue per chapter: `path` is the chapter, `chapter` and `volume` say which, and `related_paths` lists the files of a folder (for a `.cbz`, `diagnostic` names its entries) |
| `page_entries_too_large` | `inspect` | A `.cbz` chapter has image entries that declare they expand to more than a page can be (over 256 MiB, or over 1000 times their stored size and over 16 MiB); they were left out ([ADR 0018](adr/0018-an-entry-cannot-expand-to-more-than-a-page-can-be.md)). One issue per chapter: `path` is the archive, `chapter` and `volume` say which, and `diagnostic` lists the entries with their sizes (`name (bytes from stored bytes)`) |
| `link_skipped` | `inspect` | A symbolic link was not followed because it does not lead to a file inside the input folder (see [ADR 0014](adr/0014-links-stay-inside-the-input.md)); `path` is the link, and `chapter` (and `volume`, when the chapter has one) is set for a page link With `--batch`, a link to a manga folder in the library is reported too, as an invocation-level issue in the top-level `issues` (`path` is the link, and there is no manga, chapter or volume); it is not followed |
| `no_chapters_found` | `inspect` | No folder or CBZ chapter units were found |
| `page_listing_failed` | `inspect` | A unit's pages could not be listed |
| `empty_chapter` | `inspect` | A recognized unit contains no pages |
| `unparsed_chapter` | `parse` | No registered parser recognized a unit name |
| `volume_folder_conflict` | `inspect` | A chapter in the folder of one volume has a name that says another; the name wins. `volume` is the one from the name, `path` the chapter |
| `metadata_load_failed` | `metadata` | The mapping file could not be read or validated: a volume number that is not a number from 0 to 100000, a chapter range that is too large, more than 100,000 chapters listed, or a `chapter_versions` choice that is not about one chapter or is given twice for a chapter |
| `metadata_duplicate_chapter` | `metadata` | The mapping file lists chapters under more than one volume; the last listing applies. One issue per file: `path` is the file, and the message names the first of them |
| `metadata_volume_conflict` | `metadata` | Filename and metadata volumes disagree; filename wins |
| `chapter_choice_unmatched` | `metadata` | The mapping file chooses files for a chapter, and none of them is a unit that claims it; the choice was not used and the units are in conflict. `chapter` and `volume` say which, and `related_paths` lists the units that claim it |
| `unassigned_chapter` | `group` | A parsed chapter has no volume assignment |
| `chapter_gap` | `group` | Whole-number chapters are missing inside a volume |
| `chapter_conflict` | `group` | Multiple sources claim the same volume/chapter key |
| `no_volumes_produced` | `group` | No intended output survived grouping |
| `volume_write_failed` | `write` | An intended volume could not be written; the volumes after it are still attempted, and the exit code is 1 |
| `interrupted` | `inspect` or `write` | The run was stopped by Ctrl-C or SIGTERM. The volume that was being written was removed, volumes already finished stay, the volumes not written are in the report with `written: false`, and the exit code is 130. In `-batch`, the manga that were left are not started (an invocation-level issue says so) |
| `combined_write_failed` | `write` | A `-combine` run's single combined file could not be written |

## Evolution rule

Consumers must ignore unknown object fields and unknown issue codes. Mangabind may add fields or
codes within protocol version 1. Removing a field, changing its JSON type, changing a documented
value's meaning, or changing framing requires a new protocol version and a new ADR.
