#!/bin/bash
# verifyskamazon: element roll, game.exe FUN_005a8910 vs d2d Rng::range, both on the worker host.
#   tools/emu/run-on-worker.sh --ttl 1800 vam_roll.sh N SEED
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-20000}; SEED=${2:-5}
OUT=${D2_SCRATCH:?}/vam
mkdir -p "$OUT"
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
B=$HOME/d2d-lab/pr/sorcrest-out/build
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
if [ ! -x "$OUT/vam_test" ] || [ "$HERE/vam_test.cpp" -nt "$OUT/vam_test" ]; then
	IN=""
	for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
	STORM=$B/_deps/stormlib-src/src
	[ -d "$STORM" ] || STORM=$HOME/d2d-lab/src/build/_deps/stormlib-src/src
	(cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$STORM -O2 -std=c++26 -include $SRC/cmake/no_near_far.hpp -o "$OUT/vam_test" "$HERE/vam_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
fi
UP=$HOME/d2d-lab/pr/palskill
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$UP/tools/emu
cd "$UP/tools/emu"
nice $PY "$HERE/vam_roll.py" gen "$N" "$SEED" > "$OUT/rcases.txt"
nice $PY "$HERE/vam_roll.py" game "$OUT/rcases.txt" > "$OUT/rgame.txt" 2> "$OUT/rgame.err"
nice "$OUT/vam_test" --roll "$OUT/rcases.txt" > "$OUT/rd2d.txt"
wc -l "$OUT/rcases.txt" "$OUT/rgame.txt" "$OUT/rd2d.txt"
paste -d' ' "$OUT/rcases.txt" "$OUT/rgame.txt" "$OUT/rd2d.txt" | "$PY" -I -c '
import sys
n = diff = state = excl = maxhit = nonzero_span = 0
for l in sys.stdin:
    lo, hi, mn, mx, g, glo, ghi, d, dlo, dhi = map(int, l.split())
    n += 1
    if mx > mn:
        nonzero_span += 1
        if g == mx: maxhit += 1
        if not mn <= g <= mx - 1: excl += 1
    if g != d: diff += 1
    if (glo, ghi) != (dlo, dhi) and mx > mn: state += 1
print("CASES", n, "span>0", nonzero_span, "game result == max", maxhit, "game result outside [min,max-1]", excl, "game vs d2d result differs", diff, "state after differs", state)
'
