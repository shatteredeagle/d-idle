#!/bin/bash
# prprep2 (worker host): Cold Enchanted r4: rebase on current upstream main, the oracle commit builds on its own, level clamp [1, 110] with its source.
export PATH=$HOME/.local/bin:$PATH
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
F=$HOME/d2d-lab/fork/src; T=$HOME/d2d-lab/wt-pp-cold4
cd $F
[ -d $T ] || git worktree add -b fid/cold-enchanted-length-r4 $T upstream/main 2>&1 | tail -1
cd $T
$G cherry-pick 511d907d 2>&1 | tail -1
grep -n "boss_elem_length" tests/monsters/test_umod_stats.cpp
sed -i 's/cold_len = boss_elem_length(boss, level);/cold_len = boss.elem_len;/' tests/monsters/test_umod_stats.cpp
$G add tests/monsters/test_umod_stats.cpp; $G commit -q --amend --no-edit
$G cherry-pick 1b4036a3 2>&1 | tail -1
sed -i 's/cold_len = boss.elem_len;/cold_len = boss_elem_length(boss, level);/' tests/monsters/test_umod_stats.cpp
python3 - <<'PY'
p="components/rules/uniques.hpp"; s=open(p).read()
i=s.index("// The cold length a boss's blow carries"); j=s.index("inline int boss_elem_length")
k=s.index("\n",j)
new="""// The cold length a boss's blow carries (frames): FUN_005a1c70 adds the unit's level x 5 + 100 (0x5a1dcd), the level taken as at least 1 and at
// most MonLvl's last row (0x5a1cf5..0x5a1d1c: 110 in 1.14d); ghostly's FUN_005a1080 adds 150.
inline int boss_elem_length(const BossStats& stats, int level) { return stats.elem_len + stats.elem_len_per_level * std::clamp(level, 1, 110); }"""
s=s[:i]+new+s[k:]
open(p,"w").write(s)
PY
grep -n "boss_elem_length" -B3 components/rules/uniques.hpp | head -12
git add -A components tests; $G commit -q --amend --no-edit
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --msg-filter 'grep -v "^(cherry picked from"' --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -3 | cut -c1-110
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -2
git status --short | head -3
