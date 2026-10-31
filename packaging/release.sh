#!/usr/bin/env bash
# Build a versioned release tarball locally. Never uploads, tags or pushes anything.
#   packaging/release.sh [--dry-run] [VERSION]      VERSION default: git describe --tags --always --dirty
# Layout (linux-amd64): <name>/bin/<tools>, <name>/helper/d2input.exe (windows/386), RELEASE_NOTES.md, CHANGELOG.md,
# THIRD_PARTY.md, VERSION, SHA256SUMS (every file in the tree); beside it <name>.tar.gz and <name>.tar.gz.sha256.
# --dry-run builds into a temp dir (removed afterwards) and runs packaging/release_test.sh on the result.
# Env: DIST=dir overrides the output dir (default dist/).
set -euo pipefail
cd "$(dirname "$0")/.."

dry=0
if [ "${1:-}" = "--dry-run" ]; then dry=1; shift; fi
ver="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"  # a copy without .git needs VERSION or gets "dev"
name="d2lod-bot-$ver-linux-amd64"
dist="${DIST:-dist}"
if [ $dry = 1 ]; then dist="$(mktemp -d)"; trap 'rm -rf "$dist"' EXIT; fi
stage="$dist/$name"
rm -rf "$stage" "$dist/$name.tar.gz" "$dist/$name.tar.gz.sha256"
mkdir -p "$stage/bin" "$stage/helper"

# Every cmd/ below is a shipped tool. Live-test rigs (act2live, navlive, ...) are not released.
LINUX_BINS="d2bot d2fleet d2doctor d2soak probes d2sentinel d2cheat d2multi d2snap d2view d2watch d2dash oog"
# d2bot reads main.version (kept for compatibility), the others read buildinfo.Version.
ld="-X main.version=$ver -X d2lod-bot/internal/buildinfo.Version=$ver"
for c in $LINUX_BINS; do
  [ -d "cmd/$c" ] || { echo "release.sh: cmd/$c missing" >&2; exit 1; }
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$ld" -o "$stage/bin/$c" "./cmd/$c"
done
# The only Windows program; it runs inside the Wine prefix, so no installer is shipped (docs/release.md).
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags "-H=windowsgui $ld" -o "$stage/helper/d2input.exe" ./cmd/d2input

# Web assets are go:embed in the binaries (d2watch, d2dash, d2view, d2fleet setup wizard); nothing to copy.
echo "$ver" > "$stage/VERSION"
cp THIRD_PARTY.md CHANGELOG.md "$stage/"
[ -f LICENSE ] && cp LICENSE "$stage/"

# Release notes: the "Unreleased / known issues" section plus the newest dated section of CHANGELOG.md.
awk '
  /^## / { n++ }
  n >= 1 && n <= 2 { print }
' CHANGELOG.md > "$stage/RELEASE_NOTES.md"
[ -s "$stage/RELEASE_NOTES.md" ] || { echo "release.sh: no CHANGELOG sections found" >&2; exit 1; }
sed -i "1i # d2lod-bot $ver (linux-amd64)\n" "$stage/RELEASE_NOTES.md"

(cd "$stage" && find . -type f ! -name SHA256SUMS | LC_ALL=C sort | sed 's|^\./||' | xargs sha256sum > SHA256SUMS)
epoch="$(git log -1 --format=%ct 2>/dev/null || date +%s)"
tar -C "$dist" --sort=name --owner=0 --group=0 --numeric-owner --mtime="@$epoch" -czf "$dist/$name.tar.gz" "$name"
(cd "$dist" && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")

if [ $dry = 1 ]; then
  bash packaging/release_test.sh "$dist/$name.tar.gz" "$ver"
  echo "dry run OK"
else
  echo "$dist/$name.tar.gz"
fi
