#!/bin/sh
# fidrevive (worker host): see docs/research/fid-revive-refill.md section 6; run through `labrun orion -- sh <this>` from the synced scratch clone of the fork branch. Needs T/.parent_setup.cpp = `git show up/main:components/game/fight/setup.cpp` of the scratch clone, and FR_DIFF = the path of fr_diff.sh.
export D2_BUILD=$HOME/d2d-lab/src/build D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe D2_MPQ_DIR=/tmp/ug-emu
T=$HOME/labrun/herdr/scratch-fidrevive; B=$HOME/d2d-lab/build-fidrevive; O=/tmp/fr_out
cp $T/components/game/fight/setup.cpp $O/fixed_setup.cpp
cp $T/.parent_setup.cpp $T/components/game/fight/setup.cpp
nice -n 19 ninja -C $B -j2 test_game > $B.base.log 2>&1; echo "parent build rc=$?"
sh ${FR_DIFF:?path of fr_diff.sh} before
(cd $B && nice -n 19 ./tests/test_game > $O/full.before.log 2>&1; echo "full test_game on parent rc=$?"; tail -3 $O/full.before.log)
cp $O/fixed_setup.cpp $T/components/game/fight/setup.cpp
nice -n 19 ninja -C $B -j2 test_game > $B.build2.log 2>&1; echo "fixed rebuild rc=$?"
(cd $B && nice -n 19 ./tests/test_game > $O/full.after.log 2>&1; echo "full test_game fixed rc=$?"; grep -c '^OK' $O/full.after.log; grep "respawn refills" $O/full.after.log)
sh ${FR_DIFF:?path of fr_diff.sh} after2
