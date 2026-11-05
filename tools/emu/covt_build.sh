#!/bin/bash
# covtreasure2 (the worker host): compile tools/emu/covt_xp_test.cpp by hand against the libraries of the scratch
# upstream build used by ms_build.sh (sorcrest checkout; its combat.hpp holds the fixed kill_award).
#   tools/emu/run-on-worker.sh --ttl 1500 covt_build.sh  ->  $D2_SCRATCH/covt_xp_test
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${1:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:?}
mkdir -p "$OUT"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
for t in covt_xp_test covt_regen_test; do
  nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o $OUT/$t $here/$t.cpp components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
  echo built $OUT/$t
done
