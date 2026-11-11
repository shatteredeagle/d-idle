#!/bin/bash
# prprep2 (worker host): Cold Enchanted r4: each commit builds, the differential on the parent (oracle commit) and the claim, seeds 1 and 9.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold4; B=$HOME/d2d-lab/build-pp-cold4; O=$HOME/d2d-lab/pp-cold4; mkdir -p $O
cd $T
for seed in 1 9; do n=100000; [ $seed = 9 ] && n=50000; cd $T/tools/emu; [ -s $O/game-$seed.tsv ] || uv run python umod_stats.py $n $seed > $O/game-$seed.tsv 2>/dev/null; done
cmp() { # label seed
  cd $T/tools/emu; $B/tests/test_umod_stats --umod-stats $O/game-$2.tsv > $O/$1-$2.d2d && uv run python umod_stats.py --compare $O/game-$2.tsv $O/$1-$2.d2d 2>&1 | grep -v WARNING > $O/$1-$2.full
  echo "$1 seed $2: $(head -1 $O/$1-$2.full)"; python3 $HERE/pp_cold_count.py $O/$1-$2.full
}
cd $T; git checkout -q HEAD~1; echo "oracle commit alone:"; git log --format=%h -1
bash $HERE/pp_build.sh cold4 test_umod_stats mpq-cat | tail -1
for s in 1 9; do cmp parent $s; done
git checkout -q fid/cold-enchanted-length-r4; echo "claim:"
bash $HERE/pp_build.sh cold4 test_umod_stats test_boss_cold_length test_monsters test_game | tail -1
for s in 1 9; do cmp claim $s; done
for t in test_boss_cold_length test_monsters test_game; do timeout 900 $B/tests/$t >/tmp/pp4_$t.out 2>&1; echo "$t rc=$?"; done
git rev-parse HEAD
