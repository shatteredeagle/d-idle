#!/bin/bash
# prprep (worker host): commit the glue test with the Cold Enchanted claim, run the neighbours, presubmit, push.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold; B=$HOME/d2d-lab/build-pp-cold; O=$HOME/d2d-lab/pp-cold
cd $T
git add tests/game/test_boss_cold_length.cpp
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q --amend -m "Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)

A Cold Enchanted unique's cold rode the blow with no length, so the chill was one frame; game.exe adds the unit's level x 5 + 100 to the cold length. test_boss_cold_length runs the real make_boss.
Agent: Claude Code"
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --msg-filter 'grep -v "^(cherry picked from"' --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log -3 --format='%h %an %cn | %s'
bash $HERE/pp_build.sh cold test_boss_cold_length test_boss_level test_umod_stats test_monsters test_game mpq-cat | tail -2
for t in test_boss_cold_length test_monsters test_game; do timeout 900 $B/tests/$t >/tmp/pp_$t.out 2>&1; echo "$t rc=$?"; done
cd tools/emu
uv run python boss_level.py gen 100000 20261108 > $O/bl.tsv && uv run python boss_level.py game $O/bl.tsv > $O/bl.g && $B/tests/test_boss_level --boss-level $O/bl.tsv > $O/bl.d && uv run python boss_level.py compare $O/bl.tsv $O/bl.g $O/bl.d 2>&1 | grep -v WARNING | tail -4
cd $T
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -vE '^ok' | tail -12
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -2
git push forgejo fid/cold-enchanted-length-r1 2>&1 | tail -1; git rev-parse HEAD
