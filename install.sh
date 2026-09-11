#!/bin/sh
# Downloads and installs the latest mangabind release for macOS/Linux.
# Usage: curl -fsSL https://raw.githubusercontent.com/gustavommcv/mangabind/main/install.sh | sh
set -e

repo="gustavommcv/mangabind"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux | darwin) ;;
  *)
    echo "mangabind: unsupported OS: $os" >&2
    exit 1
    ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "mangabind: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

version=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$version" ]; then
  echo "mangabind: could not determine the latest release" >&2
  exit 1
fi

archive="mangabind_${os}_${arch}.tar.gz"
url="https://github.com/$repo/releases/download/${version}/${archive}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading mangabind $version for $os/$arch..."
curl -fsSL "$url" -o "$tmp/$archive"
tar -xzf "$tmp/$archive" -C "$tmp"

install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"
mv "$tmp/mangabind" "$install_dir/mangabind"
chmod +x "$install_dir/mangabind"

echo "Installed to $install_dir/mangabind"

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
