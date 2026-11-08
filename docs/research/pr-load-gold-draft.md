# Draft (local, NOT posted): "game: a saved purse or stash over its cap is 0 on entering a game"

> status: draft 2026-11-08 by agent decomp5, for the independent verifier and then the user. Candidate branch `local/fid-load-gold-clamp` (private fork remote), commit 42d40b09 on upstream main 1453f853. Card TASK-203. Proof and numbers: [../probes/fid/fid-inv-limits.md](../probes/fid/fid-inv-limits.md). Maintainer rules (memory d2-pr-proof-standard): one subsystem per batch, one change per commit with test and oracle, Claims table, numeric noreply identity, last line `Agent: Claude Code`, no Co-Authored-By.

Title: `game: a saved purse or stash over its cap is 0 on entering a game`

Body:

> game.exe's loader FUN_0056b180 puts the saved gold and stash gold back through FUN_0056af80 -> FUN_0053ff00 (0x56afb4..0x56b00f), which sets the stat to 0 when it would pass its cap: clvl x 10000 for the purse (FUN_00622e70), 2,500,000 for the stash (FUN_00623460). `World::take_character` kept both.
> `tools/emu/fl_gold.py` runs the real FUN_0056b180 on a stat block (539 cases: 11 levels x 7 purse values x 7 stash values) and `test_game --load-gold` prints d2d's `World::enter`; 363 of 539 differ before, 0 after.
> Docs-sync warn: no doc lists the load clamp; none changed. `--quick` skips the build: `test_game` rc 0 here.
>
> Agent: Claude Code

## Claims table

| Claim | Evidence | Check |
|---|---|---|
| game.exe zeroes a loaded purse over clvl x 10000 | `fl_gold.py game` (real FUN_0056b180 -> FUN_0056af80), 231 purse cases over the cap give 0 | rerun `fl_gold.py game` on the 539 cases |
| game.exe zeroes a stash over 2,500,000 | same, 231 stash cases | same |
| equal values (at the cap) are kept | cases `g = cap`, `b = 2,500,000` | same |
| d2d did not | `test_game --load-gold` on 1453f853: 363 of 539 differ | build the parent, run `--load-gold` |
| d2d does now | 0 of 539; unit block "a save's gold over its cap is 0" | `test_game` |
| Reachability | U: needs an edited or imported save | not claimed in the PR |
