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
    *.dc6|*.DC6|*.dcc|*.DCC|*.dt1|*.DT1) bad "game art file: $f (the watch art is read from the user's install at runtime, docs/paperdoll-art-build.md)" ;;
    internal/watch/web/*.png|internal/watch/web/*/*.png|*/web/art/*|*/cache/art/*) bad "art image in the tree: $f (art is never committed or embedded, docs/paperdoll-art-build.md)" ;;
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

# --- drift report as a warning only (tools/backlog/drift.py exits 0): tasks whose status disagrees with git ---
python3 -I tools/backlog/drift.py --summary || true

# --- d2d shim register (docs/d2d-deviations.md): every d2d branch outside the adapters needs a row ---
python3 -I tools/shims/check.py || fail=1
python3 -I tools/shims/test_check.py >/dev/null 2>&1 || bad "tools/shims/test_check.py"

# --- dark-launch ledger: every [dark-launch] knob has a row, no open row past its review-by date (docs/process/dark-launch.md) ---
python3 -I tools/darklaunch/check.py || fail=1
python3 -I tools/darklaunch/test_check.py >/dev/null 2>&1 || bad "tools/darklaunch/test_check.py"

# --- contracts: headers, requirements, knob citations (docs/process/contracts-after-openspec.md 3.8) ---
python3 -I tools/contracts/check.py || fail=1
python3 -I tools/contracts/test_check.py >/dev/null 2>&1 || bad "tools/contracts/test_check.py"

# --- branch ownership: changed files inside the card's owns:, ADR/decisions/GLOSSARY only with allow-adr (tools/ci/ownscheck.py) ---
# Owns findings are warnings until 2026-12-01 (rollout); the ADR guard fails now. main and non-agent branches are skipped.
python3 -I tools/ci/ownscheck.py --warn-owns-until 2026-12-01 || fail=1
python3 -I tools/ci/test_ownscheck.py >/dev/null 2>&1 || bad "tools/ci/test_ownscheck.py"

# --- threshold governance (P10, docs/process/threshold-governance.md): knob net count and evidence rows; report-only until one session has passed ---
python3 -I tools/ci/knobbudget.py --report || fail=1
python3 -I tools/ci/test_knobbudget.py >/dev/null 2>&1 || bad "tools/ci/test_knobbudget.py"

# --- input layer I6: pointer call sites outside the adapters may fall, never rise (docs/design/input-i5-i7.md) ---
python3 -I tools/ci/pointercount.py || fail=1
python3 -I tools/ci/test_pointercount.py >/dev/null 2>&1 || bad "tools/ci/test_pointercount.py"

# --- no decompiled game code or game data tables (docs/process/gamecode-guard.md) ---
tools/ci/gamecode_guard.sh || fail=1

# --- watch UI panel contract: layered CSS, panel scope, no inline styles (tools/uilint, docs/ui-regression-plan.md) ---
go run ./tools/uilint >/dev/null 2>&1 || { go run ./tools/uilint; bad "tools/uilint (watch UI panel contract)"; }

# --- build tables, combat profiles and the capability table agree (fable review 8; ratchet data/build-cap-ratchet.json) ---
go test -count=1 ./internal/plandata -run TestBuildTablesConsistent >/dev/null 2>&1 || { go test -count=1 ./internal/plandata -run TestBuildTablesConsistent; bad "build data consistency (internal/plandata TestBuildTablesConsistent)"; }

[ "${1:-}" = "--hygiene-only" ] && { [ $fail = 0 ] && echo "hygiene ok"; exit $fail; }

# --- gofmt ---
out=$(gofmt -l $(git ls-files -co --exclude-standard '*.go') 2>&1)
[ -n "$out" ] && bad "gofmt differs:"$'\n'"$out"

# --- vet ---
go vet ./... || bad "go vet"

# Opt-in (CROSSBUILD=1): windows+darwin vet/build of the shipped binaries; the full matrix is the ci.yml crossbuild job.
if [ "${CROSSBUILD:-}" = 1 ]; then tools/ci/crossbuild.sh --quick || fail=1; fi

[ $fail = 0 ] && echo "check ok"
exit $fail
