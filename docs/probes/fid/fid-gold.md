# fid-gold: gold drop and gold pickup, game.exe vs d2d vs the bot

> status: finished 2026-11-11 by agent decomp10 (Sonnet), offline. Result: **the drop side was already V run (treasure-trace.md section 4); the pickup side is new: the purse is capped at 10,000 gold per character level and the stash at 2,500,000, and a pile that does not fit is taken only up to the cap (V read).** Nothing run in this card; PR: none. Consumer: the poverty and looting knobs (`internal/env/tuning_loot.go`), no new card.

Tags: V run (upstream or earlier traces, cited), V read (this run), D, I, U.

## Questions

**Q1. Pile size. (S, V run in treasure-trace.md section 4 and 8)** coins = `ilvl + rand(5 ilvl)`, at least 1, ilvl = the monster level (the area level for Hell etc.), so mean 3.5 ilvl - 0.5; a champion class entry multiplies by 1280/256 = 5; gold find `coins x (100 + gf) / 100`. **There is no difficulty term**: Hell pays more only because its monsters have higher levels (Blood Moor Hell 31 per kill vs 0.39 Normal). Piles per kill: 0.131 for a plain class, 0.1125 casters and archers, 0.6 per champion, more for uniques (treasure-trace.md). Per kill by ilvl (plain Normal Act 1/2): 1: 0.39, 2: 0.85, 3: 1.31, 4: 1.77, 5: 2.23, 6: 2.69, 10: 4.53, 15: 6.83, 20: 9.12, 30: 13.7 (D arithmetic from the table there).

**Q2. What a pickup adds. (V read FUN_0055c850, called from FUN_00563560 step 2 for a gold item)** `cap = FUN_00622e70(picker) = stat 12 (level) x 10000` (0x622e80: `imul eax, eax, 0x2710`); `purse = stat 14`; `pile = the pile's stat 14`. If `purse + pile > cap` the take is `cap - purse` and the rest (`pile - take`) is left (0x55c882..0x55c899: `edi = cap - purse`, leftover = `pile - edi`, passed on to FUN_00540900 for the pile that stays: I). Else the whole pile. The take is added to stat 14 with FUN_006272b0 (0x55c92a). So **a character with the purse at its cap picks up nothing and the pile stays on the ground** (I for the remainder staying; the message path was not followed). Cap examples: clvl 1: 10,000; clvl 12: 120,000; clvl 25: 250,000.

**Q3. Stash. (V read)** `FUN_00623460` returns the constant 0x2625a0 = 2,500,000 (the stash gold cap, matches fid-loadhp-trace.md "gold and bank clamped to their caps"). Death loses gold by the rule in d2d-fidelity.md check 8 (120 cases match: 120/120), not re-read.

**Q4. d2d. (S)** `Loot::take` and `rules::pick_up` handle gold apart (rules.hpp comment "gold apart"); whether d2d applies the clvl x 10000 cap on pickup was not read (U): docs/d2d-fidelity.md 3c noted autoloot absence only.

## Deviations d2d vs game.exe

| id | Rule | Tag | Differential |
|---|---|---|---|
| D1 | purse cap clvl x 10000 on pickup, leftover stays | game V read; d2d U | buildable: FUN_0055c850 with a fake picker (stat 12, 14) and pile vs d2d `Loot::take` on gold; or live: `debug stat` gold above the cap then walk over a pile |

## Bot findings (not changed)

- B1 `internal/env/tuning_loot.go:11` `GoldMinStack` 10 and `GoldPileShare` 50: by Q1 the knob is right in shape (treasure-trace.md B3 already notes that at ilvl 1 no pile reaches 10); nothing new.
- B2 the bot never checks the purse cap: at clvl 20 the cap is 200,000, far beyond the bot's economy (`LowGoldBelow` 2000, `ValuableMinGold` 1500), so the cap is unreachable in practice (D1 can stay unbuilt). The stash deposit (2,500,000) is likewise far away. Documented so nobody adds a cap rule.
- B3 `internal/resource/sim.go:123,140` `GoldPerKill: 6`: wrong by the treasure card (B1 there); unchanged by this card.

## Drill recipe (not run)

R1 (D1): `debug stat` is not available for gold; `cheat gold` through the d2cheat ledger on a clvl 1 throwaway to 9,990 gold, then pick up a pile of 3 or more; read `dump stats` gold: expect 10,000 and the pile still on the ground with the rest.

## Matrix proposal

IT (gold): "drop size V run (treasure-trace), pickup cap V read (clvl x 10000, stash 2.5M), d2d cap U".
