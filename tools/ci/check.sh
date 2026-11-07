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
  case "$f" in
    docs/probes/*) [ "$sz" -gt 524288 ] && bad "probe result over 512 KB (cap it: d2drill -cap $f): $f ($sz bytes)" ;;
  esac
done < <(files)

# --- merge-lines.txt of the cards this branch touches must parse (tools/merge_lines_check.py) ---
python3 tools/merge_lines_check.py --changed || fail=1

# --- docs/cards.md is generated and not committed (TASK-116); only check the generator runs ---
python3 -I tools/backlog/cards_md.py --check || fail=1

# --- d2d shim register (docs/d2d-deviations.md): every d2d branch outside the adapters needs a row ---
python3 -I tools/shims/check.py || fail=1
python3 -I tools/shims/test_check.py >/dev/null 2>&1 || bad "tools/shims/test_check.py"

# --- contracts: headers, requirements, knob citations (docs/process/contracts-after-openspec.md 3.8) ---
python3 -I tools/contracts/check.py || fail=1
python3 -I tools/contracts/test_check.py >/dev/null 2>&1 || bad "tools/contracts/test_check.py"

# --- no decompiled game code or game data tables (docs/process/gamecode-guard.md) ---
tools/ci/gamecode_guard.sh || fail=1

[ "${1:-}" = "--hygiene-only" ] && { [ $fail = 0 ] && echo "hygiene ok"; exit $fail; }

# --- gofmt ---
out=$(gofmt -l $(git ls-files -co --exclude-standard '*.go') 2>&1)
[ -n "$out" ] && bad "gofmt differs:"$'\n'"$out"

# --- vet ---
go vet ./... || bad "go vet"

[ $fail = 0 ] && echo "check ok"
exit $fail
