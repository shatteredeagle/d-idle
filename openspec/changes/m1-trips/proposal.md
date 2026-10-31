## Why

docs/basics-audit.md found three trip triggers that waste gold or time: the `else if` at the poverty branch of choresDue reopens a sell trip at 4 or 5 items with a scroll, bypassing the 6-item fix (row 4, G2); gold earned between visits is carried above the keep line with no trip reason, so a death loses it (E6, G8); and the pack-full thresholds are two percentages (80 and 85) instead of the SME's 4 free cells (row 14, G9).

## What Changes

- internal/bot/goal.go: the second sell-trip door applies povertySellWorth; a `gold` trip reason when carried gold exceeds the keep line by a margin; one pack-full knob in free cells (default 4) replacing InvFullPct and TripPackUsedPct.
- internal/bot/trips_test.go: TestChoresDueSecondDoor, TestGoldTripAboveKeep, TestPackFullFreeCells.
- Delta: town-basics ADDED `Sell trips only when worth it`, `Gold trip above the keep line`, `Pack-full threshold`.

## Dependencies

After cards xpgap and basicsfix are archived (both own internal/bot/goal.go). xpgap's `resource.povertySellMin` covers both poverty branches (docs/d2d-xp-gap.md): if G2 is closed by it, the card keeps only the test and the spec line for it.
