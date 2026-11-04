## Why

Live RT run (docs/retreatlive.md): RT.drink sets life to 20 percent but chickenPct is 25, so the bot correctly chickens and the drill fails on its own setup; RT.portal passes but its 50 tick bound is loose (1 tick measured).

## What Changes

RT.drink setup life is derived from the bot knobs (above chickenPct, below drinkLife), not a literal; RT.portal bound tightened to the proposed 10 ticks; tests; live rerun recipe updated.

## Impact

Drill code only, offline tested; the live rerun comes after merge. Not verified live.

Overlap (declared): m1-liveproof owns internal/drill/**; this card only edits retreat.go and retreat_test.go there.
