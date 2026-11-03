#!/bin/bash
# covmissiles (the worker host): the Sorceress / Paladin missile-hit differential for the rows not yet run.
#   tools/emu/run-on-worker.sh --ttl 3000 cm_run.sh rows            # element columns of every Sor/Pal skill row, by game.exe's own tables
#   tools/emu/run-on-worker.sh --ttl 3000 cm_run.sh hit N SEED      # N random cases + the structured grid
# Needs cm_build.sh's binaries. Prints case counts and the number of differing lines (before = upstream main, after = SKMISS fix).
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
MODE=${1:?rows|hit}; N=${2:-3000}; SEED=${3:-1}
OUT=${D2_SCRATCH:-/tmp/d2agent-covmissiles}/$MODE$SEED
mkdir -p "$OUT"
export D2=$HOME/d2d-lab/pr/palskill
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
BIN=${D2_SCRATCH:-/tmp/d2agent-covmissiles}
if [ "$MODE" = rows ]; then
  nice $PY "$HERE/cm_missile_hit.py" rows
elif [ "$MODE" = flight ]; then
  bash "$HERE/mpq-table.sh" missiles > "$OUT/missiles.tsv"
  wc -l "$OUT/missiles.tsv"
  nice $PY "$HERE/cm_flight.py" game "$OUT" "$OUT/missiles.tsv" 2> "$OUT/game.err" || { tail -20 "$OUT/game.err"; exit 1; }
  cat "$OUT/game.err"
  nice "$BIN/flight_test" --flight "$OUT/cases.txt" 2> "$OUT/d2d.err" | grep -P '\t[0-9]+\t' > "$OUT/d2d.tsv"
  wc -l "$OUT/cases.txt" "$OUT/game.tsv" "$OUT/d2d.tsv"
  python3 "$HERE/cm_flightcmp.py" "$OUT"
else
  $PY "$HERE/cm_missile_hit.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
  nice $PY "$HERE/cm_missile_hit.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
  tail -2 "$OUT/game.err"
  for v in before after; do
    nice "$BIN/hit_$v" --missile-hit "$OUT/cases.tsv" 2> "$OUT/$v.err" | grep -E '^[0-9]+ ' > "$OUT/$v.tsv"
    echo "$v: $(wc -l < "$OUT/$v.tsv") lines, DIFF $(diff "$OUT/game.tsv" "$OUT/$v.tsv" | grep -c '^<' || true)"
  done
  wc -l "$OUT/cases.tsv" "$OUT/game.tsv"
fi
