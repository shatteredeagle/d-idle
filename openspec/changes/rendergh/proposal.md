## Why

PR #136 (card hitlen) made d2d hold a get-hit for game.exe's length. CodeRabbit then noticed that d2d's renderer still draws monsters and the merc at the base animation rate and loops the frames, so the longer hold replays the animation (a Zombie: 680 ms hold over a 335 ms animation). game.exe's client steps the get-hit animation once at the get-hit rate and stands on the last stepped frame. That is a second claim, separate from #136, and needs its own PR candidate under the user's PR standard (docs/process/upstream-pr-standard.md).

## What Changes

- The client's frame rule is traced in game.exe 1.14d (the monster's client tick, the mode table row, the over test and the step, the draw's frame getter) and tagged V / S / U; the player's path is traced too and left out of the claim.
- `tools/emu/hit_render.py` (oracle: game.exe's own code under unicorn) prints the drawn frame at each tick of the hold for >= 100,000 cases; `tools/emu/hit_render_harness.cpp` quotes d2d's old lines (the before); upstream's `test_combat --get-hit-frames` is the after.
- A minimal patch in a new worktree of the private clone (`fix/get-hit-render`, stacked on #136's head eb89646e): `rules::get_hit_frame`, the renderer's `get_hit_elapsed`, the merc's FHR in the view; upstream tests, pr_guard; nothing pushed, posted or filed.
- Three docs: proof, draft PR body, evidence pack with PROOF complete / incomplete and the push commands (not run).

## Out of scope

The player's own drawn frames (d2d's prate stretch: measured, not fixed); a joined game's merc FHR on the wire; what the client shows after its own end (the player's last hold tick).
