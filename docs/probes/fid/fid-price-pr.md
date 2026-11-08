# fid-price-pr (TASK-76, matrix rows VN01 VN02 VN03 IT08 VN06): vendor prices, d2d against game.exe, re-checked on current main

> status: finished 2026-11-08 by agent decomp5 (Sonnet). Result: **ALREADY UPSTREAM, no PR candidate.** The first run of 2026-11-07 (agent fidprice, [fid-vendor-price-trace.md](../../research/fid-vendor-price-trace.md)) is reproduced on the newer upstream main 1453f853. Nothing posted, nothing pushed (no new branch).

Tags: V run (game.exe's own FUN_0062efb0 run by the emulator and compared with d2d's `item_price` / `repair_cost` through `test_game --price`, byte for byte), V read, I inferred, U unverified.

## 1. Claim and verdict

Claim of the card: buy 39,626, sell 38,939 and repair 42,705 of 100,000 oracle cases differ; fix them by an upstream PR (price-trace.md Q5 D1 to D10, pr-vendor-repair-gate-draft.md).

Verdict: **the claim is out of date; every deviation D1 to D10 is merged upstream** (list with commits: fid-vendor-price-trace.md Q1). The oracle on upstream main **1453f853** (2026-11-08, newer than the earlier run's 9c78f391):

| Run (`tools/emu/d5_price_run.sh`, which calls `fp_price_runs.sh`) | Cases | buy / sell / repair differ | Cases with a difference |
|---|---|---|---|
| gen seed 7 (the matrix row's number) | 100,000 | 1 / 0 / 13 | 14 |
| gen seed 1 | 100,000 | 0 / 0 / 16 | 16 |
| edge seed 3 (bases around 0xffff / 0x10000) | 20,000 | 2 / 1 / 31 | 34 |
| tomes (4 vendors x 3 difficulties x 6 quantities x 2 tomes) | 144 | 0 / 0 / 0 | 0 |

Seed 7 equals the earlier run to the case (1 / 0 / 13). The residual is the earlier trace's R1 to R4 (32-bit overflow of buy x missing durability above 2^31 at prices of 26 million gold and more, produced by random magic-stat magnitudes; V run, unreachable with 1.14d data, I): not a PR claim, unchanged. Tag: V run for the counts; "unreachable" stays I.

## 2. Acceptance criteria

- #1 (39,626 / 38,939 / 42,705 before, 0 after): the "before" numbers belong to a tree before #152 and the later merges; today's tree has the residual above (14 cases of 100,000, all with the overflow family). Not ticked as written: the 0 is not reached and the remainder is not reachable (I).
- #2 (consumable prices, tomes): ticked earlier; 2,304 + 144 agreeing cases, reproduced today for the 144 tome cases.
- #3 (PR draft): none, nothing to send (already upstream). Not ticked.
- #4 (VN06, bot early gold plan with sell 50 percent): bot side, not touched here; the rule the bot's `SellValue` implements is the oracle's (fid-vendor-price-trace.md, B1). Left open.

## 3. Deviations and bot findings

D1 (carried from fid-vendor-price-trace.md) repair overflow above 2^31: V run, I (unreachable). Differential: yes, `fp_price_runs.sh gen|edge`. No new deviation found.
Bot findings: unchanged (fid-vendor-price-trace.md B1). Sale over the purse cap (earlier D4, "game.exe's sell handler not traced"): d2d's `store_sell` already follows FUN_00579510 (the excess is returned to drop at the player's feet, FUN_0055b060; rules.hpp comment), so the "excess is lost" wording of price-trace.md Q4 is out of date; the sell handler itself was not run natively here (U). See also fid-inv-limits.md (the cap functions are V run).

## 4. Drill recipe

Live quote check (TASK-56): open Akara at clvl 3, read the TP tome price with the stock quantity (expect 250 + 50 per scroll held); sell a rare identified item and compare the gold change with `item_price(..., sell)`.
