# 8. Ship raw binaries with a one-line install script, not a package manager

Date: 2026-09-11

## Status

Accepted

## Context

A user testing Mangabind needed the Go toolchain installed just to run `go install ...@latest`,
and wanted `mangabind` callable from any terminal without manually moving a `.exe` into a folder
on PATH.

The first idea considered was publishing to Homebrew (macOS/Linux) and Scoop (Windows) - both
have first-class support in GoReleaser (already the planned release tool per ADR 0002), and both
solve "install and it's just on PATH." But neither is actually installed by default: a user
without Homebrew would need to install Homebrew first, and a user without Scoop would need to
install Scoop first - which doesn't remove a prerequisite, it just swaps "install Go" for "install
a package manager." It would also mean maintaining two extra auxiliary repositories (a Homebrew
tap, a Scoop bucket) and a stored access token, ongoing infrastructure for a project still in
early development with no evidence yet that this specific gap (package-manager installability)
matters to more than one person. That's designing for hypothetical scale rather than the actual
problem.

## Decision

Publish cross-platform binaries via GitHub Releases (GoReleaser, `.goreleaser.yml`, triggered by
pushing a `v*` tag - see `.github/workflows/release.yml`), plus two small install scripts
committed to the repo itself:

- `install.sh` for macOS/Linux (`curl ... | sh`): downloads the right archive for the running
  OS/arch, extracts it, and drops the binary in `~/.local/bin`.
- `install.ps1` for Windows (`irm ... | iex`): downloads the `.zip`, extracts it to
  `%LOCALAPPDATA%\Programs\mangabind`, and adds that folder to the user's PATH if it isn't there
  already.

Both use only what already ships with the OS (`curl`/`tar`/`sh` on macOS/Linux, PowerShell on
Windows) - no new prerequisite to install first, no external repository to maintain, no secret
beyond the `GITHUB_TOKEN` GitHub Actions already provides for free.

## Consequences

One command gets `mangabind` on PATH on any of the three target platforms, which is what actually
mattered here. Publishing to Homebrew/Scoop/winget remains a reasonable later addition if the
project grows enough that package-manager installability is a real, demonstrated want - not a
door we're closing, just not opening before there's a reason to.
