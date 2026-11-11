#!/bin/bash
T=$HOME/d2d-lab/wt-decomp5; cd $T
sed -i 's#tools/emu/d5_gold_load.py#tools/emu/fl_gold.py#g' tests/game/test_game.cpp
sed -i 's# (TASK-79)##' tools/emu/fl_gold.py
grep -n "d5_\|TASK" tests/game/test_game.cpp tools/emu/fl_gold.py components/game/world/setup.cpp | head
git add -A components tests tools/emu/fl_gold.py
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q --amend --no-edit
git log --oneline -1
python3 tools/pr_guard.py up/main HEAD 2>&1 | tail -3
python3 tools/presubmit.py --quick --base up/main 2>&1 | tail -3
git push origin local/fid-load-gold-clamp 2>&1 | tail -3
