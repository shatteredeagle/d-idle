# Admission function (P4, card admit1, TASK-970)

> status: step 1 built (package, data, tests); step 2 wired in shadow for the four entry rows and the quest-walk kit (knob `bot.admission`, default 0); control and the other providers not yet.

One pure function decides whether a hero may enter an area or leave town. Package `internal/admit`; floors are rows in
`internal/plandata/admission.json` (mirrored in `docs/plan/data/`). Source: review `research/fable-logic-review-2026-11-25.md`
sections 1.2, 2.4, 7.1 P4.

## API
- `MayEnter(view, area, purpose) (Verdict, error)` and `MayLeaveTown(view, purpose)`; `Rows.MayEnter/MayLeaveTown` take rows (tests, shadow replay).
- `Verdict{Refused, Need, Wake, Why, Row, KitOnly}`. `Need` and `Wake` are the codes and words of `needTable` in `internal/bot/outcome.go`
  (`needs_potions`, `needs_gold_farm`, `too_weak`, `route_closed`, `not_ready`); the bot converts a refusal to a `StepOutcome` with
  `needTable`, the type is not forked, and `TestAdmitNeedsMatchNeedTable` keeps the two lists equal. `KitOnly` says a chores trip fixes it.
- `View` is plain data filled by the caller from the snapshot and the knobs (archetype, level, life, healing potions, portal, gold,
  area, first field, try anyway, plus small callbacks for town, monster level and the not-yet-row judges).
- Purposes: quest-walk, farm, level, boss, corpse-run, trip. Exempt from every row: corpse run, trip home, the owner's try anyway,
  any difficulty but Normal.

## Rows (V: numbers moved unchanged from the old files)
| row | was | rule |
|---|---|---|
| `roads` underground_passage (areas 10, 5) | `earlyroutes.json` routes | level by archetype (melee 7, caster 8), max life by archetype (melee 180, caster 100), kit 2 healing + portal |
| `knobFloors` den_of_evil | `denFloor` | level from the existing knob named per archetype (`campaign.denMinLevelMelee/Caster`), value supplied in `View.Knobs`, not copied |
| `questWalk` | `earlyroutes.json` | clvl 5 to 14: 2 healing and a portal to leave town for a quest; short purse is `needs_gold_farm` |
| `poverty` | `povertyfarm.json` | no potions from clvl 8: monster level at most clvl minus 3 |
| `kitFloor` | `povertyfarm.json` | from clvl 8 under 4 potions: first field only |
| `lean` | `povertyfarm.json` | no portal and under 4 potions: no other field under 60 percent life; same line at the town exit |

The old judges (`earlyRouteCheck`, `questWalkCheck`, `povertyAreaWhy`, `kitFloorWhy`, `leanEntryWhy`) keep their bodies but read the numbers from
`admission.json` through `admit.Load` (`internal/bot/earlyroute.go`, `povertyfarm.go`): one home per number. `earlyroutes.json` is now numberless;
`povertyfarm.json` keeps only the measured farm list.

## Providers not yet rows (step 2 and later)
Monastery road and Barracks (`task.RouteWhy`, `BarracksWhy`), Forgotten Tower (`towerWhy`, ladder point), Andariel and boss floors, gear gate, kit gate
(`kitGateActive`), readiness (`questboard`), quest floors (`legacyWall`), area bans, zone model shadow. They plug in today through `View.Legacy`
(asked last, refusal is `route_closed`); each becomes a row or a typed provider one at a time with the shadow-then-flip habit.

## Step 2 as built
- `internal/bot/admission.go`: `admitEnter` (called last in `areaGuardWhy`, after the route, Forgotten-Tower-road and Barracks guards) runs the old four row judges and
  `admit.MayEnter`; `admitLeave` (called by `fillEarly`) runs `questWalkCheck` and `admit.MayLeaveTown`. Under `bot.admission` 0 the old words are returned,
  a disagreement is logged once per case as `admission shadow` (area, purpose, old, new, row, need); under 1 the function's words are returned.
  The lean life line at the town exit has no old judge: logged as a difference, never acted on yet.
- `ops status --why` (the explain view) carries `admission`: mode, agree and differ counts, the last refusal with row, need and wake.
- First shadow finding (offline): the kit-floor words of admit lacked "unless a tier opens more"; aligned so the replay compares equal.
- Known old/new differences the shadow will show: the old lean judge ignores the owner's try anyway, the function exempts it.
- Property test `internal/admit/admit_prop_test.go` (T3 style, two archetypes): a reflex (corpse run, trip, try, non-Normal) is never refused, refusals
  are closed data, deterministic, the town is open, one more potion or a portal never closes a door.
- Not wired: `povertyFarmArea` and the early-road loop in `fillEarly` still call the old judges directly.

## Step 2 plan (original)
1. Chokepoint: the travel gate (outcomes2 TASK-958) and `guardedExit`/`routeGuardHop` (TASK-961) call `MayEnter`; the town exit calls `MayLeaveTown`.
2. Knob `bot.admission` (default 1, 0 = old path for one release). Shadow first: log `admission: old=... new=...` where they differ.
3. `ops status --why` / explain gets an `admission` line: reason, need, wake, row.
4. Invariants `field-entry-no-kit` and `field-entry-hurt` stay quiet for a refused entry because no entry happens.
5. Deletes after a clean 24 h watch: `barracksGuardWhy`, `denFloor`, `andarielLow`, `kitSafeTier`, `dungeonMouths` literals.
