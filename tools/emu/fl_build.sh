#!/bin/bash
# fidfix1: build test_game from the pilot branch (private fork branch pilot/load-life-mana) and run it.
# Run on the worker host with labrun (lab paths hard-coded as in b2b_build.sh). Pass BASE=1 to build the branch's parent instead (test must fail there).
set -u
T=$HOME/d2d-lab/fidfix1-pilot; B=$HOME/d2d-lab/build-fidfix1
[ "${BASE:-0}" = 1 ] && { git -C $T checkout -q HEAD~1 -- components/game/world.cpp; B=${B}-base; }
D=$HOME/d2d-lab/src/build/_deps
mkdir -p $B
[ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j2 test_game > $B.build.log 2>&1
echo "build rc=$?"; grep -E "error|FAILED" $B.build.log | head -20; tail -2 $B.build.log
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
f=$(ls $B/tests/test_game $B/tests/*/test_game 2>/dev/null | head -1)
(cd $B && timeout 1500 nice -n 19 $f > $B.run.log 2>&1); echo "test_game rc=$?"; grep -E "OK: a save|Assertion|assert" $B.run.log | head; tail -1 $B.run.log | cut -c1-120
