#!/bin/bash
# stateev_diff (the worker host): link tools/emu/stateev_diff_probe.cpp against the libraries of a scratch upstream
# build (default: the sorcrest scratch build; SRC = its source tree), same flags as ms_build.sh. Output: $D2_SCRATCH/stateev_probe.
#   tools/emu/run-on-worker.sh --ttl 1500 stateev_diff_build.sh > /tmp/d2agent-pa-stateevdiff/build.txt 2>&1
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${1:-$HOME/d2d-lab/pr/sorcrest-out/build}
S=${D2_SCRATCH:?}
git -C "$SRC" log --oneline -1 2>/dev/null || echo "(source tree is not a git checkout)"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice -n 19 c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$S/stateev_probe" "$here/stateev_diff_probe.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built "$S/stateev_probe"
