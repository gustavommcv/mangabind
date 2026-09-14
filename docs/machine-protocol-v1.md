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
  "tool_version": "0.4.0",
  "capabilities": ["report"]
}
```

`tool_version` is `dev` for an unversioned local build. Mangabound must require an exact supported
`protocol_version` and independently verify the pinned release version and executable checksum.

## Planning and execution

```text
mangabind --input <folder> --output <folder> --dry-run --json
mangabind --input <folder> --output <folder> --json
mangabind --input <library> --output <folder> --batch --dry-run --json
```

Stdout is exactly one JSON report. Planning mode reports `mode: "plan"` and all volume records have
`written: false`. Execution reports `mode: "execute"`; `written` becomes true only after that CBZ
was successfully closed. Human output is unchanged when `--json` is absent.

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
`units`, ordered intended `volumes`, structured `issues`, and totals.

### Units

Every discovered chapter folder or chapter CBZ has one unit record:

- `kind`: `"folder"` or `"cbz"`
- `parser`: whether a parser matched, its stable parser name, and the original parsed volume,
  chapter, special suffix, title, group, and language when present
- `metadata_assignment.status`: `"not_applicable"`, `"not_requested"`, `"not_found"`, `"applied"`,
  `"confirmed"`, or `"ignored_conflict"`; `metadata_volume` is present when metadata matched
- `effective_volume`: the assignment used for grouping, when one exists
- `page_count`
- `disposition`: `"included"`, `"unassigned"`, `"unparsed"`, `"empty"`, `"conflict"`, or `"failed"`

The parser's original `volume` is never rewritten when metadata fills a missing assignment;
`effective_volume` records the result. This distinction lets a consumer show exactly where an
assignment came from.

### Volumes

Every intended output has its numeric volume, absolute output path, page count, ordered source unit
names, and `written` state. A failed write remains present with `written: false`.

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
| `invalid_arguments` | `configuration` | A flag could not be parsed |
| `missing_input` | `configuration` | No input was provided |
| `metadata_file_with_batch` | `configuration` | One shared metadata override was requested for batch mode |
| `library_scan_failed` | `inspect` | The batch root could not be read |
| `input_scan_failed` | `inspect` | A manga input could not be read |
| `unsupported_input_file` | `inspect` | A sibling file is not a supported chapter unit |
| `no_chapters_found` | `inspect` | No folder or CBZ chapter units were found |
| `page_listing_failed` | `inspect` | A unit's pages could not be listed |
| `empty_chapter` | `inspect` | A recognized unit contains no pages |
| `unparsed_chapter` | `parse` | No registered parser recognized a unit name |
| `metadata_load_failed` | `metadata` | The mapping file could not be read or validated |
| `metadata_volume_conflict` | `metadata` | Filename and metadata volumes disagree; filename wins |
| `unassigned_chapter` | `group` | A parsed chapter has no volume assignment |
| `chapter_gap` | `group` | Whole-number chapters are missing inside a volume |
| `chapter_conflict` | `group` | Multiple sources claim the same volume/chapter key |
| `no_volumes_produced` | `group` | No intended output survived grouping |
| `volume_write_failed` | `write` | An intended volume could not be written |

## Evolution rule

Consumers must ignore unknown object fields and unknown issue codes. Mangabind may add fields or
codes within protocol version 1. Removing a field, changing its JSON type, changing a documented
value's meaning, or changing framing requires a new protocol version and a new ADR.
