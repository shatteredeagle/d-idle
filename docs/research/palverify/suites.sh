#!/bin/bash
# suites.sh: build and run the upstream test targets that cover the changed code, base and head (agent palverify).
R=$HOME/d2d-lab/palverify; export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
T="test_combat test_rules test_game_rules test_game test_missile_hit test_monsters"
for w in base head; do
  t=""; for x in $T; do t="$t tests/$x"; done
  [ $w = head ] && t="$t tests/test_hammer_hit"
  nice ninja -C $R/build-$w -j4 $t > $R/suites-$w.build.log 2>&1; echo "$w build rc=$?" >> $R/suites.log
  for x in $T $([ $w = head ] && echo test_hammer_hit); do
    [ -x $R/build-$w/tests/$x ] || { echo "$w $x missing" >> $R/suites.log; continue; }
    (cd $R/$w && timeout 600 nice $R/build-$w/tests/$x > $R/suites-$w-$x.out 2>&1); echo "$w $x rc=$? $(tail -1 $R/suites-$w-$x.out | cut -c1-100)" >> $R/suites.log
  done
done
echo done >> $R/suites.log
