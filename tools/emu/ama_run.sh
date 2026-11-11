#!/bin/bash
# skamazon (the worker host): the Amazon calc and missile-hit differentials.
#   tools/emu/run-on-worker.sh --ttl 3000 ama_run.sh calc N SEED
#   tools/emu/run-on-worker.sh --ttl 3000 ama_run.sh hit N SEED
# Needs the binaries of ama_build.sh. Prints case counts and the number of differing lines.
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
MODE=${1:?calc|hit}; N=${2:-3000}; SEED=${3:-1}
OUT=${D2_SCRATCH:-/tmp/d2agent-skamazon}/$MODE$SEED
mkdir -p "$OUT"
export D2=$HOME/d2d-lab/pr/palskill
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
BIN=${D2_SCRATCH:-/tmp/d2agent-skamazon}
if [ "$MODE" = calc ]; then
  $PY "$HERE/ama_calc.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
  nice $PY "$HERE/ama_calc.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
  nice "$BIN/ama_test" --calc "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E "^[0-9]+ [-0-9x]" > "$OUT/d2d.tsv"
  wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
  echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<' || true)"
else
  $PY "$HERE/ama_missile_hit.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
  nice $PY "$HERE/ama_missile_hit.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
  tail -2 "$OUT/game.err"
  for v in before after; do
    nice "$BIN/hit_$v" --missile-hit "$OUT/cases.tsv" 2> "$OUT/$v.err" | grep -E '^[0-9]+ ' > "$OUT/$v.tsv"
    echo "$v: $(wc -l < "$OUT/$v.tsv") lines, DIFF $(diff "$OUT/game.tsv" "$OUT/$v.tsv" | grep -c '^<' || true)"
  done
  wc -l "$OUT/cases.tsv" "$OUT/game.tsv"
fi
