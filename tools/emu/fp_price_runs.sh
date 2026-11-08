#!/bin/bash
# fidprice: the differentials of docs/research/fid-vendor-price-trace.md, on the worker host.
#   fp_price_runs.sh TREE BUILD OUTDIR gen N SEED | edge N SEED | clamp LIMIT CASES | tomes
# TREE is a clean checkout of upstream d2decomped main with `build` -> BUILD linked (tools/mpq-cat),
# BUILD has tests/test_game. Needs uv, D2_MPQ_DIR (the user's own game files) and D2_PATCH_INSTALLER.
# Prints `cases N, buy/sell/repair differ B S R, cases with a difference M, max buy P`; the differing
# lines (game.exe's `id buy sell repair` then d2d's) are in OUTDIR/mism.txt.
set -eu
T=$1; B=$2; O=$3; mode=$4; shift 4
export PATH=$HOME/.local/bin:$PATH
mkdir -p "$O"
here=$(cd "$(dirname "$0")" && pwd)
cp "$here"/price_edge.py "$here"/fp_clamp.py "$T/tools/emu/" 2>/dev/null || true
cd "$T/tools/emu"
case $mode in
  gen)   nice -n 19 uv run python price_oracle.py gen "$1" "$2" > "$O/cases.tsv" ;;
  edge)  nice -n 19 uv run python price_edge.py "$1" "$2" > "$O/cases.tsv" ;;
  clamp) python3 fp_clamp.py "$1" < "$2" > "$O/cases.tsv" ;;
  tomes) # a tome's book id is the item's first suffix word: 0 = tbk, 1 = ibk (suf0 is 0 in this build)
         i=0; for npc in akara charsi gheed malah; do for diff in 0 1 2; do for qty in 0 1 4 5 10 20; do
           echo "$i ibk 2 0 1 0 0 0 0 -1 1 0 0 0 $qty 0 $npc $diff -"; i=$((i+1))
           echo "$i tbk 2 0 0 0 0 0 0 -1 1 0 0 0 $qty 0 $npc $diff -"; i=$((i+1))
         done; done; done > "$O/cases.tsv" ;;
esac
nice -n 19 uv run python price_oracle.py game "$O/cases.tsv" 2>/dev/null > "$O/g.txt"
(cd "$B" && nice -n 19 tests/test_game --price "$O/cases.tsv" 2>/dev/null | grep -E '^[0-9]+ ' > "$O/d.txt")
paste -d' ' "$O/g.txt" "$O/d.txt" | awk -v d="$O" '
  { b += ($2 != $6); s += ($3 != $7); r += ($4 != $8); if ($2 > mx) mx = $2
    if ($2 != $6 || $3 != $7 || $4 != $8) { m++; print $0 > d "/mism.txt" } }
  END { print "cases", NR ", buy/sell/repair differ", b+0, s+0, r+0 ", cases with a difference", m+0 ", max buy", mx }'
