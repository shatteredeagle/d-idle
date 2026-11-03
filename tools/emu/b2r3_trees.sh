#!/bin/bash
# b2rebase2: see docs/research/pr-batch2-body.md; run on the worker host with labrun (hard-coded lab paths, as the other lab scripts).
# measurement trees on upstream main 005c7cab: base = main + the bundle's two test files; d1 = base + the Teleport D1 commit (bundle head~2) + the test file.
# Builds only the two diff binaries (test_sk_mana, test_tele_spot).
set -u
R=$HOME/d2d-lab/fork/wt-b2r3; D=$HOME/d2d-lab/src/build/_deps; BR=local/batch2-bundle-r3
mk() { # name ref
  T=$HOME/d2d-lab/wt-b2r3-$1; B=$HOME/d2d-lab/build-b2r3-$1
  [ -d $T ] || git -C $R worktree add -q --detach $T $2 || return 1
  for f in tests/game/test_sk_mana.cpp tests/game/test_tele_spot.cpp; do git -C $R show $BR:$f > $T/$f; done
  mkdir -p $B
  [ -f $B/build.ninja ] || nice -n 10 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=ON \
   -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
   -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
   -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -5 $B.cmake.log; return 1; }
  nice -n 10 ninja -C $B -j4 test_sk_mana test_tele_spot > $B.build.log 2>&1; echo "$1 build rc=$?"; grep -E "error" $B.build.log | head -5
}
mk base 005c7cab
mk d1 $(git -C $R rev-parse $BR~2)
