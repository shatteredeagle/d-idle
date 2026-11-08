#!/bin/sh
# fidrevive (worker host): see docs/research/fid-revive-refill.md section 6; run through `labrun orion -- sh <this>` from the synced scratch clone of the fork branch.
export FO_TREE=$HOME/labrun/herdr/scratch-fidrevive FO_BUILD=$HOME/d2d-lab/build-fidrevive
mkdir -p $FO_BUILD
T=$FO_TREE; B=$FO_BUILD; D=$HOME/d2d-lab/src/build/_deps
[ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j2 test_game > $B.build.log 2>&1
echo "build rc=$?"; grep -E "error|FAILED" $B.build.log | head -20; tail -2 $B.build.log
