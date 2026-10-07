#!/bin/sh
# Tests install.sh against a release made on the spot, with no network: a real
# mangabind built here, packed the way a release packs it, with its
# checksums.txt, served from a folder through file://. Run it from anywhere:
#
#   sh scripts/test-installers.sh
#
# It needs Go (to build the program) and what install.sh needs (curl, tar).
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

failed=0
pass() { echo "ok    $1"; }
fail() {
  echo "FAIL  $1" >&2
  failed=1
}

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi
}

# A real program, in the archive a release would hold. The names are the
# installer's: GOOS and GOARCH are what it maps uname to.
(cd "$root" && go build -trimpath -ldflags "-X main.version=0.0.0-test" -o "$work/mangabind" ./cmd/mangabind)
os=$(go env GOOS)
arch=$(go env GOARCH)
archive="mangabind_${os}_${arch}.tar.gz"

release="$work/release"
mkdir "$release"
tar -czf "$release/$archive" -C "$work" mangabind
(cd "$release" && sha256 "$archive" >checksums.txt)

# install_from RELEASE_DIR INSTALL_DIR [env assignments...]: runs the installer
# the way a person does, from standard input, and keeps what it said.
install_from() {
  release_dir=$1
  install_dir=$2
  shift 2
  env MANGABIND_BASE_URL="file://$release_dir" MANGABIND_INSTALL_DIR="$install_dir" "$@" sh <"$root/install.sh" >"$work/out" 2>"$work/err"
}

# A damaged release: the same checksums.txt over another archive.
damaged="$work/damaged"
mkdir "$damaged"
cp "$release/checksums.txt" "$damaged/"
printf '#!/bin/sh\necho "not what was published"\n' >"$work/evil"
chmod +x "$work/evil"
tar -czf "$damaged/$archive" -C "$work" evil

# 1. The release installs, runs, and leaves nothing else in the folder.
if install_from "$release" "$work/i1" && [ "$("$work/i1/mangabind" --version)" = "mangabind 0.0.0-test" ] &&
  [ "$(ls -A "$work/i1")" = "mangabind" ] && [ -x "$work/i1/mangabind" ]; then
  pass "a release installs, runs, and leaves only the program"
else
  fail "a release installs, runs, and leaves only the program"
  cat "$work/out" "$work/err" >&2
fi

# 2. It can be run from a file too, and over an install that is there.
mkdir -p "$work/i2"
printf 'old' >"$work/i2/mangabind"
if env MANGABIND_BASE_URL="file://$release" MANGABIND_INSTALL_DIR="$work/i2" sh "$root/install.sh" >/dev/null 2>&1 &&
  [ "$("$work/i2/mangabind" --version)" = "mangabind 0.0.0-test" ]; then
  pass "it replaces an installed program"
else
  fail "it replaces an installed program"
fi

# 3. A download that is not what checksums.txt says is refused, and nothing is touched.
mkdir -p "$work/i3"
printf 'old' >"$work/i3/mangabind"
if install_from "$damaged" "$work/i3"; then
  fail "a damaged archive was installed"
elif grep -q "checksum" "$work/err" && grep -q "tampered" "$work/err" && [ "$(cat "$work/i3/mangabind")" = "old" ] && [ "$(ls -A "$work/i3")" = "mangabind" ]; then
  pass "a damaged archive is refused and what was installed is left alone"
else
  fail "a damaged archive is refused and what was installed is left alone"
  cat "$work/err" >&2
fi

# 4. A checksums.txt that does not list the archive.
nolist="$work/nolist"
mkdir "$nolist"
cp "$release/$archive" "$nolist/"
echo "0000000000000000000000000000000000000000000000000000000000000000  mangabind_plan9_mips.tar.gz" >"$nolist/checksums.txt"
if install_from "$nolist" "$work/i4"; then
  fail "an archive that checksums.txt does not list was installed"
elif grep -q "does not list" "$work/err" && [ ! -e "$work/i4/mangabind" ]; then
  pass "an archive that checksums.txt does not list is refused"
else
  fail "an archive that checksums.txt does not list is refused"
  cat "$work/err" >&2
fi

# 5. No checksums.txt at all: nothing can be checked, so nothing is installed.
nosums="$work/nosums"
mkdir "$nosums"
cp "$release/$archive" "$nosums/"
if install_from "$nosums" "$work/i5"; then
  fail "an archive with no checksums.txt was installed"
