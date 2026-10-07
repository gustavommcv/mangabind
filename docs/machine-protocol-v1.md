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
  "tool_version": "0.6.0",
  "capabilities": ["report", "progress-json"]
}
```

`tool_version` is `dev` for an unversioned local build. Mangabound must require an exact supported
`protocol_version` and independently verify the pinned release version and executable checksum.

`capabilities` lists optional features a consumer may rely on: `report` is the versioned JSON report
this document describes, and `progress-json` is the opt-in progress side channel described below.
Releases before 0.6.0 advertise only `report`; a consumer should check for the capability, not
compare release versions.

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

- `kind`: `"folder"` or `"cbz"`
- `parser`: whether a parser matched, its stable parser name, and the original parsed volume,
  chapter, special suffix, title, group, and language when present
- `metadata_assignment.status`: `"not_applicable"`, `"not_requested"`, `"not_found"`, `"applied"`,
  `"confirmed"`, or `"ignored_conflict"`; `metadata_volume` is present when metadata matched
- `effective_volume`: the assignment used for grouping, when one exists
- `page_count`: the images in the unit. A page is an image by its name (`.jpg`, `.jpeg`, `.png`,
  `.gif`, `.webp`, `.bmp`, `.avif`, `.jxl`, `.tif`, `.tiff`); see
  [ADR 0016](adr/0016-a-page-is-an-image.md)
- `disposition`: `"included"`, `"unassigned"`, `"unparsed"`, `"empty"`, `"conflict"`, or `"failed"`

The parser's original `volume` is never rewritten when metadata fills a missing assignment;
`effective_volume` records the result. This distinction lets a consumer show exactly where an
assignment came from.

### Volumes

Every intended output has its numeric volume, absolute output path, page count, ordered source unit
names, and `written` state. A failed write remains present with `written: false`.

When `-combine` was used, the manga report's `combined_output_path` is set instead of each volume
having its own written file: every volume record still describes its own `number`, `page_count`,
and `chapters`, but `output_path` points at that one combined file and `written` stays `false` for
each individual entry - the write succeeded or failed as a whole, reflected by whether
`combined_output_path`'s file actually exists and by the absence of a `combined_write_failed`
issue. See [ADR 0012](adr/0012-combine-series-into-one-volume.md).

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

Version 1 codes are:

| Code | Stage | Meaning |
|---|---|---|
| `invalid_arguments` | `configuration` | A flag could not be parsed, or an argument that is not a flag was left over |
| `missing_input` | `configuration` | No input was provided |
| `metadata_file_with_batch` | `configuration` | One shared metadata override was requested for batch mode |
| `library_scan_failed` | `inspect` | The batch root could not be read |
| `input_scan_failed` | `inspect` | A manga input could not be read |
| `unsupported_input_file` | `inspect` | A sibling file is not a supported chapter unit |
| `unsupported_page_files` | `inspect` | A chapter holds files that are not images, apart from known junk (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `ComicInfo.xml`, `__MACOSX/` and `._` files); they were left out. One issue per chapter: `path` is the chapter, `chapter` and `volume` say which, and `related_paths` lists the files of a folder (for a `.cbz`, `diagnostic` names its entries) |
| `link_skipped` | `inspect` | A symbolic link was not followed because it does not lead to a file inside the input folder (see [ADR 0014](adr/0014-links-stay-inside-the-input.md)); `path` is the link, and `chapter` (and `volume`, when the chapter has one) is set for a page link With `--batch`, a link to a manga folder in the library is reported too, as an invocation-level issue in the top-level `issues` (`path` is the link, and there is no manga, chapter or volume); it is not followed |
| `no_chapters_found` | `inspect` | No folder or CBZ chapter units were found |
| `page_listing_failed` | `inspect` | A unit's pages could not be listed |
| `empty_chapter` | `inspect` | A recognized unit contains no pages |
| `unparsed_chapter` | `parse` | No registered parser recognized a unit name |
| `metadata_load_failed` | `metadata` | The mapping file could not be read or validated: a volume number that is not a number from 0 to 100000, a chapter range that is too large, or more than 100,000 chapters listed |
| `metadata_duplicate_chapter` | `metadata` | The mapping file lists chapters under more than one volume; the last listing applies. One issue per file: `path` is the file, and the message names the first of them |
| `metadata_volume_conflict` | `metadata` | Filename and metadata volumes disagree; filename wins |
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
