# NAVEXIT21: "no exit to that level: level 1 from 2"

Source: Gate 1 ms1 seed 17 (193 lines; seed 19: 12; seeds 16, 18, 20: 0), V from the full bot log as read by the gate record.

## Root cause (code path V by reading, trigger U)
- The line without a suffix comes from `nav.ExitTo` (internal/nav/exit.go): no gate known, explore returns `ErrExplored`, map not growing. The Blood Moor is fully fetched, so a gate can only be missing if `LevelMap.Adjacent` has no level 1 entry.
- `d2dworld.borders` REPLACED `Adjacent` on every refetch. A round where `debug nearby` no longer lists the town, or one `debug blocked` probe errors (counted as closed), returns an empty list. V: `TestBordersSurviveAnEmptyProbeRound` fails without the fix (0 border cells) and passes with it.
- Why a round came back empty in seed 17 is U (no recording of the layout, nearby and blocked replies exists in the repo). The same class was fixed for the Tracker backend in NAVFIX (docs/nav.md).
- Fix: `keepBorders` carries over contacts of a neighbour that the new round found nothing for.

## Shopping order (V by reading, test needs extracted tables)
- Order in `shopVisit`: kit potions, first scroll (`buyFirstTP`), potions to floor, scrolls. `buyFirstTP` returned at once when a tome was in the pack, even an empty one, and after the kit potions 284 gold no longer covered scroll plus two potions.
- Fix: an empty tome is no source; for that case only, a first pass before the kit potions buys one scroll when gold >= scroll + TPScrollPotions potions. Characters without a tome keep potions first (TestNoTPRebuyWhenPoor, TestScenario_PovertyLoop still pass). V: `TestEmptyTomeGetsAScrollBeforeThePotionsEatTheGold`.

## Live checklist (not done)
1. Re-run Gate 1 seed 17 (use the worker host, lab binary as in the ms1 record).
2. Pass: no `no exit to that level: level 1 from 2` run, no funded belt window over 5 virtual minutes, no no-kill over 10.
3. If the lines remain: record `debug layout`, `debug nearby`, `debug blocked` replies at the Blood Moor west edge when it happens.
