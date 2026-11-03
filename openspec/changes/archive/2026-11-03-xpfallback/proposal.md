## Why

Gate 1 seed 17 stalled 17 virtual minutes with no kill (minutes 299-315: Stony Field, town, Blood Moor, Stony Field hops, then Burial Grounds and Crypt at clvl 11 with healing under the kit minimum). codereview2 R5 to R7 name the XP fallback of the level picker as the likely cause (U: bot-0.log of the run is not available).

## What Changes

- R5: the plan gets a per-step flag `xp_fallback_full_kit` (Burial Grounds 17, Crypt 18, Mausoleum 19, Normal). The XP fallback, and the plan-order pass that follows once the preferred step is excluded, skip flagged steps unless the character has a full kit. The plan order with the guard open is unchanged.
- R6: `stepXP` no longer guesses from the entry level. An area without a monster level returns -1 (ranked last, warned once by area name); when no step has an estimate the plan order stands. `normalMLvl` gains Tamoe Highland 8, Crypt 3, Mausoleum 3 (V levels.txt per codereview2).
- R7: ties go to the later step of the plan, now a tested rule; Nightmare and Hell without install data fall back to plan order.

## Impact

internal/bot (plan.go, recoveryxp.go, runner.go), plandata progression-normal.json. Build-specific data lives in the plan table. Not verified live.
