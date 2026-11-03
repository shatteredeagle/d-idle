#!/bin/bash
# verifydmg4: (1) check that the author's r4 tree and build match my git archive of 9ea2afe9 (read only);
# (2) link tools/emu/v4_probe.cpp against that build's libgame.a (the REAL Fight::absorb / hurt_side) with
# test_game.cpp's own compile flags; (3) compile tools/emu/v4_d2d.cpp (my header glue) for base and r4.
# nice 19, one compile at a time; outputs in $D2_SCRATCH.
#   tools/emu/run-on-worker.sh --ttl 1500 v4_build.sh
set -u
S=${D2_SCRATCH:?}
here=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-mdmg-r4; B=$HOME/d2d-lab/build-mdr4; X=$S/t_r4
echo "== author tree vs my export (components, tests, cmake)"
diff -rq "$T/components" "$X/components" | head; diff -rq "$T/tests" "$X/tests" | head; diff -rq "$T/cmake" "$X/cmake" | head
echo "diff done"
echo "== newest source vs libgame.a"
ls -l --time-style=+%s "$B/components/game/libgame.a" | awk '{print "libgame.a " $6}'
find "$T/components/game" "$T/components/rules" -newer "$B/components/game/libgame.a" -type f | head
echo "(files above are newer than libgame.a; none = the library is current)"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$X/$d"; done
cd "$B" || exit 1
nice -n 19 c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O3 -std=c++26 \
	-include "$X/cmake/no_near_far.hpp" -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror \
	-o "$S/v4_probe" "$here/v4_probe.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo "probe rc=$?"
for l in base r4; do
	IN=""
	for d in components/rules components/d2s components/txt components/log; do IN="$IN -I$S/t_$l/$d"; done
	DEF=-DV4_R4; [ $l = base ] && DEF=-DV4_BASE
	nice -n 19 c++ $IN $DEF -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/v4_d2d_$l" "$here/v4_d2d.cpp"
	echo "built v4_d2d_$l rc=$?"
done
