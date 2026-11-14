#!/bin/bash
# verifyskdruid (worker host, through run-on-worker.sh): own Druid calc differential with separate blvl / lvl.
#   vsd_run.sh N SEED      game.exe FUN_00646ca0 (vsd_calc.py) vs d2d eval_calc (vsd_test.cpp) on the same cases
#   vsd_run.sh ptrs        the calc offsets of every Druid cell
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:-/tmp/d2agent-pa-verifyskdruid}/vsd
mkdir -p "$OUT"
D2=$HOME/d2d-lab/src
PY=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python
export PYTHONPATH=$HERE:$D2/tools/emu
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
if [ "$1" = ptrs ]; then cd "$D2/tools/emu"; exec nice $PY "$HERE/vsd_calc.py" ptrs; fi
N=${1:-3000}; SEED=${2:-101}; MODE=${3:-game}   # MODE game-noflag: negative control (blvl = base + bonus)
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
B=$HOME/d2d-lab/pr/sorcrest-out/build
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
if [ ! -x "$OUT/vsd_test" ] || [ "$HERE/vsd_test.cpp" -nt "$OUT/vsd_test" ]; then
  (cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$D2/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/vsd_test" "$HERE/vsd_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
fi
cd "$D2/tools/emu"
$PY "$HERE/vsd_calc.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/vsd_calc.py" "$MODE" "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
nice "$OUT/vsd_test" "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E "^[0-9]+ [-0-9x]" > "$OUT/d2d.tsv"
wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
grep ASKED "$OUT/game.err" | awk '{print $5}' | sort | uniq -c
echo "first rows (game):"; head -3 "$OUT/game.tsv"
if diff -q "$OUT/game.tsv" "$OUT/d2d.tsv" >/dev/null; then echo "DIFF 0"; else echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<')"; diff "$OUT/game.tsv" "$OUT/d2d.tsv" | head -30; fi
cp "$OUT/game.tsv" "$OUT/game-$SEED.tsv"; cp "$OUT/cases.tsv" "$OUT/cases-$SEED.tsv"
