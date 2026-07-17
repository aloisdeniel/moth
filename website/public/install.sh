#!/bin/sh
# moth installer — https://github.com/aloisdeniel/moth
#
# Downloads the latest signed moth release for this machine, verifies its
# SHA-256 against the release checksums, and installs the single static
# binary. No dependencies beyond curl/wget and tar.
#
#   curl -fsSL https://moth.dev/install.sh | sh
#
# Environment overrides:
#   MOTH_VERSION      install a specific version (e.g. "1.0.0" or "v1.0.0")
#                     instead of the latest release
#   MOTH_INSTALL_DIR  install directory (default /usr/local/bin, falling
#                     back to ~/.local/bin when not writable and sudo is
#                     unavailable)
#
# The script is deliberately boring: detect platform, download, verify,
# copy one file. Uninstall by deleting that file.

set -eu

REPO="aloisdeniel/moth"

say() { printf '%s\n' "$*"; }
fail() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

# --- platform ---------------------------------------------------------------

os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *)
    fail "unsupported OS '$os' — Windows and others: download an archive from
  https://github.com/$REPO/releases (windows_amd64.zip) or build from source."
    ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported architecture '$arch' (need amd64 or arm64)" ;;
esac

# --- fetch helpers (curl or wget) -------------------------------------------

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
  fetch_stdout() { wget -qO- "$1"; }
else
  fail "need curl or wget"
fi

# --- resolve version --------------------------------------------------------

version="${MOTH_VERSION:-}"
if [ -z "$version" ]; then
  # tag_name of the latest release, without depending on jq.
  version=$(fetch_stdout "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null |
    sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1) || true
  [ -n "$version" ] || fail "no published release found yet — moth binaries ship
  with the first tagged release. Until then, build from source (Go 1.25+):
    git clone https://github.com/$REPO && cd moth && make build"
fi
version=${version#v} # archives are named without the leading v

base_url="https://github.com/$REPO/releases/download/v$version"
archive="moth_${version}_${os}_${arch}.tar.gz"

say "moth v$version ($os/$arch)"

# --- download + verify ------------------------------------------------------

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "downloading $archive ..."
fetch "$base_url/$archive" "$tmp/$archive" ||
  fail "download failed: $base_url/$archive"
fetch "$base_url/checksums.txt" "$tmp/checksums.txt" ||
  fail "download failed: $base_url/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  sum=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
  sum=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
else
  fail "need sha256sum or shasum to verify the download"
fi
want=$(grep "  $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$want" ] || fail "checksums.txt has no entry for $archive"
[ "$sum" = "$want" ] || fail "checksum mismatch for $archive
  expected: $want
  got:      $sum"
say "checksum verified."

tar -xzf "$tmp/$archive" -C "$tmp" moth

# --- install ----------------------------------------------------------------

dir="${MOTH_INSTALL_DIR:-/usr/local/bin}"
install_to() {
  # $1 = install dir; assumes it is writable (possibly via the sudo wrapper).
  "$@" mkdir -p "$dir" 2>/dev/null || return 1
  "$@" install -m 0755 "$tmp/moth" "$dir/moth" 2>/dev/null || return 1
}

if install_to env; then
  :
elif [ -z "${MOTH_INSTALL_DIR:-}" ] && command -v sudo >/dev/null 2>&1; then
  say "installing to $dir needs sudo:"
  install_to sudo || fail "could not install to $dir"
elif [ -z "${MOTH_INSTALL_DIR:-}" ]; then
  dir="$HOME/.local/bin"
  install_to env || fail "could not install to $dir"
  case ":$PATH:" in
    *":$dir:"*) ;;
    *) say "note: add $dir to your PATH." ;;
  esac
else
  fail "could not install to $dir"
fi

say ""
say "installed: $dir/moth"
"$dir/moth" version 2>/dev/null || true
say ""
say "next steps:"
say "  moth serve --data-dir ./data     # admin console at http://localhost:8080/admin"
say "  docs: https://github.com/$REPO#readme"
