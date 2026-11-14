#!/bin/bash
# prpost (worker host): Lightning r4: every commit builds, the differential on three seeds (parent and claim), neighbours, the verifier's two variants, ctest, presubmit.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-light4; B=$HOME/d2d-lab/build-pp-light4; O=$HOME/d2d-lab/pp-light4; mkdir -p $O
cd $T
BR=fid/missile-last-unit-r4
bash $HERE/pp_build.sh light4 mpq-cat | tail -1
full() { nice -n 19 cmake --build $B -j8 > $B.all.log 2>&1; echo "full build $(git log --format=%h -1) rc=$?"; }
run() { # label seed [extra env handled by caller]
  cd $T/tools/emu
  nice -n 19 $B/tests/test_light_pierce --light-hit $2 400 > $O/$1-$2.d 2>$O/err
  nice -n 19 uv run python light_pierce.py $O/$1-$2.d > $O/$1-$2.g 2>>$O/err
  grep -E '^[HT]' $O/$1-$2.d | diff $O/$1-$2.g - > $O/$1-$2.diff
  echo "$1 seed $2: casts $(grep -c '^T' $O/$1-$2.d) game $(grep -c '^T' $O/$1-$2.g); T lines differing one side $(grep -c '^< T' $O/$1-$2.diff); casts with any H/T line differing $(grep '^[<>] [HT]' $O/$1-$2.diff | sed 's/^[<>] [HT]\([0-9]*\).*/\1/' | sort -u | wc -l)"
  cd $T
}
git checkout -q $BR~1; full
for s in 1 7 13; do run parent $s; done
git checkout -q $BR; full
for s in 1 7 13; do run claim $s; done
cd $T/tools/emu; nice -n 19 $B/tests/test_next_hit --next-hit 1 400 > $O/nh.d 2>/dev/null; uv run python next_hit.py $O/nh.d > $O/nh.g; echo "next_hit differing lines: $(grep -E '^[HT]' $O/nh.d | diff $O/nh.g - | wc -l) of $(grep -c '^T' $O/nh.d)"
cd $T
for t in test_game_missiles test_game test_monsters test_combat; do timeout 900 $B/tests/$t >/tmp/pp4_$t.out 2>&1; echo "$t rc=$?"; done
# ---- the verifier's two variants (throwaway, never committed)
var() { # rev tag
  git checkout -q --detach $1
  python3 $HOME/d2d-lab/pv/patch_test.py $T/tests/game/test_light_pierce.cpp || { echo "PATCH FAILED"; return 1; }
  sed -i 's/names = ("lightningbolt",)/names = ("lightningbolt", "bonespear", "furylightning", "arcticblast1", "infernoflame1")/' $T/tools/emu/light_pierce.py
  nice -n 19 cmake --build $B -j8 --target test_light_pierce > /dev/null 2>&1 || echo "VAR BUILD FAILED"
  for s in 1 7; do PV_PATHORDER=1 run $2-path $s; done
  for sk in "Bone Spear:bonespear" "Lightning Fury:furylightning" "Arctic Blast:arcticblast1" "Inferno:infernoflame1"; do
    PV_SKILL="${sk%%:*}" PV_ROW="${sk##*:}" run $2-${sk##*:} 1; done
  git checkout -q -- tests/game/test_light_pierce.cpp tools/emu/light_pierce.py
}
var $BR~1 parentvar
var $BR claimvar
git checkout -q $BR; git status --short | head -3; git log --format=%h -1
nice -n 19 cmake --build $B -j8 > $B.all.log 2>&1; echo "rebuild clean rc=$?"
cd $B && timeout 2400 ctest -j8 --output-on-failure > $B.ctest.log 2>&1; echo "ctest rc=$?"
grep -E "tests passed|tests failed" $B.ctest.log; grep -E "^\s+[0-9]+ - " $B.ctest.log | head -8
cd $T
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $O/presubmit.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $O/presubmit.txt | tail -8 | cut -c1-250
git rev-parse HEAD
