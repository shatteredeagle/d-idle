#!/bin/bash
# covstates2 (the worker host): the whole put differential, end to end.
#   tools/emu/run-on-worker.sh --ttl 3000 cs2_run.sh N SEED > $D2_SCRATCH/cs2_out.txt 2>&1
# gen -> game.exe's FUN_0056e970 (cs2_put.py game) -> d2d's real put_state (cs2_d2d_test, built by cs2_build.sh) ->
# the Python transcription (cs2_put.py model) -> comparisons, then the switch table (cs2_put.py put N SEED).
set -e
here=$(cd "$(dirname "$0")" && pwd)
N=${1:-100000}; SEED=${2:-20261102}
O=${D2_SCRATCH:-/tmp/d2agent-covstates2}/put_$SEED
mkdir -p "$O"
P="bash $here/run-py.sh cs2_put.py"
$P gen "$N" "$SEED" > "$O/cases.tsv"
$P game "$O/cases.tsv" > "$O/game.tsv"
nice "${D2_SCRATCH:-/tmp/d2agent-covstates2}/cs2_d2d_test" "$O/cases.tsv" > "$O/d2d.tsv"
$P model "$O/cases.tsv" > "$O/model.tsv"
wc -l "$O/cases.tsv" "$O/game.tsv" "$O/d2d.tsv" "$O/model.tsv"
$P cmp "$O/d2d.tsv" "$O/model.tsv" "d2d-real" "model"
$P cmp "$O/game.tsv" "$O/d2d.tsv" "game.exe" "d2d-real"
$P put "$N" "$SEED"
