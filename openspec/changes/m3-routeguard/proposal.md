## Why

The route guard (merc-policy spec) blocks the Monastery road and the tower below clvl 20 without a merc, and the Andariel floor is 15; the d2-sme (2026-10-31) says 20 is too high (537k XP, Act 1 is under-level from clvl 17 to 18, docs/sme/audit-bot-vs-human.md A1) and proposes Monastery 18 without a merc (14 with), tower 16 (12 with), Andariel 15 with a living merc. The numbers are the user's call (grill 0003); this card sets the knob defaults to the decided values and writes the MODIFIED deltas with them.

## What Changes

- internal/env/tuning_route.go, tuning_boss.go: defaults for route.monasteryMinLevel, route.towerMinLevel, boss.andarielMinLevel set to the decided numbers; a `withMerc` level per guard if the user takes the two-number rule.
- docs/knobs.md regenerated; internal/bot and internal/task floor tests updated (TestPickNeverEmptyInAct1 and the route guard tests).
- Delta (written at launch, after the decision): merc-policy MODIFIED `Route guard` and `Andariel floor` as complete requirements with the new numbers.

## Dependencies

After grill 0003 (docs/roadmap.md decision 4) and after xpgap is archived (it owns docs/knobs.md). skip_specs is true until the deltas are written at launch; the card must not launch without them.
