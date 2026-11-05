#!/bin/bash
# COVMONSKILLS: monster-skill calc differential on the worker host.
#   tools/emu/run-on-worker.sh --ttl 2400 cms_run.sh N SEED > $D2_SCRATCH/out.txt 2>&1
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-20000}; SEED=${2:-1}
OUT=${D2_SCRATCH:-/tmp/d2agent-covmonskills}/calc$SEED
mkdir -p "$OUT"
. "$HERE/nec_env.sh"
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
$PY "$HERE/cms_calc.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/cms_calc.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
nice "$HOME/d2d-lab/pr/sorcrest-out/build/tests/test_sk_rest_calc" --calc "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E "^[0-9]+ [-0-9x]" > "$OUT/d2d.tsv"
wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<' || true)"
diff "$OUT/game.tsv" "$OUT/d2d.tsv" | head -40 || true
tail -3 "$OUT/game.err" "$OUT/d2d.err"
