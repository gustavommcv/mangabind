#!/bin/sh
# Downloads, checks and installs a mangabind release for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.sh | sh -s -- v0.6.1
#
# With no version it installs the latest release. Settings, all optional:
#   MANGABIND_VERSION      a release to install instead of the latest, as v0.6.1 (or 0.6.1)
#   MANGABIND_INSTALL_DIR  where to put it (default: ~/.local/bin)
#   MANGABIND_BASE_URL     a folder or URL holding the release files, for a mirror or a test
#
# The archive is only installed if its SHA-256 is the one in the release's
# checksums.txt. Everything is inside main(), which is called on the last line,
# so a download that is cut short does not run half a script.
set -eu

die() {
  echo "mangabind: $*" >&2
  exit 1
}

# sha256_of prints the SHA-256 of a file, with whichever tool the system has.
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$1" | awk '{print $NF}'
  else
    die "no tool to check the download with (sha256sum, shasum or openssl); install one, or download the archive from the releases page and check it against checksums.txt by hand"
  fi
}

main() {
  repo="gustavommcv/mangabind"
  version="${1:-${MANGABIND_VERSION:-}}"
  install_dir="${MANGABIND_INSTALL_DIR:-$HOME/.local/bin}"

  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    linux | darwin) ;;
    *) die "unsupported OS: $os" ;;
  esac

  arch=$(uname -m)
  case "$arch" in
    x86_64 | amd64) arch="amd64" ;;
    aarch64 | arm64) arch="arm64" ;;
    *) die "unsupported architecture: $arch" ;;
  esac

  # Where the release files are. GitHub serves "the latest release" at a fixed
  # address, so nothing needs to ask the API (and parse its JSON) for a tag.
  if [ -n "${MANGABIND_BASE_URL:-}" ]; then
    base="${MANGABIND_BASE_URL%/}"
    label="from $base"
  elif [ -n "$version" ]; then
    case "$version" in v*) ;; *) version="v$version" ;; esac
    base="https://github.com/$repo/releases/download/$version"
    label="$version"
  else
    base="https://github.com/$repo/releases/latest/download"
    label="the latest release"
  fi

  archive="mangabind_${os}_${arch}.tar.gz"
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  echo "Downloading mangabind ($label) for $os/$arch..."
  curl -fsSL "$base/$archive" -o "$tmp/$archive" ||
    die "could not download $base/$archive (is the version right? the releases are listed at https://github.com/$repo/releases)"
  curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" ||
    die "could not download $base/checksums.txt, so the download cannot be checked; nothing was installed"

  # The line of checksums.txt for this archive: "<sha256>  <name>".
  expected=$(awk -v name="$archive" '{ n = $2; sub(/^\*/, "", n); if (n == name) { print tolower($1); exit } }' "$tmp/checksums.txt")
  [ -n "$expected" ] || die "checksums.txt does not list $archive; nothing was installed"
  actual=$(sha256_of "$tmp/$archive")
  if [ "$actual" != "$expected" ]; then
    die "the checksum of $archive is $actual, but checksums.txt says $expected; the download is damaged or has been tampered with, and nothing was installed"
  fi

  mkdir "$tmp/unpacked"
  tar -xzf "$tmp/$archive" -C "$tmp/unpacked" || die "could not unpack $archive"
  [ -f "$tmp/unpacked/mangabind" ] || die "$archive does not hold a mangabind executable"

  # Moved into place in two steps, the last one a rename inside the folder, so
  # that a failure never leaves half a program (or a running one overwritten).
  mkdir -p "$install_dir"
  mv "$tmp/unpacked/mangabind" "$install_dir/.mangabind.$$"
  chmod 755 "$install_dir/.mangabind.$$"
  mv -f "$install_dir/.mangabind.$$" "$install_dir/mangabind"

  echo "Installed to $install_dir/mangabind"
  "$install_dir/mangabind" --version || die "the installed program does not run on this system"

  case ":$PATH:" in
    *":$install_dir:"*)
      echo "Done - try: mangabind --input <dir>"
      ;;
    *)
      echo ""
      echo "$install_dir isn't on your PATH yet. Add this to your shell profile (~/.bashrc, ~/.zshrc, ...):"
      echo "  export PATH=\"\$PATH:$install_dir\""
      ;;
  esac
}

main "$@"
