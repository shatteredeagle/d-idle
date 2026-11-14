#!/bin/bash
# prpost (worker host): Cold Enchanted r5: every commit builds, the differential on two seeds (parent and claim), tests, ctest, presubmit.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold5; B=$HOME/d2d-lab/build-pp-cold5; O=$HOME/d2d-lab/pp-cold5; mkdir -p $O
cd $T
BR=fid/cold-enchanted-length-r5
bash $HERE/pp_build.sh cold5 mpq-cat | tail -1
full() { nice -n 19 cmake --build $B -j8 > $B.all.log 2>&1; echo "full build $(git log --format=%h -1) rc=$?"; }
for seed in 1 9; do n=100000; [ $seed = 9 ] && n=50000; cd $T/tools/emu; [ -s $O/game-$seed.tsv ] || uv run python umod_stats.py $n $seed > $O/game-$seed.tsv 2>$O/game-$seed.err; done
cmp() { # label seed
  cd $T/tools/emu; nice -n 19 $B/tests/test_umod_stats --umod-stats $O/game-$2.tsv > $O/$1-$2.d2d && uv run python umod_stats.py --compare $O/game-$2.tsv $O/$1-$2.d2d 2>&1 | grep -v WARNING > $O/$1-$2.full
  echo "$1 seed $2: $(head -1 $O/$1-$2.full)"; python3 $HERE/pp_cold_count.py $O/$1-$2.full; cd $T
}
git checkout -q $BR~1; full
for s in 1 9; do cmp parent $s; done
git checkout -q $BR; full
for s in 1 9; do cmp claim $s; done
for t in test_boss_cold_length test_monsters test_game; do timeout 900 $B/tests/$t >/tmp/pp5_$t.out 2>&1; echo "$t rc=$?"; done
cd $B && timeout 2400 ctest -j8 --output-on-failure > $B.ctest.log 2>&1; echo "ctest rc=$?"
grep -E "tests passed|tests failed" $B.ctest.log; grep -E "^\s+[0-9]+ - " $B.ctest.log | head -8
cd $T
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $O/presubmit.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $O/presubmit.txt | tail -8 | cut -c1-250
git rev-parse HEAD
