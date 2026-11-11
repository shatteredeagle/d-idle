#!/bin/bash
# decomp5 (worker host): commit the gold-clamp candidate on local/fid-load-gold-clamp and run the upstream guards
set -u
T=$HOME/d2d-lab/wt-decomp5; cd $T
git branch -f up/main 1453f853
git add components/game/world/setup.cpp tests/game/test_game.cpp tools/emu/fl_gold.py
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q -F - <<'MSG'
game: a saved purse or stash over its cap is 0 on entering a game

game.exe's loader FUN_0056af80 (0x56afb4..0x56b00f) puts gold and stash gold back
through FUN_0053ff00, which sets a stat to 0 when it would pass its cap: clvl x 10000
(FUN_00622e70), 2,500,000 (FUN_00623460). Oracle tools/emu/fl_gold.py runs the real
loader: 539 cases, 0 mismatches after, 363 before.

Agent: Claude Code
MSG
git log --oneline -2; git show --stat --format=%h HEAD | tail -5
python3 tools/pr_guard.py up/main HEAD 2>&1 | tail -5
python3 tools/presubmit.py --quick --base up/main 2>&1 | tail -8