elif grep -q "cannot be checked" "$work/err" && [ ! -e "$work/i5/mangabind" ]; then
  pass "a release with no checksums.txt is refused"
else
  fail "a release with no checksums.txt is refused"
  cat "$work/err" >&2
fi

# 6. No archive for this system.
noarchive="$work/noarchive"
mkdir "$noarchive"
cp "$release/checksums.txt" "$noarchive/"
if install_from "$noarchive" "$work/i6"; then
  fail "an install with no archive succeeded"
elif grep -q "could not download" "$work/err" && [ ! -e "$work/i6/mangabind" ]; then
  pass "a missing archive is an error that says so"
else
  fail "a missing archive is an error that says so"
  cat "$work/err" >&2
fi

# 7. An archive that checks out but holds no program.
empty="$work/empty"
mkdir "$empty"
echo "nothing here" >"$work/readme"
tar -czf "$empty/$archive" -C "$work" readme
(cd "$empty" && sha256 "$archive" >checksums.txt)
if install_from "$empty" "$work/i7"; then
  fail "an archive without the program was installed"
elif grep -q "does not hold a mangabind executable" "$work/err" && [ ! -e "$work/i7/mangabind" ]; then
  pass "an archive without the program is refused"
else
  fail "an archive without the program is refused"
  cat "$work/err" >&2
fi

# 8. A hash written in capitals is the same hash.
upper="$work/upper"
mkdir "$upper"
cp "$release/$archive" "$upper/"
awk '{ print toupper($1) "  " $2 }' "$release/checksums.txt" >"$upper/checksums.txt"
if install_from "$upper" "$work/i8" && [ -x "$work/i8/mangabind" ]; then
  pass "a checksum in capitals is accepted"
else
  fail "a checksum in capitals is accepted"
  cat "$work/err" >&2
fi

# 9. The address the release is asked for, with no network: a stand-in for curl
# that writes down what it was asked for and fails.
shim="$work/shim"
mkdir "$shim"
cat >"$shim/curl" <<'SHIM'
#!/bin/sh
for argument in "$@"; do
  case "$argument" in http*) echo "$argument" >>"$MANGABIND_TEST_LOG" ;; esac
done
exit 22
SHIM
chmod +x "$shim/curl"
asked() { # asked [env assignments...]: the first address the installer asks for
  : >"$work/log"
  env PATH="$shim:$PATH" MANGABIND_TEST_LOG="$work/log" MANGABIND_INSTALL_DIR="$work/i9" "$@" sh "$root/install.sh" >/dev/null 2>&1 || true
  head -n 1 "$work/log"
}
want="https://github.com/gustavommcv/mangabind/releases"
if [ "$(asked)" = "$want/latest/download/$archive" ]; then
  pass "with no version it asks for the latest release"
else
  fail "with no version it asks for the latest release (asked for: $(asked))"
fi
if [ "$(asked MANGABIND_VERSION=v0.7.0)" = "$want/download/v0.7.0/$archive" ]; then
  pass "a version from the environment is a release by that tag"
else
  fail "a version from the environment is a release by that tag (asked for: $(asked MANGABIND_VERSION=v0.7.0))"
fi
if [ "$(asked MANGABIND_VERSION=0.7.0)" = "$want/download/v0.7.0/$archive" ]; then
  pass "a version without its v gets one"
else
  fail "a version without its v gets one"
fi
: >"$work/log"
env PATH="$shim:$PATH" MANGABIND_TEST_LOG="$work/log" MANGABIND_INSTALL_DIR="$work/i9" sh -s -- 0.6.1 <"$root/install.sh" >/dev/null 2>&1 || true
if [ "$(head -n 1 "$work/log")" = "$want/download/v0.6.1/$archive" ]; then
  pass "a version given as an argument is a release by that tag"
else
  fail "a version given as an argument is a release by that tag (asked for: $(head -n 1 "$work/log"))"
fi
if grep -q "api.github.com" "$work/log"; then
  fail "the installer asks the API for a tag"
else
  pass "the installer does not ask the API for anything"
fi

# 10. The file is shell, as written.
if sh -n "$root/install.sh" && sh -n "$0"; then
  pass "install.sh and this script parse"
else
  fail "install.sh and this script parse"
fi
if command -v shellcheck >/dev/null 2>&1; then
  if shellcheck "$root/install.sh" "$0"; then
    pass "shellcheck has nothing to say"
  else
    fail "shellcheck has something to say"
  fi
else
  echo "skip  shellcheck is not installed"
fi

exit "$failed"
