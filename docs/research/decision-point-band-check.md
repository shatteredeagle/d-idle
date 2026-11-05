# Decision-point band edges checked against the repo tables

> status: read-only check, 2026-11-05. Compares every band edge of [decision-point-sme-review.md](decision-point-sme-review.md) (unchanged) and the urgency inputs of [decision-point-store-spec.md](decision-point-store-spec.md) section 2.3 with the repo's own data. Docs only; no code, no live game. The SME review is not edited.

Tags: **V** = read in this worktree (file:line given) or a value the repo marks verified live in the cited doc; **S** = repo doc quoting a source; **U** = unverified or my inference. "SME value" is the d2-sme answer from memory (SME (U)). Verdicts: MATCH (same value or same edge), DIFFERS (repo has data and it disagrees or has another shape), NO REPO DATA (nothing to compare with). Classic 1.14d, Act 1 Normal.

## Table

| # | band edge | SME value | repo value (file:line) | verdict | proposed value |
|---|---|---|---|---|---|
| 1 | life: upper edge | 70 | `combat.kiteLifePct` 70 (V internal/env/tuning_combat.go:90), `town.healBelowLifePct` 70 (V internal/env/tuning_town.go:64) | MATCH | 70 |
| 2 | life: 40 | 40 | `resource.minEngageLifePct` 40 (V internal/env/tuning_resource.go:81), but it is an engage floor, not a drink edge | MATCH (other role) | dropped from the life key by the 2026-11-05 decision (spec 2.1); stays a knob |
| 3 | life: 25 | 25 | `combat.chickenPct` 25 (V internal/env/tuning_combat.go:85) | MATCH | 25 |
| 4 | life: 55 and 35 | none | `combat.drinkLifePct` 55, `combat.drinkRejuvPct` 35 (V internal/env/tuning_combat.go:85) | DIFFERS (repo has two edges the SME lacks) | five bands 25/35/55/70, adopted by the user 2026-11-05 |
| 5 | gold: 100 | 100 (one TP) | `resource.kitTPGold` 100 (V internal/env/tuning_resource.go:86); town portal scroll sells for 100 (V docs/d2d-fidelity.md:180, live) | MATCH | 100 |
| 6 | gold: 500 (tome plus potions) | 500 | `town.tpTomeMinGold` 450 (V internal/env/tuning_town.go:66); a Tome of Town Portal costs 450 live, base 250 (V docs/d2d-fidelity.md, tome row); `resource.recoveryKitGold` 500 (V internal/env/tuning_resource.go:90) | MATCH | 500 |
| 7 | gold: 1500 | 1500 | no gold-on-hand knob at 1500. `loot.valuableMinGold` 1500 is an item value (V internal/env/tuning_loot.go:29), `loot.lowGoldBelow` 2000 (same line) | NO REPO DATA | keep 1500 as the gamble gate (spec t1.gamble) until a play-economy doc sets it |
| 8 | gold: repo edges the SME lacks | none | 300: `resource.tripGoldFloor` (V tuning_resource.go:89), `loot.poorGoldBelow` (V tuning_loot.go:30); 60: `resource.tripMinGold` (V tuning_resource.go:84) | DIFFERS (shape) | add an edge at 300 (poor) to the gold band: below 100 / 100 to 300 / 300 to 500 / 500 to 1500 / above 1500; 60 stays a knob |
| 9 | kit OK: healing potions | at least 2 | route kit needs 4 healing (`routeHealing` 4, V internal/task/route.go:169); `resource.kitMinHP` 4 from `resource.kitGateLevel` 8 (V tuning_resource.go:90). 2 is only the trip trigger `town.tripMinHPPotions` 2 (V tuning_town.go:65) | DIFFERS | OK = 4 healing from clvl 8, 2 below clvl 8 (kitGateLevel splits it); THIN = fewer; NONE = 0 and no portal |
| 10 | kit OK: portal | 1 scroll, or tome with 3 or more | route kit needs a portal with 2 charges (`routePortals` 2, V internal/task/route.go:170); `town.tpScrollTarget` 2 (V tuning_town.go:66). Tome holds 20 (`tomeCap`, S internal/town/chores.go:674) | DIFFERS | OK = 2 scrolls or a tome with 2 charges; THIN = 1 |
| 11 | kit THIN: gold with no portal | under 150 | scroll 100 plus `town.tpScrollPotions` 2 potions at 30 each = 160 (V tuning_town.go:70 default 2; potion price V docs/d2d-fidelity.md:184) | DIFFERS (small) | 160, derived from the two knobs, not a literal |
| 12 | kit NONE: gold with no scroll | under 100 | `resource.kitTPGold` 100 (see row 5) | MATCH | 100 |
| 13 | level gap: bands 0 / +1..+3 / +4..+6 / +7 | gap = area monster level minus hero level | no gap rule in the repo; floors are absolute clvl per area (`route.towerMinLevel` 15, `route.monasteryMinLevel` 14, V internal/env/tuning_route.go:31). Traced Normal mean monster level: Blood Moor 1, Cold Plains 2, Stony Field 3, Dark Wood 5, Black Marsh 6, Den 1, Cave 1 2, Cave 2 2, Burial Grounds 2, Outer Cloister 8, Catacombs 2 11 (V internal/bot/recoveryxp.go:68). Plan enter levels are 1 to 14 (V internal/plandata/progression-normal.json) | DIFFERS (form), edges +4 and +7 NO REPO DATA | the gap is 0 or less at every plan entry (enter clvl minus 1 to minus 6 against the traced levels), so the positive bands fire only when the hero is below the plan floor; keep the bands, expect them rarely populated; tower and monastery levels are not in the trace |
| 14 | free pack rows (0 / 1 / 2 or more), trigger at 1 or fewer | rows of 4 | `town.tripPackUsedPct` 80 (V internal/env/tuning_town.go:65) and `loot.freeCellsKeep` 2 (V tuning_loot.go:29), both in cells or percent. Pack size comes from the layout table (V internal/data/layout.go:51) | DIFFERS (small) | 80 percent used equals 2 of 10 columns free of 4 rows, close to the SME trigger; define the band from cells (spec 5.3 already says so): rows free = free cells / 10 rounded down |
| 15 | stash band 50 / 85 / full | percent | stash percent is measured (V internal/town/chores.go:361) but no knob has an edge | NO REPO DATA | keep 50 / 85 / full |
| 16 | death streak 1 / 2 / 3 or more | 3 stops | `resource.gearDeathsHigh` 4 (V tuning_resource.go:88) adds gear risk, it does not stop a run; Blood Raven budget `resource.brAttempts` 3 (docs/knobs.md) | DIFFERS | keep 3 as the question edge; 4 stays the gear-risk edge |
| 17 | run time vs expected 0.7 / 1.3 | ratios | no run-time-versus-expected measure; `route.freshMinXP` 200 and `route.freshMaxNoGain` 3 are XP and counts (V tuning_route.go:33) | NO REPO DATA | keep 0.7 / 1.3 until a measure exists (spec 5.3 gap 5) |
| 18 | belt state: full / half / under 2 | full, half, under 2 | `town.beltFillPct` 100, `town.tripBeltColumnPct` 50, `town.tripMinHPPotions` 2 (V internal/env/tuning_town.go:65-66) | MATCH (all three) | as the SME |
| 19 | gear band: composite, 3 values | 3 | graded risk: medium at 3 points, high at 6 (`resource.gearRiskMedium` 3, `gearRiskHigh` 6, V internal/env/tuning_resource.go:88; `gearGateMode` 1, V docs/knobs.md:345) | MATCH (3 values: low / medium / high) | low below 3, medium 3 to 5, high 6 or more. Corrects spec 5.3, which says "no gear composite exists" for t2.bossunsafe |
| 20 | area classes | town / field / boss | town flag per area (V internal/plandata/areas.json); plan has a `farm` flag; no boss class in the area table | NO REPO DATA for field and boss | town from areas.json; boss class from the quest/boss tables (not read here) |
| 21 | hero life at clvl 5 (SME urgency answer) | about 60 | Sorceress starts with 40 life (V docs/livedrills3.md:3); the Den knob calls a caster "about 40 life" at clvl 4 to 5 (V docs/knobs.md:288). 40 plus about 2 per level is 48 at clvl 5 (U, no vitality) | DIFFERS | use 40 to 50 for a clvl 5 caster in examples; read the real value live before freezing |
| 22 | restore of the best early healing potion (SME urgency answer) | Minor Healing about 45, Light about 75 | Minor 30, Light 60, Healing 100, Greater 180, Super 320 for Sorceress, Necromancer, Druid (V docs/sme/mechanics-combat.md:223-229); 45 is the Amazon, Paladin, Assassin base. d2d drank 30.156 per Minor on a Sorceress (V docs/d2d-fidelity.md:77) | DIFFERS | class-adjusted table from item data (spec 2.3 uses it) |
| 23 | potion price | Minor about 30 | 30 live (V docs/d2d-fidelity.md:184) | MATCH | 30 |
| 24 | urgency edges: htl 2 / 4 / 6, D5 at 0.5 L, hit window 1.5 s, heal factor 0.6 | numbers | no damage window, largest hit or hit timer exists to compare (not found in this pass; combat code not read in depth) | NO REPO DATA | keep as tunable data (spec 2.3); calibrate by replaying soak deaths and a heal-tick drill |

