#!/bin/bash
# COVMONSKILLS: column differential on the worker host (builds the d2d probe first).
#   tools/emu/run-on-worker.sh --ttl 2400 cms_cols_run.sh > $D2_SCRATCH/cols.txt 2>&1
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:-/tmp/d2agent-covmonskills}/cols
mkdir -p "$OUT"
bash "$HERE/cms_build.sh"
. "$HERE/nec_env.sh"
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
$PY "$HERE/cms_cols.py" > "$OUT/game.tsv" 2> "$OUT/game.err"
"$HOME/d2d-lab/pr/cms-out/cms_cols_test" > "$OUT/d2d.tsv" 2> "$OUT/d2d.err"
wc -l "$OUT/game.tsv" "$OUT/d2d.tsv"
echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<' || true)"
diff "$OUT/game.tsv" "$OUT/d2d.tsv" | head -60 || true
