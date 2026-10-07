# 17. A volume is written whole or not at all, and Ctrl-C stops cleanly

Date: 2026-10-07

## Status

Accepted.

## Context

The `.cbz` writer did `os.Create(outPath)`, which truncates the destination at once, and then copied the pages in. An audit of the released binary ([`docs/audit-2026-10.md`](../audit-2026-10.md), finding 3) found what that does when a run does not reach the end, reproduced three ways:

- A good 480 MB volume existed and one page became unreadable. The run failed with exit code 1, and `Big - Vol.01.cbz` was now 312 MB: **a well-formed archive with the pages after the failure missing**, in the place of the good one, looking finished.
- Ctrl-C during the write left a 98 MB file with no central directory: not a valid archive, with the name of a finished volume, and no message.
- A write that failed with a simulated full disk left a 2 MB partial file. The error was reported, but the file stayed.

The one close that matters for a correct `.cbz` (`zip.Writer.Close`, which writes the central directory) was checked, but closing the file itself, where a network file system or a quota reports a write error held back until then, was not, and `.golangci.yml` excluded it from `errcheck` with a comment that said the important close was already covered.

Re-running a conversion over its previous output is the normal way to try again, so the failed re-run was taking the good copy with it.

## Decision

- **A volume is built under another name and moved into place when it is complete.** The file is created next to the destination as `.<name>.<random>.part` (in the same folder, so that moving it is a rename within one file system), is written, has its central directory written, is flushed to disk (`Sync`) and closed, **each step checked**, and only then is it renamed onto the destination. The destination is therefore always what it was before or the whole new volume, never part of one. An existing volume of that name is replaced, in one step, which is what re-running a conversion has always meant.
- **On any failure the part file is removed**, and the previous volume is untouched. The part file is created with the permissions an ordinary new file gets (not a temporary file's 0600), which is what the volume will have. Its name begins with a dot and ends in `.part`, so no later run takes it for a chapter and a file manager hides it.
- **Ctrl-C and SIGTERM stop the run cleanly.** The first one cancels the run: the page being copied is finished, no further page or volume is started, the part file is removed, volumes already finished stay, and the program exits with **130**. The terminal says what was done, once, in a line of its own (`Interrupted while writing volume 2; the unfinished file was removed.`), and does not report it as a failed write. A **second** Ctrl-C is not caught and ends the program at once, for the case where stopping is itself taking too long.
- **Machine output** reports an interruption as an issue with the code `interrupted` (stage `write`, or `inspect` when nothing was written yet; severity `error`, recoverable): in the manga's issues, with the volume's path when a file was being written, and for `-batch` also at the top level when the manga that were left were not started. The volumes not written are in the report with `written: false`, as after a failure, no `completed` progress event follows the volume that was interrupted, and the report is still the one document on stdout. The code is within protocol version 1.
- An interruption **ends a `-batch`** (the manga that are left are not tried), where an ordinary failure does not: the person asked for the run to stop.
- The exit code of an interruption is 130 whichever signal it was.
- The `errcheck` exclusion for closing a file is kept, because a deferred `Close` on a file that was only read is noise, but its comment no longer claims that the write path is covered by it: the write path now checks its closes itself.

## Consequences

A run that fails halfway no longer takes a good volume with it, and an interrupted run leaves nothing a reader or a later run could mistake for a book. The cost is disk space for a moment (the new volume exists beside the old one until the rename) and, for a hard kill (`SIGKILL`, the second Ctrl-C, a power cut), a leftover `.<name>.<random>.part` that can be deleted by hand; it is the only file such an end can leave, and the volume it was meant to become is not damaged. The page copy is not interruptible mid-page, so stopping takes as long as the largest page.

Renaming over an existing file is atomic on Linux and macOS, and on Windows `os.Rename` replaces the destination too; a destination that is open in another program makes the rename fail, and the run reports it as a failed volume.
