#!/bin/bash
# prpost (worker host): Lightning r4 and Cold r5 on current upstream main, verifier's second-pass edits.
export PATH=$HOME/.local/bin:$PATH
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
F=$HOME/d2d-lab/fork/src
cd $F; git fetch upstream 2>&1 | tail -1
MSG='grep -v "^(cherry picked from" | awk '"'"'/^Agent: Claude Code/ && prev != "" {print ""} {print; prev=$0}'"'"
fin() { # tree
  FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --msg-filter "$1" --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
  git log --format='%h %an %cn %s' -3 | cut -c1-100
}
# ---- Lightning r4
T=$HOME/d2d-lab/wt-pp-light4
[ -d $T ] || git worktree add -b fid/missile-last-unit-r4 $T upstream/main 2>&1 | tail -1
cd $T
$G cherry-pick f724f9ee 5effaf98 2>&1 | tail -1
sed -i 's|`T<case> <hits per monster>`. tools/emu/next_hit.py replays the F|`T<case> <hits per monster>`. tools/emu/light_pierce.py replays the F|' tests/game/test_light_pierce.cpp
grep -n "os\." tools/emu/light_pierce.py | head -3
sed -i '/^import os$/d' tools/emu/light_pierce.py
$G add -A tests tools; $G commit -q --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true $G rebase -q -i --autosquash upstream/main 2>&1 | tail -2
fin "$MSG"
git log -2 --format=%B | cat -A | grep -n "Agent" | head
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
# ---- Cold r5
cd $F
T=$HOME/d2d-lab/wt-pp-cold5
[ -d $T ] || git worktree add -b fid/cold-enchanted-length-r5 $T upstream/main 2>&1 | tail -1
cd $T
$G cherry-pick 4945679c 30713b60 2>&1 | tail -1
python3 - <<'PY'
p="components/rules/uniques.hpp"; s=open(p).read()
s=s.replace("most MonLvl's last row (0x5a1cf5..0x5a1d1c: 110 in 1.14d); ghostly's FUN_005a1080 adds 150.","most MonLvl's last row (0x5a1cf5..0x5a1d1c: max_level, the table's rows - 1); ghostly's FUN_005a1080 adds 150.")
s=s.replace("inline int boss_elem_length(const BossStats& stats, int level) { return stats.elem_len + stats.elem_len_per_level * std::clamp(level, 1, 110); }","inline int boss_elem_length(const BossStats& stats, int level, int max_level) { return stats.elem_len + stats.elem_len_per_level * std::clamp(level, 1, max_level); }")
open(p,"w").write(s)
p="components/game/ai.cpp"; s=open(p).read()
a="d2d::rules::boss_elem_length(boss, monster.stats.level)"; assert a in s
s=s.replace(a,"d2d::rules::boss_elem_length(boss, monster.stats.level, int(game_data.monsters.lvl.size()) - 1)")
open(p,"w").write(s)
p="tests/monsters/test_umod_stats.cpp"; s=open(p).read()
a="cold_len = boss_elem_length(boss, level);"; assert a in s
s=s.replace(a,"cold_len = boss_elem_length(boss, level, int(damage_rows.size()) - 1);")
open(p,"w").write(s)
p="tests/game/test_boss_cold_length.cpp"; s=open(p).read()
a="100 + 5 * std::min(monster.stats.level, 110)"; assert a in s
s=s.replace(a,"100 + 5 * std::clamp(monster.stats.level, 1, int(data->monsters.lvl.size()) - 1)")
open(p,"w").write(s)
PY
$G add -A components tests; $G commit -q --amend --no-edit
fin "$MSG | sed 's/test_boss_cold_length runs the real make_boss\./test_boss_cold_length runs the real make_boss. test_umod_stats switches from the bare elem_len to boss_elem_length (its d2d side must follow the rule it checks)./'"
git log -2 --format=%B | head -14
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
