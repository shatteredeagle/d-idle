#!/bin/bash
# decomp12 (worker host): compile d12_arrive_probe.cpp by hand against build-decomp12 (as vb2_probe_build.sh). Needs d12_setup.sh first.
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=$HOME/d2d-lab/wt-decomp12; B=$HOME/d2d-lab/build-decomp12; OUT=$HOME/d2d-lab/d12; mkdir -p $OUT
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice -n 19 c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/d12_arrive_probe" "$here/d12_arrive_probe.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built $OUT/d12_arrive_probe
