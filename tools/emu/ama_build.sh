#!/bin/bash
# skamazon (the worker host): hand-compile the d2d side of the Amazon differentials
# against scratch upstream builds. Run through the wrapper (ttl 1500):
#   tools/emu/run-on-worker.sh --ttl 1500 ama_build.sh
# Outputs under $D2_SCRATCH: ama_test (nec_test.cpp: --calc, generic by skill id),
# hit_before (sk_missile_hit_test.cpp on the skmana build = upstream main 72132b99),
# hit_after (same source on the skmiss build = with the SKMISS fix 23e2baec).
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:-/tmp/d2agent-skamazon}
mkdir -p "$OUT"
build() {  # SRC BUILD_DIR TEST_CPP OUT_BIN
  local SRC=$1 B=$2 CPP=$3 BIN=$4 IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
  (cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$BIN" "$CPP" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
  echo built "$BIN"
}
build $HOME/d2d-lab/pr/palskill $HOME/d2d-lab/pr/skmana-out/build $HERE/nec_test.cpp "$OUT/ama_test"
build $HOME/d2d-lab/pr/palskill $HOME/d2d-lab/pr/skmana-out/build $HERE/sk_missile_hit_test.cpp "$OUT/hit_before"
build $HOME/d2d-lab/pr/skmiss $HOME/d2d-lab/pr/skmiss-out/build $HERE/sk_missile_hit_test.cpp "$OUT/hit_after"
