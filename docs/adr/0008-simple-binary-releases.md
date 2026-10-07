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

## Amendment (2026-10-07): the scripts check what they download

An audit ([`docs/audit-2026-10.md`](../audit-2026-10.md), finding 15) found that the install scripts, the thing most people run first, downloaded the latest archive and put it on the PATH without checking it against the release's `checksums.txt` (published for exactly that), could not install a given version, found the tag by running `grep` over the GitHub API's JSON (which also made the first command depend on an unauthenticated API with a rate limit), and ran whatever part of the script had arrived if the download was cut short. The decision stands: raw binaries, two scripts, nothing else to install. What changed inside it:

- **The archive's SHA-256 must be the one in `checksums.txt`**, or nothing is installed: a mismatch, a `checksums.txt` that is missing or does not list the archive, and a system with no tool to compute the hash (`sha256sum`, `shasum` or `openssl`; PowerShell has `Get-FileHash`) all stop the script, with a message that says which. This guards against a damaged or corrupted download and against a mirror or a proxy that serves something else. It does not guard against the release itself being replaced, since `checksums.txt` comes from the same place as the archive; signing the release is a separate decision that has not been made.
- **A version can be chosen**: as an argument (`sh -s -- v0.7.0`) or in `MANGABIND_VERSION` (with or without the `v`). Without one the script fetches `releases/latest/download/<archive>`, an address GitHub keeps for exactly that, so nothing asks the API for a tag. `MANGABIND_INSTALL_DIR` chooses the folder, `MANGABIND_BASE_URL` points at a mirror (and is what the tests use), and `MANGABIND_NO_MODIFY_PATH` keeps the Windows script from editing the user's PATH.
- **The program is put in place with a rename** inside its folder, so that a failure never leaves half a program or overwrites a running one in place, and the script runs the program with `--version` to say what was installed and that it runs on this system.
- **Everything is inside one function called on the last line**, so a download cut short is a syntax error and not half an install (the usual way `curl | sh` goes wrong).
- **The scripts are tested** (`scripts/test-installers.sh`, `scripts/test-installers.ps1`, and the Installers workflow): against a release made on the spot with a real program, including the refusals; for the version logic, against a stand-in for `curl` and for `Invoke-WebRequest` that writes down the address it was asked for; and, weekly and when a script changes, against the real latest release on all three systems.

