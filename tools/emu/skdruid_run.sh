#!/bin/bash
# skdruid (worker host, through run-on-worker.sh): build the d2d side by hand, run game.exe's calc
# machine and d2d on the same cases, diff.   skdruid_run.sh N SEED
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-3000}; SEED=${2:-1}
OUT=${D2_SCRATCH:-/tmp/d2agent-pa-skdruid}/skdruid
mkdir -p "$OUT"
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
B=$HOME/d2d-lab/pr/sorcrest-out/build
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
if [ ! -x "$OUT/skdruid_test" ]; then
  (cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/skdruid_test" "$HERE/skdruid_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
fi
D2=$HOME/d2d-lab/src
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
$PY "$HERE/skdruid_calc.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/skdruid_calc.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
nice "$OUT/skdruid_test" --calc "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E "^[0-9]+ [-0-9x]" > "$OUT/d2d.tsv"
wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
if diff -q "$OUT/game.tsv" "$OUT/d2d.tsv" >/dev/null; then echo "DIFF 0"; else echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<')"; diff "$OUT/game.tsv" "$OUT/d2d.tsv" | head -20; fi
