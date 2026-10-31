#!/usr/bin/env bash
# Verify a release tarball: layout, SHA256SUMS, version stamps, release notes. Usage: release_test.sh TARBALL VERSION
set -euo pipefail
tb="$1"; ver="$2"
name="$(basename "$tb" .tar.gz)"
work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT
fail() { echo "FAIL: $*" >&2; exit 1; }
tar -C "$work" -xzf "$tb"
r="$work/$name"
[ -d "$r" ] || fail "top dir $name missing"

for c in d2bot d2fleet d2doctor d2soak probes d2sentinel d2cheat d2multi d2snap d2view d2watch d2dash oog; do
  [ -x "$r/bin/$c" ] || fail "bin/$c missing or not executable"
  out="$("$r/bin/$c" --version)" || fail "bin/$c --version failed"
  [ "$out" = "$c $ver" ] || fail "bin/$c --version printed '$out', want '$c $ver'"
done
[ -s "$r/helper/d2input.exe" ] || fail "helper/d2input.exe missing"
[ "$(head -c2 "$r/helper/d2input.exe")" = "MZ" ] || fail "d2input.exe is not a PE file"
grep -aq "$ver" "$r/helper/d2input.exe" || fail "version not stamped into d2input.exe"
for f in VERSION CHANGELOG.md RELEASE_NOTES.md THIRD_PARTY.md SHA256SUMS; do [ -s "$r/$f" ] || fail "$f missing"; done
[ "$(cat "$r/VERSION")" = "$ver" ] || fail "VERSION file wrong"
head -1 "$r/RELEASE_NOTES.md" | grep -q "$ver" || fail "RELEASE_NOTES.md header has no version"
grep -q "^## Unreleased" "$r/RELEASE_NOTES.md" || fail "RELEASE_NOTES.md lacks the Unreleased section"
(cd "$r" && sha256sum --quiet -c SHA256SUMS) || fail "SHA256SUMS does not verify"
n=$(find "$r" -type f ! -name SHA256SUMS | wc -l); m=$(wc -l < "$r/SHA256SUMS")
[ "$n" = "$m" ] || fail "SHA256SUMS lists $m files, tree has $n"
(cd "$(dirname "$tb")" && sha256sum --quiet -c "$name.tar.gz.sha256") || fail "tarball checksum does not verify"
# Embedded web assets really are in the binaries (go:embed => the files are not shipped loose).
[ "$(stat -c %s "$r/bin/d2watch")" -gt 1000000 ] || fail "d2watch suspiciously small"
if find "$r" -name '*.exe' ! -path '*/helper/d2input.exe' | grep -q .; then fail "unexpected extra .exe"; fi
if find "$r" \( -name '*.key' -o -name '*.sav' -o -name '*.d2s' -o -name '*.mpq' \) | grep -q .; then fail "forbidden file type in release"; fi
echo "release layout OK: $name"
