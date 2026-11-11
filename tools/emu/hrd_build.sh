#!/bin/bash
# hirereader (the worker host): compile tools/emu/hrd_d2d_test.cpp by hand against the libraries of a scratch upstream
# build (same recipe as mst_build.sh; the purse rules are the same there).
#   tools/emu/run-on-worker.sh --ttl 1500 hrd_build.sh > /tmp/d2agent-hirereader/build.txt 2>&1
# -> $D2_SCRATCH/hrd_d2d_test
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${1:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:-/tmp/d2agent-hirereader}
mkdir -p $OUT
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o $OUT/hrd_d2d_test $here/hrd_d2d_test.cpp components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built $OUT/hrd_d2d_test
