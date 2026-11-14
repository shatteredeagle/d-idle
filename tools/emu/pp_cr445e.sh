#!/bin/bash
# prpost2 (worker host): resolve the README conflict of the review-fix cherry-pick (the merged-in main already has the Lightning row), continue, re-run the gates.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold5; B=$HOME/d2d-lab/build-pp-cold5; O=$HOME/d2d-lab/pp-cold5
cd $T
python3 - <<'PY'
p="tools/emu/README.md"; L=open(p).read().split("\n")
out=[]; i=0
while i < len(L):
    if L[i].startswith("<<<<<<< "):
        j=L.index("=======",i); k=next(x for x in range(j,len(L)) if L[x].startswith(">>>>>>> "))
        head=L[i+1:j]; mine=L[j+1:k]
        keep=[h for h in head if "test_umod_stats" not in h]      # drop the old umod row, keep the Lightning row main now has
        out += mine + keep
        i=k+1
    else:
        out.append(L[i]); i+=1
open(p,"w").write("\n".join(out))
PY
grep -c '^<<<<<<<\|^>>>>>>>' tools/emu/README.md
grep -n "umod_stats.py 100000\|light_pierce --light-hit" tools/emu/README.md | cut -c1-40
git add tools/emu/README.md
GIT_EDITOR=true $G cherry-pick --continue 2>&1 | tail -1
$G cherry-pick 2293437f 2>&1 | tail -1
git status --short | head -3
git log --format='%h %an %s' -5 | cut -c1-90
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
grep -E '^(FAIL)' $O/presubmit4.txt | head -5 | cut -c1-250; grep -E 'umod|boss_cold|uniques|ai.cpp' $O/presubmit4.txt | grep -v '^ok' | head -5 | cut -c1-200
git rev-parse HEAD
