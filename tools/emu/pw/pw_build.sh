#!/bin/bash
# prwarp (worker host): pw_build.sh REV [TARGET...]  check REV out in wt-pw, build into build-pw (all targets when none given)
set -u
export PATH=$HOME/.local/bin:$PATH
rev=$1; shift
T=$HOME/d2d-lab/wt-pw; B=$HOME/d2d-lab/build-pw; D=$HOME/d2d-lab/src/build/_deps
git -C $T checkout -q $rev || exit 1
mkdir -p $B; ln -sfn $B $T/build
[ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j10 "$@" > $B.build.log 2>&1; rc=$?
echo "BUILD $(git -C $T log --format='%h %s' -1 | cut -c1-60) targets=[${*:-all}] rc=$rc"
[ $rc = 0 ] || grep -E "error|FAILED" $B.build.log | head -12
exit $rc
