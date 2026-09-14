# 11. Add a versioned JSON planning and execution report

Date: 2026-09-14

## Status

Accepted; supersedes ADR 0009.

## Context

ADR 0009 rejected machine-readable output because there was no demonstrated consumer and a second
output format would have been premature. Mangabound is now that concrete consumer. It must inspect
every local chapter unit, present and persist chapter-to-volume assignments, explain conflicts and
gaps, and execute a pinned Mangabind binary without scraping prose written for a terminal.

The other conclusions in ADR 0009 remain sound: human-readable output follows normal CLI stream and
exit-code conventions; batch mode is explicit and isolates failures; dry-run performs the real plan
without writing; quiet mode suppresses only routine prose. Replacing those behaviors would break
existing users for no benefit.

## Decision

- Add `--json`. Without it, human-readable behavior is unchanged. With it, stdout contains exactly
  one JSON report and no human progress text.
- `--json --dry-run` is planning mode and never writes a CBZ. `--json` without `--dry-run` executes
  the same plan and records whether each intended volume was written.
- Add `--protocol-version`, which needs no input and returns a small JSON handshake containing the
  independent machine-protocol version, executable version, tool name, and capabilities.
- Version 1 reports every discovered chapter unit, the parser that matched and its original fields,
  metadata lookup status and applied value, effective assignment, page count, final disposition,
  intended volume output, warnings, errors, and aggregate totals. Batch output contains one report
  per manga and preserves per-manga failure isolation.
- Structured issues carry stable `tool`, `severity`, `code`, `stage`, context, `recoverable`, and
  human-readable `message` fields. `diagnostic` is optional technical detail and is not the primary
  user message.
- Ordering is deterministic: scanned units retain natural scanner order, volumes use numeric order,
  chapters inside volume records use final chapter order, and issues follow the pipeline stage that
  discovered them. Paths are absolute. No timestamps or durations appear in the report.
- Exit codes remain 0 for success, 1 for runtime failure, and 2 for invalid usage. When machine mode
  was requested, a non-zero exit still produces a parseable report on stdout whenever the process
  can do so.
- The protocol version increases only for a breaking change. Consumers must ignore unknown fields,
  so additive fields do not require a version increase. Fields cannot be removed, retyped, or given
  incompatible meaning within version 1.

The normative field and value definitions live in
[`docs/machine-protocol-v1.md`](../machine-protocol-v1.md).

## Consequences

Mangabound can inspect and execute Mangabind through an explicit, pinned contract instead of parsing
terminal prose. The protocol becomes release surface and therefore requires real-fixture regression
tests. Human users pay no compatibility cost unless they explicitly opt into machine mode.

Maintaining JSON is now justified work. Any future reversal or breaking protocol change requires a
new ADR rather than silently changing version 1.
