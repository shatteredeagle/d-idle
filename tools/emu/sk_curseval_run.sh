#!/bin/bash
# SKCURSEVAL: the curse-calc differential on the worker host.
#   tools/emu/run-on-worker.sh --ttl 1500 sk_curseval_run.sh N SEED > $D2_SCRATCH/out.txt 2>&1
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-20000}; SEED=${2:-1}
OUT=${D2_SCRATCH:-/tmp/d2agent-skcurseval}/curse
mkdir -p "$OUT"
. "$HERE/nec_env.sh"
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
$PY "$HERE/sk_curseval.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/sk_curseval.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
nice "$HOME/d2d-lab/pr/sorcrest-out/build/tests/test_sk_rest_calc" --calc "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E "^[0-9]+ [-0-9x]" > "$OUT/d2d.tsv"
wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<' || true)"
md5sum "$OUT/game.tsv" "$OUT/d2d.tsv"
