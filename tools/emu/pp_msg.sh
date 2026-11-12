#!/bin/bash
cd $HOME/d2d-lab/wt-pp-light || exit 1
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q --amend -m "Lightning hits one unit a frame and skips the last (FUN_005ae1f0)

A LastCollide row with neither NextHit nor Pierce struck every touching monster once; game.exe hands one unit a frame to FUN_005adf10 and skips only the last unit hit. Pierce rows keep their struck list.
Agent: Claude Code"
export PATH=$HOME/.local/bin:$PATH
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main 2>&1 | tail -25
