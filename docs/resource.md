# Resource policy: internal/resource (agent RESOURCE)

Pure decision functions for mana and health. No game access, no clock, no goroutines: callers fill `resource.Input` from a snapshot
and act on the answer. Tests: `go test -p 1 ./internal/resource` (policy tables plus the replay harness).

## Functions
| Function | Question it answers |
|---|---|
| `PreEngage(in, t) PreDecision` | A pack is in view, nothing fights yet: `Proceed`, `RestHere`, `RetreatRest`, `DrinkMana`, `DrinkLife`, `TownHeal`, `Avoid`. Carries `WaitSec` (rest timeout) and `TargetMana`. |
| `AttackChoice(in, t) AttackDecision` | Fighting: `Cast` (skill), `Free`, `Melee`, `Wait` (stop casting), `Retreat`. |
| `PotionPolicy(in, t) PotionDecision` | Drink life / mana / rejuv now (`Best` = highest tier). One drink per call; the caller keeps the potion gap. |
| `PlanFor(class, clvl) BeltPlan`, `PlanBuys(in, t, prices) Buys` | Belt column layout and tier by level; shopping list inside the gold budget. |
| `TownTripTrigger(in, t) Trip` | Outside a fight: `Go`, `Futile`, method portal or walk, reason. |
| `AbortFight(in, t) AbortDecision` | `Continue`, `RetreatOut`, `PortalOut`, `LeaveGame`. Run after `PotionPolicy`. |
| `Replay(Scenario, Policy, Tuning) Report` | Deterministic regen simulation; `ResourcePolicy()` vs `LegacyPolicy()`. |

### Rules in one place
- **Rest before a pack** (casters, `ManaNeed` 2 = Sorceress/Necromancer/Druid, 1 = others): target = max(`restStartManaPct` of the pool, `castsPerMonster` x pack x cheapest cost), capped at 90%. Rest if it takes at most `restMaxSec`; out of sight when pack danger >= `retreatDanger`. If it would take longer: go in with melee/free fallback, else one mana potion, else rest to the timeout. With hostiles already on top, `PreEngage` answers `Proceed`.
- **Life does not regenerate on its own (U).** Resting heals only with gear regeneration (`Input.LifeRegen`). Below `engageLifePct` the fix is a spare healing potion (above `hpReserve`) when the pack is dangerous or life < `minEngageLifePct`, else free healing in town, else `Avoid`.
- **Low-mana ladder** (Kolbot LowManaSkill, prior-art 04): first ready affordable spell in preference order; below `economyManaPct` the cheapest one; then a free skill; then melee (life >= `meleeMinLifePct`, <= `meleeMaxPack` hostiles, danger < 2); then `Wait` (stop casting, regen) while standing is safe; else `Retreat`.
- **Potions** thresholds equal the combat defaults (drink 55, rejuv 35, chicken 25/30 Hell, mana 25). Out of a fight a life potion is only drunk below `minEngageLifePct`. A mana potion is held while mana still pays 3 casts, and when poor (gold < `poorGold`) with a melee/free fallback and a non-elite pack (life potions come first when poor).
- **Belt**: `PlanFor`: casters hp hp mp mp (rejuv replaces the last mp column from clvl 31, as plan.json does: hp hp mp rv, and hp rv mp rv from 61); Amazon/Paladin/Assassin hp hp hp mp; Barbarian no mana. Preferred tier by clvl 1-7 minor, 8-15 light, 16-27 normal, 28-44 greater, 45+ super, capped by plan.json's band cap (2 up to 12, 4 up to 30, 5 above). Divergence from `internal/town/belt/plan.json`: two mana columns for casters below clvl 31 (live starvation). 
- **Gold**: spendable = gold - `reserveGold` (- 100 without a portal scroll); not poor: at most `potionGoldPct` of it. Order: `hpFirst` healing, `mpFirst` mana (all classes with ManaNeed >= 1), fill healing, fill mana, rejuv. The tier drops until that first batch fits, so a poor character buys minor potions.
- **Trip**: healing <= `tripHPMin`; casters with mana potions <= `tripMPMin` and starved (rest to budget over the timeout) or nearly out of healing too; life < `tripHealPct` with no regeneration (free healer). Potion reasons are `Futile` below `tripMinGold` or when gold grew less than `futileStep` since the last trip ended; the heal reason is never futile. Needs a portal or `NearTown`; never during a fight.
- **Abort**: no potion and life <= chicken: portal if within 8 points of the line and a portal exists (1.3 s vs 1.8 s to leave), else leave. Life under the drink line, no healing, and no way to attack (no mana for the cheapest spell, no mana/rejuv potion, no melee/free) with hostiles: portal, else retreat toward town (`NearTown`), else leave. Merc dead + >= `mercDeadPack` hostiles (or elite pack) + life under `engageLifePct`: portal/retreat. Mana alone never aborts.

