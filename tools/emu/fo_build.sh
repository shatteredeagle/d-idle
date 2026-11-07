#!/bin/bash
# fidoracle (worker host): make test_game from the synced fork checkout FO_TREE at nice 19 with 2 jobs; touches the two sources so ninja redoes them.
set -u
T=${FO_TREE:?set FO_TREE to the synced scratch checkout of the fork branch}; B=${FO_BUILD:-$HOME/d2d-lab/build-fidoracle}
D=${FO_DEPS:-$HOME/d2d-lab/src/build/_deps}
mkdir -p $B
[ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
touch $T/components/game/world/setup.cpp $T/tests/game/test_game.cpp
nice -n 19 ninja -C $B -j2 test_game > $B.build.log 2>&1
echo "build rc=$?"; grep -E "error|FAILED" $B.build.log | head -20; tail -2 $B.build.log
