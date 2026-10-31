# Potion and belt policy (d2-sme review, 2026-10-29)

Tags: V = verified from code or logs (how stated), S = secondary source, EXP = veteran judgment (d2-sme), not a fact, U = unverified.
Code: `internal/resource/{potion,beltplan,abort}.go`, `internal/combat/{safety,resource}.go`, `internal/town/{beltbuy,shop,beltfill}.go`. All knobs are in docs/knobs.md.

## What changed
| gap | change | knob (default) |
|---|---|---|
| retreat on thin belt ignored the pack | `thinBeltRetreat` counts belt HP+RV plus the pack's HP/RV potions the belt has room for (`Belt.Spare`: min(pack, free slots)). With spares the bot drinks and fights on; `beltFillStep` (bot/resource.go) moves the spares to the belt as soon as no hostile is within ClearRadius. No refill inside a fight (it opens the inventory). | `resource.retreatCountPack` (1; 0 = belt only). Note `resource.retreatBelt` itself defaults to 0 (retreat off), so the default behaviour of a run does not change. |
| no belt purchase | town chore `beltbuy` after the potion shop: cheapest belt in the repair vendor's stock with more potion slots than the worn one, gold must stay above MinGoldReserve + a minimal kit (TP scroll + `hpFirst` healing potions). A failed attempt is not repeated until gold grew by `futileGoldStep`. Rebuy after a death also buys the belt slot below `earlyGearMax` once `beltBuyLevel` is reached. | `town.beltBuyLevel` (10; 0 = never) |
| mana potion held back by the cast budget | the "mana still pays for N casts" rule (hard-coded 3) is a knob | `resource.manaPaysCasts` (3 = old; 0 = drink at the line) |

## Belt size (V code, belts.txt via `belt.Slots`)
No belt: 4 slots, one row. Sash and Light Belt: 8 (2 rows). Belt: 12 (3 rows); Heavy and Plated Belt in this install 12 or 16 per belts.txt (U, not read here). The column plan is per column times rows (`BeltPlan.capacity`, `belt.Band.Capacity` use the real slot count), so a bigger belt automatically plans more potions per column. EXP: a belt beats any armor at clvl 12 (play-economy.md), so `beltBuyLevel` is 10.
Untested live: which vendor carries which belt (the repair NPC of the act is searched; an absent belt just logs "nothing affordable and viable"), and that equipping over a worn belt swaps (equipUnit swaps for other slots, V in the sim only).

## Drink thresholds: evidence, no default change
Ours: healing below 55 percent, rejuvenation below 35, mana below 25 (35 for mana-heavy builds), chicken 25 (Hell 30). EXP veteran: healing 60-65, rejuvenation 35-40, mana 30. Kolbot defaults 75/40/30 (S).
Soak logs `~/d2lod-live/soak24/runs/*.log` (12 runs, 2026-10-29, Normal Act 1, clvl 9-12; V by script over `combat fight` and `combat potion` lines):
- 2265 fights, 75 healing, 2 rejuvenation and 51 mana drinks, 15 deaths.
- Fights in which a potion was drunk, by the lowest life percent reached (`minlife`): 30-39: 6, 40-49: 24, 50-59: 29, 80+: 47 (the 80+ are mana drinks). So a healing drink at line 55 bottoms out 6 to 25 points below the line: the reaction lags, as docs/soak-analysis.md already measured (V).
- 306 fights ended with minlife under 50 and no potion at all (belt empty or poor character, V: `potions=0`).
- Deaths: in the 8 deaths read in detail none follows a combat drink line; the last log lines are navigation or `status life: 7..11` (exploring a level with a Vile Hunter / Dark Ranger pack, or `town portal attempt failed`). The death cause is no potion or burst damage outside the fight loop, not a late drink threshold.
Conclusion: the data does not show that 55 is too low; it shows the effective drink point is about 45, and the problem deaths are out of potions or outside the drink loop. Proposal (EXP, not applied): `combat.drinkLifePct` 60 and `combat.drinkManaPct` 30 once a soak with the tracked `minlife` per fight is available for A/B. The knobs exist; change them there, not in code.

## Casters and mana (V code)
Order every tick: `Decide` (life, chicken, rejuvenation, healing; mana is switched off there) then `PotionPolicy` for the mana potion, then `AttackChoice`. The cheapest-skill swap (`economy`) starts below `resource.economyManaPct` (40) which is above the mana drink line (25), so a caster swaps skill first and drinks later; and the potion is skipped while mana pays 3 casts. The EXP veteran wants the drink at about 30 percent before the swap: set `combat.drinkManaPct` 30 and `resource.manaPaysCasts` 0 (not default, no live evidence).

## Column plan (V code `resource.PlanFor`, EXP target)
Ours: casters (Sorceress, Necromancer, Druid) hp hp mp mp to clvl 30, hp hp mp rv from 31 (rv from 61 hp rv mp rv); Paladin, Amazon, Assassin hp hp hp mp; Barbarian all hp. EXP target: casters hp mp mp rv from clvl 15, melee hp hp hp rv. Not applied: the plan is a pure function without a knob and rejuvenation is not in vendor stock early (the shop falls back to healing, V shop.go). Before clvl 15 the plan fills healing first (`hpFirst` 3 before any mana potion, V PlanBuys).
Tier rule (V): `pickPotion` buys the highest tier within the plan cap that fits the gold, so Light is bought when Light is for sale and affordable; Minor only when nothing better is for sale or affordable.

## TP, boss consumables
- Tome (V): bought at gold >= `town.tpTomeMinGold` (450, the tome's price S) with none owned; then the target is 5 scrolls. EXP "by clvl 6 or 450 gold" is the same condition because the tome costs 450. Left as is.
- Antidote for Andariel, Thawing for Duriel (not done): there is no boss readiness list to extend. `resource.AntidotesToCarry` exists only for Hell and is not wired to shopping (V grep). A follow-up would add the two potions to the Act 1/2 boss gate.
