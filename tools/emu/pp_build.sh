#!/bin/bash
# prprep (worker host): configure + build test targets of a pp worktree.   pp_build.sh light|cold TARGET...
set -u
n=$1; shift
T=$HOME/d2d-lab/wt-pp-$n; B=$HOME/d2d-lab/build-pp-$n; D=$HOME/d2d-lab/src/build/_deps
mkdir -p $B; ln -sfn $B $T/build
[ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
git -C $T log --oneline -1
nice -n 19 ninja -C $B -j8 "$@" > $B.build.log 2>&1; rc=$?
echo "build rc=$rc"; grep -E "error|FAILED|unknown target" $B.build.log | head; tail -1 $B.build.log
exit $rc
