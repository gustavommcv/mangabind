# 14. Follow a link only when it leads to a file inside the input

Date: 2026-10-04

## Status

Accepted; amended 2026-10-07 (a library's links - see the end).

## Context

A chapter folder is read with `os.ReadDir`, and every entry that is not a folder is taken for a page. A symbolic link to a file is such an entry, and opening it follows the link: the bytes of the target are copied into the volume, whatever the target is. mangabind goes by the name of the page, not by what is in it. An independent audit of Mangabound and its tools reproduced it with the released binary: a folder `Hostile/Vol.01 Ch.001/` with `001.png` and two links, one to an image and one to a text file outside the folder, produced a `.cbz` that held both, with exit code 0 and no warning in the report.

The folder is what the person chose to bind. A link in it that leads elsewhere puts into a book a file they never chose, and the book is then saved, shared over OPDS or uploaded. The people most exposed are those who extract a manga archive made by someone else: `tar` and `zip` extractors on Linux and macOS recreate links. Mangabound's own OPDS server already refuses to serve a link that leads outside a shared folder; this is the same rule at the other door.

Some links are legitimate. Someone who seeds a download from one place and keeps the library as links into it has links to files outside the manga folder, on purpose; and a chapter that shares a credits page with another has a link inside it.

## Decision

- A symbolic link is followed only when its real path, with every link on the way resolved, is a regular file inside the manga's own input folder. The input folder is the one given to `-input`, or, with `-batch`, the folder of each manga.
- This applies to a page of a chapter folder, and to a `.cbz` chapter that is a link. A link to a folder, or to nothing at all, is skipped too: the first could never be copied as a page, and for the second nothing can be said of where it leads.
- A link that is not followed is not an error. It is left out of its chapter, and reported as a warning with the code `link_skipped` at the `inspect` stage, with the link's own path in `path` and, for a page, the chapter it was found in and that chapter's volume when it has one. The message says why: it leads outside the input folder, it does not lead to a file, or the input folder could not be resolved to compare with. A chapter left with no pages is reported as an empty chapter, as before. The warning is in the plan (`--dry-run`) as well as in the run, and the page counts leave the link out.
- The comparison is between real paths, so a link to a link, and a link through a folder that is itself a link, are judged by where they end up. A manga folder given by a shortcut is compared by its real path too.
- Skipping is the whole mitigation: nothing follows the link's target for any other purpose, and nothing about the target (its name, its size) is put in the report.
- At the root of the manga, a link that is not a `.cbz` is as it was: an unrecognized file, skipped, whether it leads to a file or to a folder.
- `link_skipped` is a new code within protocol version 1, which ADR 0011 allows: consumers must ignore codes they do not know, and one that does not know it still sees an ordinary warning with a message.

## Alternatives considered

- **Skip every link.** Simpler, and no comparison to get wrong. It would also break the people who keep their library as links to a seeded download, with no way for them to say that they mean it. A flag to allow them would be one more thing for a frontend to expose, for something that the containment rule already answers.
- **Leave it to the caller.** A frontend can look for links before it runs mangabind, as Mangabound could. But every caller would need to do it, and mangabind is the one that opens the files: it is the one place that cannot be bypassed.
- **Check again as each page is copied.** This would narrow the time between the check and the copy. It would also need the real path of every page to be resolved twice, for a gain only against someone who can write to the folder while it is being bound, who has other ways to put a file in it.

## Consequences

A book made from a folder with a link that leads outside lacks that page, and says so. For the person who meant it, the fix is to put a copy in the folder.

Not covered: a hard link to a file elsewhere is the same file under another name and cannot be told from one; a Windows junction or other mount point is not a symbolic link to Go and is not checked, but it can only lead to a folder, which cannot be copied as a page; and a link swapped for another between the folder being read and its pages being copied is not caught. All three need someone who already controls the folder.

The checks run for a link only. A folder without links costs nothing more than before.

Tests make real links, and skip where the system does not allow one (Windows without Developer Mode or an elevated shell); the Linux and macOS runs of CI make them.

## Amendment (2026-10-07): a library's links

The decision said the input folder is "the one given to `-input`, or, with `-batch`, the folder of each manga", and so covered what is inside a manga. It left out the entries of the library folder itself: a link there to a manga folder was skipped without a word, because only real folders were taken for manga and nothing said that a link had been passed over (finding 9 of [the October 2026 audit](../audit-2026-10.md)). Following it would put whatever folder it points to into a volume, which is what this ADR exists to prevent.

- With `-batch`, a link in the library folder that leads to a folder is **not followed and not a manga of the run**, and a link that leads nowhere is reported too. Both are `link_skipped` warnings at the `inspect` stage, with the link's own path: on the terminal as `warning: found a link "Name" that leads to a folder, and the manga folders of a library are not followed through links, skipped`; in the machine report as an invocation-level issue (there is no manga to attach it to).
- A link to a file in the library folder is as a file is: not a manga, and not worth a word.
- Someone who keeps a library as links into a download folder puts the folders there, or runs mangabind on each target; the README says so for links in general.

