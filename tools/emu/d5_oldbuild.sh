#!/bin/bash
# decomp5 (worker host): build test_game of an OLD upstream commit $1 with the --load-d2s mode, in its own tree and build dir
set -u
REV=$1; N=$2
T=$HOME/d2d-lab/wt-decomp5-$N; B=$HOME/d2d-lab/build-decomp5-$N; D=$HOME/d2d-lab/src/build/_deps
cd ~/d2d-lab/fp-scratch.git && git worktree add -f --detach $T $REV 2>&1 | tail -1
D5_TREE=$T python3 $HOME/labrun/herdr/code/d2lod-wt/decomp5/tools/emu/d5_patch_loadd2s.py || exit 1
mkdir -p $B; ln -sfn $B $T/build
nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j2 test_game > $B.build.log 2>&1
echo "build rc=$?"; grep -E "error" $B.build.log | head; tail -1 $B.build.log
