#!/bin/bash
# prpost2 (worker host): the maintainer merged main into the PR branch; put the three review-fix commits on top of that head, re-run, presubmit.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold5; B=$HOME/d2d-lab/build-pp-cold5; O=$HOME/d2d-lab/pp-cold5
cd $T
git fetch ghfork fid/cold-enchanted-length-r5 2>&1 | tail -1
git branch -f pr445-fixes HEAD
git checkout -q -B fid/cold-enchanted-length-r5 FETCH_HEAD
git log --format='%h %an %s' -2 | cut -c1-90
$G cherry-pick 15d0f82e 79474d0c 2293437f 2>&1 | tail -1
git log --format='%h %an %s' -4 | cut -c1-90
nice -n 19 cmake --build $B -j8 > $B.all.log 2>&1; echo "full build rc=$?"
cd tools/emu
for seed in 1 9; do
  $B/tests/test_umod_stats --umod-stats $O/game-$seed.tsv > $O/m-$seed.d2d && uv run python umod_stats.py --compare $O/game-$seed.tsv $O/m-$seed.d2d 2>&1 | grep -v WARNING > $O/m-$seed.full
  python3 $HERE/pp_cold_count.py $O/m-$seed.full
done
cd $T
for t in test_boss_cold_length test_monsters test_game; do timeout 900 $B/tests/$t >/tmp/pp5c_$t.out 2>&1; echo "$t rc=$?"; done
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $O/presubmit4.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $O/presubmit4.txt | tail -6 | cut -c1-250
git rev-parse HEAD
