#!/bin/bash
# decomp6 (worker host): rebuild fid/cold-enchanted-length cleanly: commit 1 (differential, short names fixed), commit 2 (the claim)
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
cd $T
git rebase --abort 2>/dev/null
git checkout -q -f -B fid/cold-enchanted-length upstream/main || exit 1
python3 $HERE/sk6_cold_tool.py $HERE $T
f=tests/monsters/test_umod_stats.cpp
sed -i -E 's/std::istringstream in\(line\)/std::istringstream fields(line)/; s/\bin >>/fields >>/g; s/\bdm\b/damage_rows/g; s/\bel\b/element_rows/g' $f
sed -i -E 's/std::istringstream m\(mods_s\); std::string id; while \(std::getline\(m, id, /std::istringstream parts(mods_s); std::string mod; while (std::getline(parts, mod, /; s/mods.push_back\(std::stoi\(id\)\)/mods.push_back(std::stoi(mod))/' $f
sed -i 's/FUN_005a17e0 strong, FUN_005a1910 fast,/FUN_005a17e0 strong, the fast init,/' tools/emu/umod_stats.py
grep -n -E "\bin\b|\bel\b|\bm\b|\bid\b|\bdm\b" $f | head -5
python3 - <<'PY'
p="tools/emu/README.md"
s=open(p).read()
row="| A unique's, champion's and minion's spawn stats (FUN_005a2120 and the mods' inits: life, level, experience, damage, speed, resists, enchanted damage and cold length, mana burn) | `uv run python umod_stats.py 100000 1 > game.tsv; ../../build/tests/test_umod_stats --umod-stats game.tsv > d2d.tsv; uv run python umod_stats.py --compare game.tsv d2d.tsv` | the fields d2d models: `cases N`, then each field that differs (a minion has no mods in d2d: its rows differ) |\n"
key="| A Blessed Hammer hit"
assert key in s
s=s.replace(key,row+key,1)
open(p,"w").write(s)
PY
ID=(-c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com)
git add tools/emu/umod_stats.py tests/monsters/test_umod_stats.cpp tools/emu/README.md
git "${ID[@]}" commit -q -F - <<MSG
tests: the unique's mods, d2d against game.exe's own inits

tools/emu/umod_stats.py runs FUN_005a2120 and the mods' inits on one unit; test_umod_stats is d2d's boss_stats on the same cases.
Agent: Claude Code
MSG
cd $B && cmake . >/dev/null 2>&1; nice -n 19 ninja -j2 test_umod_stats 2>&1 | tail -1
$B/tests/test_umod_stats --umod-stats $O/umod/game.tsv > $O/umod/before3.tsv; cmp $O/umod/before.tsv $O/umod/before3.tsv && echo "BEFORE output identical to the earlier before run"
cd $T
python3 $HERE/sk6_patch_coldlen.py $T
git add components/rules/uniques.hpp components/game/ai.cpp tests/monsters/test_umod_stats.cpp docs/research/re/monsters.md
git "${ID[@]}" commit -q -F - <<MSG
Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)

A Cold Enchanted unique's cold rode the blow with no length, so the chill was one frame; game.exe adds the unit's level x 5 + 100 to the cold length.
Agent: Claude Code
MSG
git log --format='%h %an %s' -3
cd $B && nice -n 19 ninja -j2 test_umod_stats 2>&1 | tail -1
$B/tests/test_umod_stats --umod-stats $O/umod/game.tsv > $O/umod/after3.tsv; cmp $O/umod/after.tsv $O/umod/after3.tsv && echo "AFTER output identical to the earlier after run"
cd $T/tools/emu; uv run python umod_stats.py --compare $O/umod/game.tsv $O/umod/after3.tsv 2>&1 | grep -v WARNING > $O/umod/after3.cmp; python3 $HERE/sk6_umod_sum.py $O/umod/after3.cmp | head -8
cd $T
python3 tools/pr_guard.py upstream/main 2>&1 | tail -2
python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -E "FAIL|WARN|presubmit:"
git push -q -f origin fid/cold-enchanted-length 2>&1 | tail -1; git ls-remote origin fid/cold-enchanted-length | cut -c1-12
