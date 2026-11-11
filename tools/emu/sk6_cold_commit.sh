#!/bin/bash
# decomp6 (worker host): commit 2 of fid/cold-enchanted-length and the checks
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
cd $T
git checkout -q -- components tests/monsters/test_umod_stats.cpp docs 2>/dev/null
python3 $HERE/sk6_patch_coldlen.py $T
git add components/rules/uniques.hpp components/game/ai.cpp tests/monsters/test_umod_stats.cpp docs/research/re/monsters.md
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q -F - <<MSG
Cold Enchanted chills for its level x 5 + 100 frames (FUN_005a1c70)

A Cold Enchanted unique's cold rode the blow with no length, so the chill was one frame; game.exe adds the unit's level x 5 + 100 to the cold length.
Agent: Claude Code
MSG
git log --oneline -3
python3 tools/pr_guard.py upstream/main 2>&1 | tail -3
python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -E "FAIL|WARN|presubmit:"
