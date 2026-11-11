#!/bin/bash
# covpal (the worker host): hand-compile tools/emu/pal_test.cpp and pal2_test.cpp against the libraries of a scratch
# upstream build into $D2_SCRATCH. Same method as pal2_build.sh; nice, one compile at a time.
#   tools/emu/run-on-worker.sh --ttl 1500 covpal_build.sh > /tmp/d2agent-covpal/build.txt 2>&1
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${1:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:?run through tools/emu/run-on-worker.sh}
mkdir -p "$OUT"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
for t in pal_test pal2_test; do
  nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/$t" "$here/$t.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
  echo built "$OUT/$t"
done
