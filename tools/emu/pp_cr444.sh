#!/bin/bash
# prpost (worker host): CodeRabbit's second comment on the Lightning PR: assert the placed monsters are targetable; build, run seeds 1 and 7, commit on the PR branch (normal push after).
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-light4; B=$HOME/d2d-lab/build-pp-light4; O=$HOME/d2d-lab/pp-light4
cd $T; git checkout -q fid/missile-last-unit-r4
python3 - <<'PY'
p="tests/game/test_light_pierce.cpp"; s=open(p).read()
a="            monster.stats.hit_points = monster.hit_points = 1000000;"
assert s.count(a)==1
s=s.replace(a,"            assert(monster.targetable());\n"+a)
open(p,"w").write(s)
PY
git diff --stat | tail -1
nice -n 19 cmake --build $B -j8 --target test_light_pierce > /dev/null 2>&1; echo "build rc=$?"
cd tools/emu
for s in 1 7; do nice -n 19 $B/tests/test_light_pierce --light-hit $s 400 > $O/cr-$s.d 2>$O/cr.err; echo "seed $s rc=$? casts $(grep -c '^T' $O/cr-$s.d)"; uv run python light_pierce.py $O/cr-$s.d > $O/cr-$s.g 2>>$O/cr.err; grep -E '^[HT]' $O/cr-$s.d | diff $O/cr-$s.g - | wc -l; done
cd $T
$G add tests/game/test_light_pierce.cpp
$G commit -q -m "tests: Lightning on a pack asserts each placed monster is targetable

A size slot filled by a monster that is not targetable would record no hits on both sides and pass unseen.

Agent: Claude Code"
git log --format='%h %an %cn %s' -1 | cut -c1-100; git rev-parse HEAD
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
