#!/bin/bash
# b2bundle: see docs/research/pr-batch2-body.md; run on the worker host with labrun (hard-coded lab paths below, as the other lab scripts).
# measurement trees: base = 3e073888 + the bundle's two test files; d1 = base + the Teleport D1 commit + the test file. Build only the two diff binaries.
set -u
R=$HOME/d2d-lab/wt-b2bundle; D=$HOME/d2d-lab/src/build/_deps
mk() { # name ref
  T=$HOME/d2d-lab/wt-b2bundle-$1; B=$HOME/d2d-lab/build-b2bundle-$1
  [ -d $T ] || git -C $R worktree add -q --detach $T $2 || return 1
  for f in tests/game/test_sk_mana.cpp tests/game/test_tele_spot.cpp; do git -C $R show local/batch2-bundle:$f > $T/$f; done
  mkdir -p $B
  [ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=ON \
   -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
   -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
   -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -5 $B.cmake.log; return 1; }
  nice -n 19 ninja -C $B -j1 test_sk_mana test_tele_spot > $B.build.log 2>&1; echo "$1 build rc=$?"; grep -E "error" $B.build.log | head -5
}
mk base 3e073888
mk d1 $(git -C $R rev-parse local/batch2-bundle~2)
