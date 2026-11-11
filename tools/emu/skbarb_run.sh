#!/bin/bash
# skbarb: Barbarian calc differential on the worker host, run through tools/emu/run-on-worker.sh.
#   tools/emu/run-on-worker.sh --ttl 2400 skbarb_run.sh N SEED
# Builds skbarb_test.cpp by hand against the skmana scratch build (same recipe as nec_build.sh) into $D2_SCRATCH,
# then game.exe's FUN_00646ca0 (skbarb_calc.py game) vs d2d eval_calc (skbarb_test --calc).
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-3000}; SEED=${2:-1}
S=${D2_SCRATCH:-/tmp/d2agent-skbarb}
OUT=$S/run$SEED
mkdir -p "$OUT"
. "$HERE/skbarb_env.sh"
SRC=$HOME/d2d-lab/pr/palskill
B=$HOME/d2d-lab/pr/skmana-out/build
if [ ! -x "$S/skbarb_test" ] || [ "$HERE/skbarb_test.cpp" -nt "$S/skbarb_test" ]; then
  IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
  (cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O3 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$S/skbarb_test" "$HERE/skbarb_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
fi
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
$PY "$HERE/skbarb_calc.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/skbarb_calc.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
nice "$S/skbarb_test" --calc "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E "^[0-9]+ [-0-9x]" > "$OUT/d2d.tsv"
wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
if diff -q "$OUT/game.tsv" "$OUT/d2d.tsv" >/dev/null; then echo "DIFF 0"; else echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<')"; diff "$OUT/game.tsv" "$OUT/d2d.tsv" | head -20; fi