## Counts

24 rows. MATCH 1, 2, 3, 5, 6, 12, 18, 19, 23 = 9; DIFFERS 4, 8, 9, 10, 11, 13, 14, 16, 21, 22 = 10; NO REPO DATA 7, 15, 17, 20, 24 = 5. Row 2 counts as MATCH (same edge, other role).

## Findings that change the spec or the review

1. **Healing potions restore less than the SME assumed for casters** (rows 21, 22): Minor 30 and Light 60 for the Sorceress, not 45 and 75. At a clvl 5 hero of about 48 life one Minor refills 60 percent of life, which makes "potion room" fire later than the SME's figures. Spec 2.3 uses the repo's 30.
2. **Kit OK needs more than the SME's 2 healing and 1 portal** (rows 9, 10): the repo's own kit rule is 4 healing and a portal with 2 charges. The kit field should read the repo's rule, with the clvl 8 split.
3. **The gear band exists** (row 19): graded risk with edges 3 and 6 is already a 3-value composite; spec 5.3's "no gear composite exists" is wrong and should be corrected by the next spec pass.
4. **Level-gap bands rarely fire** (row 13): in every plan window the area's mean monster level is below the hero, so only +1 and above matters when a floor is skipped. The repo's guards are absolute clvl floors per area.
5. **Gold band needs a 300 edge** (row 8): poor and trip-floor knobs sit at 300, between the SME's 100 and 500.
6. **No data**: stash edges, run-time ratio, boss area class and every damage-window input of the urgency table.

Sources not read in depth (U): the death rule spec and soak gates for the streak edge; the combat package for damage tracking; monster levels of the tower and monastery areas (only the traced rows above). Raise row 13 and 24 again when the T0 card reads them.
