#!/bin/bash
# verifycursedef: build the d2d probe (once) and run both sides of the independent curse-calc check on the worker host.
#   tools/emu/run-on-worker.sh --ttl 1800 vcd_run.sh N SEED
# d2d side: the sorcrest checkout's headers (eval_calc is inline in components/rules/skills.hpp) + its build's libgame.a,
# the tree the author's test_sk_rest_calc came from.
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-20000}; SEED=${2:-424242}
OUT=${D2_SCRATCH:?}/vcd
mkdir -p "$OUT"
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
B=$HOME/d2d-lab/pr/sorcrest-out/build
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
if [ ! -x "$OUT/vcd_test" ] || [ "$HERE/vcd_test.cpp" -nt "$OUT/vcd_test" ]; then
	IN=""
	for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
	STORM=$B/_deps/stormlib-src/src
	[ -d "$STORM" ] || STORM=$HOME/d2d-lab/src/build/_deps/stormlib-src/src
	(cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$STORM -O2 -std=c++26 -include $SRC/cmake/no_near_far.hpp -o "$OUT/vcd_test" "$HERE/vcd_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
fi
"$OUT/vcd_test" --dump > "$OUT/dump.txt" 2>&1 || true
UP=$HOME/d2d-lab/pr/palskill
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$UP/tools/emu
cd "$UP/tools/emu"
nice $PY "$HERE/vcd_calc.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/vcd_calc.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
nice "$OUT/vcd_test" --calc "$OUT/cases.tsv" 2> "$OUT/d2d.err" | grep -E '^[0-9]+ ' > "$OUT/d2d.tsv"
wc -l "$OUT/cases.tsv" "$OUT/game.tsv" "$OUT/d2d.tsv"
echo "DIFF $(diff "$OUT/game.tsv" "$OUT/d2d.tsv" | grep -c '^<' || true)"
grep READS "$OUT/game.err" || tail -5 "$OUT/game.err"
cat "$OUT/dump.txt" | head -150
