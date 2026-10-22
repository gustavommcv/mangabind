# 27. A volume folder that holds chapters gives them its volume

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked that a layout of volume folders with chapters in them be read
and accepted if manga is kept that way. It becomes effective with the pull request that carries it. It settles what
[ADR 0024](0024-a-volume-named-folder-is-a-volume-of-one-chapter.md) left open, and it reads units by their path, not
their name.

## Context

A manga can be kept with a folder for each volume and the chapters inside it: `Series/第1巻/第1話`,
`Series/Volume 2/Chapter 5.cbz`. The chapters in such a layout often carry nothing but their own number, because the
folder says the rest, and their numbers often start again in every volume. mangabind read the entries directly in the
manga folder and nothing deeper, so a volume folder with no pages in it was reported as a folder with no pages, and
the chapters inside it were never seen.

The evidence that the layout exists is partial, and this record does not claim more. The downloader of MangaDex
that mangabind was made around has a format of folders of volumes, `Volume. 1/`, `No Volume/`, with the images
directly in them (its `raw-volume`), which ADR 0024 and 0026 read. Kavita nests series in publisher folders and puts
`Specials` in a subfolder of a series. Neither documents chapters in volume folders. It is how a person who sorts
a collection by hand arranges it, and the cost of reading it is small when the folder is named for a volume
alone and holds no pages, since both have to be true.

## Decision

- **A folder named for a volume alone (ADR 0024, 0026), with no pages of its own and chapter folders or archives
  in it, is a volume folder.** The chapters in it are read as the entries of the manga folder are, and a chapter whose
  name carries no volume has the folder's. Chapter numbers need not be unique across volumes. Only one level is read.
- **A volume in the name of the chapter wins**, as it wins over the metadata file. When it is another volume than the
  folder's there is a warning, `volume_folder_conflict`, and the chapter keeps its own. The metadata file is
  consulted after the folder, and the folder's volume wins over it as a name's does.
- **A folder with pages is the volume itself** (ADR 0024), whatever else is in it; the folders inside it are not read,
  as in any chapter folder. **A volume folder with nothing in it that can be a chapter** is read as any folder, and
  reported as having no pages. **A folder that is not named for a volume** (`Specials`, `Extras`) is not a volume
  folder, and is reported as a name that cannot be parsed, as before.
- **The report says where a chapter was found.** A unit found in a volume folder has `volume_folder`, the absolute
  path of the folder, a new optional field of the unit; its `parser.volume` stays what the name said and its
  `effective_volume` is the volume it has. The volume folder itself is not a unit. This is within protocol version 1,
  which allows fields and codes to be added.
- **A chapter is named by its path below the manga folder in the messages** when it is in a volume folder
  (`第1巻/第1話`), since two volume folders may hold chapters of one name. A chapter directly in the manga folder is
  named as before.
- **Units are matched by their path, not their name.** Two chapters of one name in two volume folders would be one
  unit to a match by name. The same change corrects a fault that was already there: an archive has a name without its
  extension, and the conflict or the lack of a volume of a `.cbz` or `.zip` was looked for under the name with it, so a
  conflicting archive was reported as `included`.

## Consequences

A manga kept with a folder for each volume and the chapters in it is bound into the books of its volumes, in order,
with chapter numbers that start again in each volume, and the report shows which folder each chapter came from. A
manga kept in one folder with no volumes is as before. A layout deeper than one level, or one folder of chapters not
named for a volume, is still reported as a name that cannot be parsed. The report of an archive in conflict now says
`conflict`, as that of a folder always did.
