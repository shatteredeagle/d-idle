## Why

User decision 2026-11-04 (SME version, question 6): the merc revive stall in Gate 2 seed 11 (366 'revive skipped (gold)' lines) is fixed knob first.

## What Changes

Raise merc.goldFarmMaxLevel from 16 to 18 and lower the revive reserve (KitReservePct) from 150 to 100 percent of one kit refill, log the real bar (need=) in the skip line; tests with two profiles. Overlap with card m3-merc declared (internal/merc, internal/env/tuning_merc.go, docs/knobs.md edits, not listed in owns).

Overlap (declared): internal/merc/** is owned by m3-merc; this card edits policy.go, logs.go and the merc tests (small edits) and adds revive_log_test.go there without listing it in owns.

## Impact

Offline only, nothing verified live.
