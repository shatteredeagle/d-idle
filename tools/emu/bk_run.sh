#!/bin/bash
# decomp11 (worker host): monster defender block chance, game.exe FUN_00622720 natively vs d2d's loaded rule.
#   tools/emu/run-on-worker.sh --ttl 2400 bk_run.sh
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:?}/bk
mkdir -p "$OUT"
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=$HOME/d2d-lab/pr/sorcrest-out/build
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/game/fight components/game/world components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
STORM=$B/_deps/stormlib-src/src
[ -d "$STORM" ] || STORM=$HOME/d2d-lab/src/build/_deps/stormlib-src/src
(cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$STORM -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/bk_d2d_test" "$HERE/bk_d2d_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
"$OUT/bk_d2d_test" > "$OUT/d2d.out" 2> "$OUT/d2d.err" || true
wc -l "$OUT/d2d.out"; grep -v WARNING "$OUT/d2d.err" | head -3
UP=${UP:-$HOME/d2d-lab/src}
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$UP/tools/emu
cd "$UP/tools/emu"
: > "$OUT/game.out"
for d in 0 1 2; do nice $PY "$HERE/bk_mon.py" game $d >> "$OUT/game.out" 2>> "$OUT/game.err" || true; done
grep -v WARNING "$OUT/game.err" | tail -3
wc -l "$OUT/game.out"; head -4 "$OUT/game.out"
nice $PY "$HERE/bk_mon.py" cmp "$OUT/game.out" "$OUT/d2d.out"
