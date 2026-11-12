#!/bin/bash
# Zeal-gap harness (worker host): configure and build the libraries of a d2d worktree, then compile zd_zeal_test against them.
#   zd_tree.sh WORKTREE BUILD_DIR OUT_BINARY     (WORKTREE = a d2d checkout, e.g. ~/d2d-lab/wt-przeal)
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${1:?worktree}; B=${2:?build dir}; OUT=${3:?output binary}
D=$HOME/d2d-lab/src/build/_deps
[ -f $B/build.ninja ] || nice -n 19 cmake -S $SRC -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DD2D_BUILD_APPS=OFF -DD2D_BUILD_TOOLS=OFF \
	-DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src -DD2D_BUNDLED_STORMLIB=OFF > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j4 game > $B.build.log 2>&1 || { tail -20 $B.build.log; exit 1; }
mkdir -p "$(dirname "$OUT")"
SRC=$SRC D2_SCRATCH=$(dirname "$OUT") "$here/zd_build.sh" "$B"
mv "$(dirname "$OUT")/zd_zeal_test" "$OUT"
echo built "$OUT"
