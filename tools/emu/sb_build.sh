#!/bin/bash
# smiteblock (the worker host): compile tools/emu/sb_d2d_test.cpp by hand against the libraries of a scratch upstream build
# (as pal2_build.sh), then run it: sb_build.sh CASES SEED. -j1, nice.
#   tools/emu/run-on-worker.sh --ttl 1500 sb_build.sh 20000 7
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${B:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:-/tmp/d2agent-smiteblock}
mkdir -p "$OUT"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/sb_d2d_test" "$here/sb_d2d_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built "$OUT/sb_d2d_test"
"$OUT/sb_d2d_test" "${1:-20000}" "${2:-7}"
grep -n "player_blow" "$SRC/components/rules/combat.hpp" | head -2
sed -n '299,304p' "$SRC/components/rules/combat.hpp"
