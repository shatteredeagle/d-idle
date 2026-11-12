#!/bin/bash
# decomp11 (worker host): Chain Lightning hop pick, game.exe callback 0x56bcd0 vs d2d rules::ring_pick.
#   tools/emu/run-on-worker.sh --ttl 900 ch_pick_run.sh N SEED
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-50000}; SEED=${2:-11}
OUT=${D2_SCRATCH:?}/chp
mkdir -p "$OUT"
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
UP=${UP:-$HOME/d2d-lab/src}
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$UP/tools/emu
IN=""
for d in components/game components/game/character components/game/gamedata components/game/npc components/game/fight components/game/world components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
nice c++ $IN -O1 -std=c++26 -o "$OUT/ch_pick_test" "$HERE/ch_pick_test.cpp"
cd "$UP/tools/emu"
nice $PY "$HERE/ch_pick.py" gen "$N" "$SEED" > "$OUT/cases"
nice $PY "$HERE/ch_pick.py" game "$OUT/cases" > "$OUT/game.raw" 2> "$OUT/game.err" || true
tail -n 3 "$OUT/game.err"
awk '{print $1, $2}' "$OUT/game.raw" > "$OUT/game.out"
"$OUT/ch_pick_test" < "$OUT/cases" > "$OUT/d2d.out"
wc -l "$OUT/cases" "$OUT/game.out" "$OUT/d2d.out"
diff "$OUT/game.out" "$OUT/d2d.out" | head -10 || true
echo "differing lines: $(diff "$OUT/game.out" "$OUT/d2d.out" | grep -c '^<')"
echo "picked none: $(grep -c ' -$' "$OUT/game.out"); first lines:"; head -3 "$OUT/game.raw"
