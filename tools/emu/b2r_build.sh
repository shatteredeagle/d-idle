#!/bin/bash
# b2rebase: see docs/research/pr-batch2-body.md; run on the worker host with labrun (hard-coded lab paths below, as the other lab scripts).
# bundle build + suites: configure once, -j1 nice 19 ccache
set -u
T=$HOME/d2d-lab/wt-b2bundle-r2; B=$HOME/d2d-lab/build-b2bundle-r2
D=$HOME/d2d-lab/src/build/_deps
SUITES="test_rules test_game_rules test_monsters test_monster_thinks test_monster_ai test_combat test_game test_skills test_sk_ground test_sk_mana test_tele_spot test_monster_missile test_missile_blow_poison"
mkdir -p $B
[ -f $B/build.ninja ] || nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=ON \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src -DCMAKE_EXPORT_COMPILE_COMMANDS=ON > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j1 $SUITES > $B.build.log 2>&1
echo "build rc=$?"; grep -E "error|FAILED" $B.build.log | head -20; tail -2 $B.build.log
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
for t in $SUITES; do
  f=$(ls $B/tests/$t $B/tests/*/$t 2>/dev/null | head -1)
  (cd $B && timeout 1500 nice -n 19 $f > $B.run-$t.log 2>&1); echo "$t rc=$? $(tail -1 $B.run-$t.log | cut -c1-100)"
done
