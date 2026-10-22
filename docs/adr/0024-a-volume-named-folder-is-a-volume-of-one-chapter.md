# 24. A folder or .cbz named for a volume is a volume of one chapter

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for the next steps of the plan for the names of Japanese
manga to be done on the best evidence found. It becomes effective with the pull request that carries it. It
settles what [ADR 0022](0022-japanese-volume-markers.md) left open.

## Context

A collection of Japanese manga is most often kept with one unit per volume, not per chapter. People who scan the
printed books ("自炊") end with a file or a folder of pages for each volume, and the names in the tests of
Kavita's parser, which people reported, are of that kind: `(一般コミック) [奥浩哉] いぬやしき 第09巻`,
`諌山創] 進撃の巨人 第23巻`, `Test 5巻`. Kavita reads them as volumes, one file each.

mangabind binds chapters into volumes, and a name that carries a volume and no chapter had nowhere to go: it was
reported as a name that cannot be parsed, so a collection kept that way came out as no books at all, which is what
a person with such a collection first sees. The folder holds the pages of a whole volume, so the volume is the
unit and the question was how to say so without a new idea in the model.

Three ways were weighed:

- **Leave such names unread**, as before. The collection that is kept the commonest way is the one the tool can do
  nothing with.
- **A unit that is a volume in the model**, beside chapters. It is the exact description, and a change to the
  grouping, the report and the protocol that Mangabound reads, for a case whose result is a book with the same
  pages.
- **A volume with one chapter**, numbered 1, whose pages are all of the unit's. The grouping, the order of the
  volumes, the conflicts and the combined series already work for it, and the book is what the person has: one
  volume, its pages in order.

## Decision

- **A folder or a .cbz named `第N巻` or `N巻`, after a prefix if there is one (ADR 0023), is volume N with one
  chapter,** numbered 1 and without a title, by `JapaneseVolumeParser`. The groups in brackets that a collection puts
  after the volume (`(講談社)`, `[2020]`, `【分冊版】`) are allowed, since they say who published it and when, and no
  chapter. The volume may have a decimal part.
- **Two units of one volume conflict,** as two units of one chapter do (`volume 9 chapter 1 has 2 conflicting
  sources`): a folder and a .cbz of the same volume are one volume twice.
- **Not a volume:** a range (`Series 10-15巻`, `第1-3巻`), which is several volumes in one unit; the total of a set
  (`全10巻`), because the marker is stuck to the character before it; and a name with a title after the volume
  (`第1巻 おまけ`). They stay reported.
- **A volume folder that holds chapter folders is not read as its pages.** It has none of its own, and is
  reported as a folder with no pages, so that nothing is bound without them. Reading the folders of chapters
  inside a folder of a volume is a different layout and a different decision.
- **The English `Vol.01`, `v01` is not changed.** There is no evidence of collections kept that way here, and the
  rule can be extended to them if there is.

## Consequences

A collection kept with one folder or .cbz per volume is bound into the books of its volumes, in order, with the
pages in the order of ADR 0019. The table of contents of such a book has one entry, because the source has no
chapters to show; a person who wants one per chapter needs the chapters. The report, the names inside the books
and the protocol are as before, so this is within protocol version 1, and the parser's name, `japanese-volume`, is
new. What the scanner takes as a unit (a folder of images or a .cbz) is unchanged: a volume kept as a .zip or a
.rar is still reported as an unrecognized file.
