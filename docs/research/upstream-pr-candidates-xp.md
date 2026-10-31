# Upstream PR candidates from the XP award trace (none ready)

> status: research, 2026-10-30, agent DECOMPXP (card decompxp). Nothing was pushed, posted or filed; no branch was left under `~/d2d-lab/pr/` (a `fix/kill-exp` worktree was created and removed once upstream's fix was found). The general candidate list is docs/research/upstream-pr-candidates.md (UPSTREAMGAPS); this file only covers the XP finding. Tags as in docs/research/xp-award-trace.md.

## The kill-experience bug: already fixed upstream

What our trace found (docs/research/xp-award-trace.md): d2d's `rules::kill_exp` at e36922ea used 81 / 62 / 43 / 24 / 5 % instead of game.exe's 207 / 159 / 110 / 61 / 13 of 256 (`0x6e1668`), used exp x clvl / mlvl instead of the 225 / 174 / 92 / 38 / 5 table (`0x6e1694`) when the monster is 6+ levels above a character under level 25, and had no clvl / mlvl rule from level 25 (`FUN_0057e2f0`); it also skipped the ExpRatio step (`FUN_0057e390`) and the merc share.

Upstream commit 01337ba3 (2026-10-29, "Kill experience as game.exe's FUN_0057e480 (emu-checked)", in main at 32768ddb) ports all of it: `kill_exp` with both tables and the level-25 rule, `kill_award`, `rules::scale` (the `FUN_00483360` shortcuts), the merc share, `tools/emu/kill_exp.py` and 400 oracle asserts in `tests/test_combat.cpp`; `docs/research/re/party.md` holds the reading. V: the code and the test were read in a worktree at 32768ddb; S: not built here. Our exhaustive grid (`tools/emu/xp_award.py grid`, 119,790 cases) is the same function under the same register convention; a run of it against a build >= 01337ba3 would be the only thing to add, and upstream's random 400 already cover the branches. No PR.

## Leads that would need a trace before a PR (not started)

| Lead | What is known | What a PR would need |
|---|---|---|
| Superunique and minion experience multiplier | `005a0e40` (leveladd) is level + 3, exp x 5 (V); whether `005a49b0` (superuniques) and the minion path apply it is U. d2d's Den sweep fitted Corpsefire at 19 XP where 33 x 5 = 165 is expected, but the fit cannot separate the boss from its 5 minions | read `005a49b0` and the minion spawn path for stat 13, run them in the oracle on a stubbed unit, read d2d's `boss_stats`, then a `killed Corpsefire (+N exp)` line from a d2d build >= 01337ba3. If d2d differs: one claim, one test |
| `FUN_0057e3f0`'s merc cap (level span / 64) and `0057e860` | read for control flow (V), constants are upstream's (S); upstream says it ported `merc_kill_exp` | nothing unless a d2d merc levels visibly wrong |

## For our own fork (not upstream)

The lab branch (base e36922e) carries the old `kill_exp`: rebase or cherry-pick 01337ba3 before the next replicated soak that runs past clvl 7 or fights bosses 6+ levels above the character (docs/research/xp-award-trace.md, "Consequence for us").
