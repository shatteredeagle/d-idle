#!/bin/bash
# prprep2 (worker host): Lightning r3: the differential on three seeds with the claim and on the parent (oracle commit), then the neighbours.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-light3; B=$HOME/d2d-lab/build-pp-light3; O=$HOME/d2d-lab/pp-light3; mkdir -p $O
run() { # label seed
  cd $T/tools/emu
  $B/tests/test_light_pierce --light-hit $2 400 > $O/$1-$2.d 2>$O/err; uv run python light_pierce.py $O/$1-$2.d > $O/$1-$2.g 2>>$O/err
  grep -E '^[HT]' $O/$1-$2.d | diff $O/$1-$2.g - > $O/$1-$2.diff
  echo "$1 seed $2: casts $(grep -c '^T' $O/$1-$2.d) game $(grep -c '^T' $O/$1-$2.g) T lines differing $(grep -c '^[<>] T' $O/$1-$2.diff) all lines $(grep -c '^[<>]' $O/$1-$2.diff)"
}
cd $T; git log --format='%h %s' -2 | cut -c1-80
bash $HERE/pp_build.sh light3 test_light_pierce test_next_hit test_game_missiles test_game test_monsters test_combat mpq-cat | tail -1
for s in 1 7 13; do run claim $s; done
cd $B/../wt-pp-light3/tools/emu; $B/tests/test_next_hit --next-hit 1 400 > $O/nh.d 2>/dev/null; uv run python next_hit.py $O/nh.d > $O/nh.g; echo "next_hit differing lines: $(grep -E '^[HT]' $O/nh.d | diff $O/nh.g - | wc -l) of $(grep -c '^T' $O/nh.d)"
cd $T; git checkout -q HEAD~1 && bash $HERE/pp_build.sh light3 test_light_pierce | tail -1
for s in 1 7 13; do run parent $s; done
git checkout -q fid/missile-last-unit-r3; bash $HERE/pp_build.sh light3 test_light_pierce test_game_missiles test_game test_monsters test_combat | tail -1
for t in test_game_missiles test_game test_monsters test_combat; do timeout 900 $B/tests/$t >/tmp/pp3_$t.out 2>&1; echo "$t rc=$?"; done
git rev-parse HEAD
