#!/usr/bin/env bash
# Repo gate: gofmt, vet, hygiene. Run from anywhere; exits non-zero on any failure.
# Usage: tools/ci/check.sh [--hygiene-only]
set -u
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
fail=0
bad() { echo "FAIL: $*"; fail=1; }

# Files to inspect: tracked plus untracked-not-ignored (so it works before commit).
files() { git ls-files -co --exclude-standard -z; }

# --- hygiene: never ship Blizzard bytes or secrets (gotcha 67) ---
while IFS= read -r -d '' f; do
  [ -e "$f" ] || continue
  case "$f" in
    *.mpq|*.MPQ|*.d2s|*.D2S|*.key|*.KEY) bad "forbidden file type: $f" ;;
    data/global/excel/*.txt|*/data/global/excel/*.txt) bad "extracted game table: $f" ;;
  esac
  case "$f" in
    */testdata/*|testdata/*) continue ;;
  esac
  sz=$(stat -c %s "$f")
  [ "$sz" -gt 1048576 ] && bad "file over 1 MB outside testdata: $f ($sz bytes)"
done < <(files)

[ "${1:-}" = "--hygiene-only" ] && { [ $fail = 0 ] && echo "hygiene ok"; exit $fail; }

# --- gofmt ---
out=$(gofmt -l $(git ls-files -co --exclude-standard '*.go') 2>&1)
[ -n "$out" ] && bad "gofmt differs:"$'\n'"$out"

# --- vet ---
go vet ./... || bad "go vet"

[ $fail = 0 ] && echo "check ok"
exit $fail
