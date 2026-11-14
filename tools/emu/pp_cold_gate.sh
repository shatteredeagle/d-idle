#!/bin/bash
# prprep (worker host): gate test_boss_cold_length on the patched data (presubmit data-gating WARN), rebuild, run, push both candidate branches under new names.
PRIV=${PRIV:?name of the private remote}
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
T=$HOME/d2d-lab/wt-pp-cold; B=$HOME/d2d-lab/build-pp-cold; O=$HOME/d2d-lab/pp-cold
cd $T
sed -i 's|^    assert(data);$|    assert(data);\n    if (!data->patched) { std::printf("SKIP: the oracle is 1.14d: set D2_PATCH_INSTALLER\\n"); return D2D_TEST_SKIPPED; }|' tests/game/test_boss_cold_length.cpp
grep -n "patched" tests/game/test_boss_cold_length.cpp
$G add tests/game/test_boss_cold_length.cpp; $G commit -q --fixup=HEAD
GIT_SEQUENCE_EDITOR=true $G rebase -q -i --autosquash upstream/main 2>&1 | tail -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -2 | cut -c1-100
bash $HERE/pp_build.sh cold test_boss_cold_length | tail -1
$B/tests/test_boss_cold_length 2>&1 | tail -1; echo "rc=${PIPESTATUS[0]}"
python3 $HERE/pp_cold_count.py $O/before.full $O/final.full
cd $T/tools/emu; $B/tests/test_umod_stats --umod-stats $O/game-1-100000.tsv > $O/final.d2d && uv run python umod_stats.py --compare $O/game-1-100000.tsv $O/final.d2d 2>&1 | grep -v WARNING > $O/final.full; python3 $HERE/pp_cold_count.py $O/before.full $O/final.full
cd $T; python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
git push "$PRIV" HEAD:refs/heads/fid/cold-enchanted-length-r3 2>&1 | tail -1; git rev-parse HEAD
cd $HOME/d2d-lab/wt-pp-light; git push "$PRIV" HEAD:refs/heads/fid/missile-last-unit-r2 2>&1 | tail -1; git rev-parse HEAD
