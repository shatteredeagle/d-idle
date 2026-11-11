#!/bin/bash
# verifyb2 (the worker host): hand-compile a scratch probe against one tree built by vb2_build.sh.
#   vb2_probe_build.sh NAME PROBE   (PROBE: tools/emu/PROBE.cpp) -> ~/d2d-lab/vb2/bin/PROBE-NAME
set -e
here=$(cd "$(dirname "$0")" && pwd)
NAME=$1; P=$2
V=$HOME/d2d-lab/vb2
SRC=$V/wt-$NAME; B=$V/build-$NAME; OUT=$V/bin
mkdir -p "$OUT"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice -n 19 c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/$P-$NAME" "$here/$P.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built "$OUT/$P-$NAME"
