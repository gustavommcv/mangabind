# 13. Add opt-in machine progress without changing the report

Date: 2026-09-29

## Status

Accepted.

## Context

ADR 0011 gives terminal users human-readable progress and machine consumers one final JSON report. That report correctly describes the outcome, but a script processing a large manga cannot tell whether the process is inspecting chapters or how far it has copied pages into the output archives. Mangabound is one such consumer, not the only possible one: batch scripts, status dashboards, and other frontends have the same need.

Parsing the human prose would make its wording a machine contract. Replacing stdout with an event stream would break version 1's one-report guarantee. Inventing progress from file sizes or elapsed time would be misleading.

## Decision

- Add `--progress-json`, valid only together with `--json`. The normal terminal invocation and `--json` alone retain their exact behavior.
- Keep stdout as exactly one version 1 report. With the new flag, stderr is a newline-delimited stream of versioned JSON progress events. A terminal script can redirect the channels independently; neither event wording nor a GUI is needed to consume them.
- Advertise `progress-json` in the protocol handshake's capabilities. This is an additive, opt-in side channel, not a change to stdout framing or the meaning of an existing version 1 field. Document the event contract alongside the report contract.
- Emit inspection start/completion events, then actual page-copy progress while writing. Counts come from the same ordered pages used by the CBZ writer. A write-completed event is sent only after the archive closes successfully. In `--dry-run`, there are no write events. Batch progress names the current manga; `--combine` still reports the current source volume although it writes one archive.
- Progress is advisory. The final stdout report and exit code remain authoritative, including when a write fails. A broken stderr progress sink must not turn an otherwise successful archive into a failure.

## Consequences

CLI-first behavior is preserved: plain terminal users see no change, while any terminal automation can opt into machine progress. The extra output costs work only when requested. Fixtures must verify that stdout remains one report, event counts reflect real copied pages, dry-run never claims writes, and the existing human mode remains unchanged.
