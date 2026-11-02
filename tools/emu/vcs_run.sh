#!/bin/bash
# verifycovstates (worker host): build the d2d probe by hand against the scratch upstream build, then run both sides.
#   tools/emu/run-on-worker.sh --ttl 3000 vcs_run.sh STUN_N POISON_N SEED
# d2d probe: recipe of ms_build.sh (sorcrest scratch checkout + its build). Everything written under $D2_SCRATCH.
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
NS=${1:-30000}; NP=${2:-30000}; SEED=${3:-731}
OUT=${D2_SCRATCH:?}/vcs
mkdir -p "$OUT"
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=$HOME/d2d-lab/pr/sorcrest-out/build
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
if [ ! -x "$OUT/vcs_d2d_test" ] || [ "$HERE/vcs_d2d_test.cpp" -nt "$OUT/vcs_d2d_test" ]; then
	IN=""
	for d in tests components/game components/game/character components/game/gamedata components/game/npc components/game/fight components/game/world components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
	STORM=$B/_deps/stormlib-src/src
	[ -d "$STORM" ] || STORM=$HOME/d2d-lab/src/build/_deps/stormlib-src/src
	(cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$STORM -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/vcs_d2d_test" "$HERE/vcs_d2d_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
fi
UP=${UP:-$HOME/d2d-lab/src}
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
export PYTHONPATH=$HERE:$UP/tools/emu
bash "$HERE/mpq-table.sh" monstats > "$OUT/monstats.txt"
export VCS_MONSTATS="$OUT/monstats.txt"
head -c 120 "$OUT/monstats.txt"; echo
cd "$UP/tools/emu"
nice $PY "$HERE/vcs_game.py" census 2>&1 | head -5
nice "$OUT/vcs_d2d_test" --types > "$OUT/types.d2d" 2> "$OUT/types.err" || true
nice $PY "$HERE/vcs_game.py" compare types "$OUT/types.d2d" || true
grep -v WARNING "$OUT/types.err" | head -20
grep -n -E "^(willowisp8|vampire9|willowisp1|overseer6)	" "$OUT/monstats.txt" | cut -c1-60
for kind in stun poison; do
	N=$NS; [ $kind = poison ] && N=$NP
	nice $PY "$HERE/vcs_game.py" gen $kind "$N" "$SEED" > "$OUT/$kind.cases"
	nice $PY "$HERE/vcs_game.py" game $kind "$OUT/$kind.cases" > "$OUT/$kind.game" 2> "$OUT/$kind.game.err" || true
	nice "$OUT/vcs_d2d_test" --$kind "$OUT/$kind.cases" > "$OUT/$kind.d2d" 2> "$OUT/$kind.d2d.err" || true
	wc -l "$OUT/$kind.cases" "$OUT/$kind.game" "$OUT/$kind.d2d"
	tail -n 3 "$OUT/$kind.game.err"; tail -n 3 "$OUT/$kind.d2d.err"
	nice $PY "$HERE/vcs_game.py" compare $kind "$OUT/$kind.cases" "$OUT/$kind.game" "$OUT/$kind.d2d" || true
done
