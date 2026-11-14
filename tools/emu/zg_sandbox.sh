#!/bin/bash
# Zeal-gap (worker host): build a fork worktree and run test_zeal_chain and the Zeal harness against it. One heavy job.
#   zg_sandbox.sh WORKTREE BUILD_DIR
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${1:?worktree}; B=${2:?build dir}
D=$HOME/d2d-lab/src/build/_deps
[ -f $B/build.ninja ] || nice -n 19 cmake -S $SRC -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
	-DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src -DD2D_BUNDLED_STORMLIB=OFF > $B.cmake.log 2>&1
nice -n 19 ninja -C $B -j4 test_zeal_chain test_game > $B.build.log 2>&1 || { tail -20 $B.build.log; exit 1; }
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
find $B -name test_zeal_chain -type f -perm -u+x -exec {} \; ; echo "test_zeal_chain rc=$?"
find $B -name test_game -type f -perm -u+x -exec {} \; ; echo "test_game rc=$?"