## Replay result (`BloodMoorSorc`: clvl 8, 60 life, 45 mana, Fire Bolt cost 3 / 8 dmg, packs of 1-4 every 9 s, 300 s)
| | legacy (cast until empty, wait 700 ms) | resource policy |
|---|---|---|
| seconds fighting with mana below one cast | 97 s of 307 s | 0 |
| melee hits / rest seconds | 0 / 0 | 14-29 / 28-57 |
| kills per minute | 4.5 | 4.1-4.5 |
| deaths | 0 | 0 |
Regen-time sensitivity (50 / 100 / 200 s to refill, a U figure): the policy never starves in any of them and kills 3-7 more than legacy at 50 and 200 s. Honest limit: natural regen bounds a Sorceress at clvl 8; the policy does not raise kills/min, it replaces idle waiting beside monsters with resting before the pack and a staff fallback during it. The live yield levers are mana potions (belt plan), the staff, and Static/cheap skills. All simulation numbers are U (the Combi run: 13.9 kills/min, mana 69 -> 1, 0 potions, docs/combat.md, is only the shape).

## Tags
- V: nothing verified by this package (offline only).
- S: class mana needs (docs/classes), Kolbot ladder and thresholds (prior-art 04, 07), belt tiers (plan.json).
- U: natural mana regeneration (refill in 100 s, same for all classes; real value comes from charstats ManaRegen and gear), no natural life regeneration, potion prices (`DefaultPrices`), healer restores life and mana, potion heal amounts in the sim, portal 1.3 s / leave 1.8 s (from the combat doc), the clvl to tier map.

## Live checklist (for the main agent, not run)
1. Combi or Testy, mana empty, stand still 30 s, log mana every 5 s: gives the real regen per class; set `manaFullSecs` (types.go) or feed `ManaRegenB`.
2. Life at 50% in the field for 30 s: confirms no natural life regeneration.
3. Akara heal at low mana: confirms the healer restores mana.
4. Vendor prices per tier from `Stock`: pass as `Prices`.

## Integration requests (DONE by INTEGRATE, 2026-10-28; the list below is the original request)

