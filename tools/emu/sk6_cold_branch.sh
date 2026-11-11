#!/bin/bash
# decomp6 (worker host): branch fid/cold-enchanted-length, commit 1 (the differential), from upstream main
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
cd $T
git switch -q -c fid/cold-enchanted-length upstream/main || exit 1
python3 $HERE/sk6_cold_tool.py $HERE $T
python3 - <<'PY'
p="tools/emu/README.md"
s=open(p).read()
row="| A unique's, champion's and minion's spawn stats (FUN_005a2120 and the mods' inits: life, level, experience, damage, speed, resists, enchanted damage and cold length, mana burn) | `uv run python umod_stats.py 100000 1 > game.tsv; ../../build/tests/test_umod_stats --umod-stats game.tsv > d2d.tsv; uv run python umod_stats.py --compare game.tsv d2d.tsv` | the fields d2d models: `cases N`, then each field that differs (a minion has no mods in d2d: its rows differ) |\n"
key="| A Blessed Hammer hit"
assert key in s
s=s.replace(key,row+key,1)
open(p,"w").write(s)
PY
git add tools/emu/umod_stats.py tests/monsters/test_umod_stats.cpp tools/emu/README.md
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q -F - <<MSG
tests: the unique's mods, d2d against game.exe's own inits

tools/emu/umod_stats.py runs FUN_005a2120 and the mods' inits on one unit; test_umod_stats is d2d's boss_stats on the same cases.
Agent: Claude Code
MSG
git log --oneline -2
cd $B && cmake . > /dev/null 2>&1; nice -n 19 ninja -j3 test_umod_stats 2>&1 | tail -2
