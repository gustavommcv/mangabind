# 20. Full-width digits and spaces in chapter names

Date: 2026-10-22

## Status

Accepted by the owner on 2026-10-22, when he asked for it to be done after a look at how manga comes to
people who read Japanese. It becomes effective with the pull request that carries it.

## Context

A Japanese input method types digits, the full stop and the space in their full-width forms: `０１２`,
`．`, and the ideographic space U+3000. A folder named `Vol.０１ Ch.００１` or `Vol.01　Ch.001` is the same
chapter as `Vol.01 Ch.001`, but every parser read digits and spaces in ASCII only, so such a folder was
reported as one that could not be parsed and left out of the books (mangabind 0.7.2 does that for each of
the four conventions). Kavita's parser has met the same names: one of its examples is a title followed by
U+3000 and `第1話`.

The order of a chapter's pages already counts the digits of every script as numbers
([ADR 0019](0019-pages-are-put-in-order-as-kcc-does.md)); the names of the chapters did not.

The text of a name matters too. The title of a chapter goes into the name of its folder inside the `.cbz`
([ADR 0006](0006-chapter-directories-for-kcc-toc.md)), which is what a reader's table of contents shows, so a
title written with full-width letters or punctuation has to come out as it was written.

## Decision

- **The parts of a name that say a number or separate words accept the full-width forms where they accept
  the ASCII ones:** the digits of a volume number, a chapter number and a special suffix (`x１`), the full stop
  of a decimal (`１２．５`), and the space. The patterns are still written with `\d` and `\s`; `widen` makes
  those accept both widths, and a number is folded to ASCII before it is read.
- **Nothing else is changed in a name.** A title, a group and a language are returned as they were written,
  full-width characters included. The words (`Vol`, `Chapter`, `Ch`), the dash, the colon, the parentheses and the
  brackets are not widened: a name written with them in full-width letters is not read, as before.
- **Not a normalization of the whole name.** NFKC would give the same digits, but it would also turn every
  full-width letter and punctuation mark of a title into an ASCII one, and the standard library has no NFKC
  (the project has no third-party dependency). Names decomposed by macOS (NFD) need nothing: no part of a name that
  is read has a decomposed form.

## Consequences

A folder named in full-width digits is grouped into its volume, and the same chapter in both widths (`Vol.01 Ch.002`
and `Vol.０１ Ch.００２`) is one chapter twice, so it is a conflict, as two copies in ASCII are. The folder names inside
the books, the report and the protocol are as before (the parsers keep their names), so this is within protocol
version 1. A name that was skipped as one that could not be parsed is now read: the books of a manga with such
names have the chapters they were missing.