Wired: `env.ResourceTuning` (struct in env, the 28 knobs registered from `cmd/d2bot/resource.go` through `env.RegisterKnobs`, `resource.FromEnv` keeps the potion thresholds single-sourced from `env.CombatTuning`); combat `PreEngage` (rest or retreat-rest before a pack, one `Act` per drink, `ErrTownPortal` for a free heal), `AttackChoice` (low-mana ladder incl. a `KindMelee` skill: left click on the target, offered when the left skill is Attack), `PotionPolicy` for the mana potion (`Safety`), `AbortFight` after the survival step; bot `TownTripTrigger` in `choresDue` (`Futile` silences the plain potion rule, `LastTripGold` from the end of the last chore); town belt `PlanForClass` takes its column kinds from `resource.PlanFor` (casters: two mana columns below clvl 31; the tier caps stay plan.json's). Tests: `go test -p 1 ./internal/combat` (rest, melee, portal), `./cmd/d2bot` (28 knobs, edit reaches the policy, replay with the wired tuning).
Live (i4, Booti, clvl 5-6): the rest happens (`combat resting before the pack`), mana regenerates about 7 per 21 s at 43 max (full in about 130 s, not the U 100 s: the rest timeout is 1.5 x estimate + 3 s), the melee fallback fires when a spell cannot be paid (first version used reach 2 and dropped a target 3 subtiles away after 12 approach steps: reach is now 7, the click walks by itself).

## Original integration requests (nobody edited outside internal/resource)
- **env**: register `resource.Knobs()` (28 knobs, keys `resource.restStartManaPct, castsPerMonster, restMaxSec, engageLifePct, minEngageLifePct, retreatDanger, hpReserve, economyManaPct, meleeMinLifePct, meleeMaxPack, lowManaWaitMs, waitMaxSec, poorGold, reserveGold, potionGoldPct, hpFirst, mpFirst, tripHPMin, tripMPMin, tripMinGold, futileStep, tripHealPct, mercDeadPack, fleeLifePct, kitTPGold, kitPotionGold, povertyMargin, povertyLevels`) in `env.Knobs()` plus a way to load `resource.Tuning`. The package imports env, so env cannot import it: add a `ResourceKnobs func() []Knob` hook or append from the registrar in cmd. The potion thresholds (drink 55/35, chicken 25/30, mana 25) duplicate `combat.*`; map `Tuning.DrinkLifePct` etc. from `env.CombatTuning` when building a `resource.Tuning` so they stay single-sourced.
- **internal/combat** (`Fighter.Clear`): before each pack call `PreEngage` (loop on `RestHere/RetreatRest` until `Proceed` or `WaitSec`, re-asking every 1-2 s with fresh mana); replace the `PickAttack` mana fall-through and the 700 ms wait by `AttackChoice` (needs a `Melee` skill entry: the basic attack on the left button); call `PotionPolicy` instead of the mana branch of `Decide`; call `AbortFight` after it, mapping `PortalOut` to `ErrTownPortal`, `LeaveGame` to the chicken path, `RetreatOut` to the kite/retreat step.
- **internal/bot**: call `TownTripTrigger` every idle tick (replaces/extends `town.NeedsTrip`; use `Futile` to silence repeated trips, `LastTripGold` = gold when the last chore ended). Feed `PackDanger` from the danger table, `MercLifePct` when merc support exists.
- **internal/town**: `PlanBuys` can replace the fixed fill target (pass `Stock` prices); `PlanFor` for the column order (note the two-mana-column difference).
- **internal/leveler**: `Poor()` is the shared definition of poverty; the `bot.starveGold` knob was removed (it had no reader), `resource.poorGold` and `resource.povertyMargin` are the live ones.

## Poverty and flee (agent/poverty)

Evidence (V, docs/stallfatal.md section 5, soak log): a level 12 Sorceress with 0 gold, 0 potions, no town portal stood in
melee range 51 s (life 62 -> 22 %) until the chicken; hours at level 12 'resupplying' with 'town trip abandoned'.

- Flee (`resource.AbortFight`, kind `Flee`): no healing/rejuv potion, no portal, life under `resource.fleeLifePct` (50) and
  above the chicken line: `combat.abortFight` walks 3 kite steps away from the hostiles (`awayPoint`), every tick while
  they stay in the fight radius. The chicken stays the last resort at the real chicken line. Replaces the old
  `RetreatOut` for "no portal, near town". Limit (U): the target is away from the hostiles, not toward the level entrance
  or town; monsters faster than the player still catch up. Offline test: combat `TestFleesWithoutPotionOrPortal`.
- Poverty (`resource.PovertyCheck`, `bot.updateStarvation`): on when the kit is incomplete (no portal, or fewer than
  `hpFirst` healing potions) and gold < `KitCost` = `kitTPGold` + `hpFirst * kitPotionGold` (100 + 3*30 = 190; the scroll
  price is S, the potion price V); off at cost + `povertyMargin` (60) or when the kit is complete. One log line per
  change: `poverty mode: on|off` with reason, gold, missing.
  - Farm area (`starveArea`): with a portal, the plan area for `level - povertyLevels` (4), cleared and banned areas
    excluded; without a portal only Blood Moor (area 2), because `town.walkToTown` only walks from there (V, trip.go).
  - Sell first: in poverty one sellable item in the field starts a sell trip (before the old `SellMinItems` rule); the
    count the vendor left behind is remembered so a pack of unsellable junk does not loop.
- `resource.BuyKit` is the scroll-first purchase order. NOT wired: `town.shopVisit` buys potions before scrolls and the
  adapter sets `Scrolls: Potions`, so with 120 gold potions eat the scroll money. Request to the TOWN owner: in
  `shopVisit` (internal/town/chores.go) call `buyScrolls` before `buyPotions`, or size `buyPotions` with `resource.BuyKit`.
- U: whether Blood Moor is still not cleared in the same game (a cleared farm area ends poverty farming and the planner
  falls back to the plan area; a new game resets it).

## Early Sorceress (EARLYSORC, 2026-10-29, offline analysis of the Soaky soak)

Findings (V = read from the soak log `livebot/20261029-085652-d2bot`, U = inferred):
- V: Soaky resumed at clvl 4 and made one `[heal]` trip only (no shop step ran): NeedsTrip skips gold reasons below
  `TripMinGold` (60), so gold was under 60, i.e. no starting kit and nothing to buy. Fresh D2 characters hold no gold (docs/cheats.md).
- V: the quest task "Clear the Den of Evil" is one 15 minute step; the Den has monsters near all the time, and
  `NearTown` was true only in Blood Moor, so TownTripTrigger answered "no portal, town too far" and the free heal never fired.
  `abort.Flee` ran in circles (random break-away points) for 25 minutes at 21-27/73 life.
- V: "out of mana: backing off" loops: a Sorceress' left skill is a spell, so `resSkills` offers no melee fallback; with mana 1
  and more than MeleeMaxPack hostiles, AttackChoice returns Retreat. U: needs a combat-side staff swing (left skill Attack).
- Fixed here: `resource.NearTownArea` (Blood Moor, Den, Cold Plains: the fields `town.walkToTown` can walk home from) feeds
  `Input.NearTown` in the bot; a mana-starved caster without mana potions gets the free heal (the healer refills mana, U);
  the policy's trip carries `Plan.Heal`. Not fixed (other owners): combat's own `nearTownAreas` should use `NearTownArea`;
  the Den quest floor in `task.NormalAct1` (1) lets a kit-less clvl 1-3 character in.

Recommended Sorceress plan, levels 1-8 (S: general D2 knowledge, U for the numbers): no kit exists at the start, so
level 1-3 in Blood Moor with Fire Bolt on the right hand and a staff swing as the mana-out fallback, sell every drop
(also chipped gems) at Akara/Charsi, buy in this order: Town Portal scroll (the way out), 2-3 Minor Healing potions, then
mana potions; heal at Akara whenever life < 50% or mana is starved (free, about 60 s round trip from the Den).
Take the Den only with a scroll or 3 potions, or at clvl >= 5; Cold Plains from clvl 5; put the first skill points
into Fire Bolt / Warmth (mana regen) and stat points into vitality and energy.
