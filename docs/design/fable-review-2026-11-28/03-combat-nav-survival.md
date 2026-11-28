# Fable review 3 of 8: combat, navigation and survival merged 2026-11-27

> status: review, read-only. Base `75e4c2f4e` (last commit before 2026-11-27 00:00) against `origin/main` at `415b11f25`. No Go was changed and no test was added; every finding is from reading the merged code (V = read at the cited line; S = inferred from the code, not run; U = not checked).

## What matters first

`data/darklaunch.json` records `combat.deliveryUpkeep` on trial at value 2 on one sandbox since 2026-11-27 (V, the row's `trial_value` and `note`). Everything described below as "mode 2" is therefore live on that hero now, including the stance the code comment and the profile rows themselves say must wait: `internal/combat/stance.go:26` ("knob 2 waits for the survival model's contact archetype") and the three `insidePack` rows in `internal/combat/classdata/profiles.json` (hammerdin, hammerdin_zeal, poisonnova; each source says "must judge before knob 2"). One knob switches upkeep, mark, aim, corpse aim, stance, press, move and whirl together, so the upkeep trial switched the unjudged ones on with it.

## Findings, ranked by risk to a live hero

### 1. HIGH: insidePack walks a hurt hero into the pack; the leave rule needs a hostile in contact (mode 2, live on the trial sandbox)

`internal/combat/stance.go` `packPlan`:

```go
if contact >= s.LeaveSurround || (contact > 0 && p.MaxLife > 0 && p.Life*100 < p.MaxLife*s.LeaveLifePct) {
```

- With no hostile within `stancePackContact` (3 subtiles) the life rule is off. A hammerdin or poison-nova necromancer at 20 percent life facing archers, shamans or any pack 5 to 30 subtiles away gets `"inside"`: a walk to `standOff` (2 or 3) subtiles from the centroid of every hostile within `stanceThreatRange` (30).
- After a `"leave"` (to `standOff+4`) the next tick has contact 0, so the plan is `"inside"` again: leave, walk back in, leave, at low life.
- The centroid counts hostiles within 30 subtiles with no line-of-sight or reachability test, so the stance pulls the hero toward packs the fight loop has not engaged.
- `stanceTick` checks `evadeOK` but not `f.mayEvade()` or `f.mv != nil`, unlike every other caller of `f.evade` (`fight.go:565`, `:676`, `:694`, `meleeGuard`). A hero the pin rule put in "fighting in place" (`pinHold`) is still stepped by the stance every tick after a cast; this is the stall the pin rule was written for (the comment at `fight.go:1018` cites 10 minutes without a cast).

Knob: `combat.deliveryUpkeep` 2 (default 0; 2 on the trial sandbox). Recommendation: take the three `insidePack` rows out of the data until the model has judged them (the knob cannot separate them), or give the stance its own switch; make the life rule independent of contact, refuse `"inside"` under `leaveLifePct`, and add `mayEvade()`.

Must-fail test sketch (two builds: hammerdin, poisonnova): snapshot with life 20 percent, four hostiles at 10 subtiles, none within 3; assert `packPlan` does not return `"inside"`. Second: set `f.noEvadeUntil` in the future, assert `stanceTick` returns false.

### 2. HIGH: the no-potion recovery rule is skipped for a hero without a town portal (always on)

`internal/bot/broke.go:153`: `recoveryAreaBase` returns through `starveStep` when `!HasTP`, before `recoveryPoorExclude` (`:188`) runs. The incident the rule was written for is a clvl 4 hero with 15 gold and an empty belt, which cannot own a portal scroll; the test (`internal/bot/recoverydeath_test.go:22`) adds a scroll to every case, so this path is not covered.

In the no-portal branch `starveStep` calls `homeBandArea` (`internal/bot/resource.go:229`), which picks the plan area of the hero's own level among waypoint-reachable areas. Only `kitGateActive` (act 1, Normal, kit gate holding) keeps it on the first field. Scenario: clvl 12 to 20 hero in act 2 or later, no potion, no scroll, broke: the recovery farm goes to an own-level area with an empty belt.

The same fallback is reached with a scroll when the exclusion leaves nothing (`Plan.Pick` fails at `broke.go:195`): at clvl 1 to 3 the limit `clvl - noPotionGap` is zero or less and every area is excluded; later, when the allowed low areas are cleared or banned. `starveStep` does not read `ex`.

Knob: none; `poverty.recoveryFromLevel` is 1 in `internal/plandata/admission.json`. Recommendation: apply `recoveryPoorExclude` in the no-portal branch and pass the exclusion into `starveStep`/`homeBandArea`, with the first field as the floor.

Must-fail test sketch: the existing table without the portal scroll, once in act 1 with `RecoveryKitGate` 0 and once in act 2 at clvl 16 (Sorceress and Paladin); assert the picked area's monster level is at most `clvl-3` or the area is the act's first field.

### 3. HIGH when `bot.zoneModel` is on: the band rule can remove every safe zone and the hole fallback then picks the hardest one left

`internal/survival/controller.go:235`: `in.Cands = l.inBand(in)` runs before any zone is judged, and "one zone is left" (`:402`) counts zones that are locked, forced out after a death, over the margin or depleted.

Scenario (Normal, clvl 15, `bandDrop` 6): candidates A (monster level 8, safe, eligible) and B (monster level 12, hero died there or margin low). `inBand` drops A, `judge` finds B not eligible, `len(elig) == 0`, and the hole branch returns B with the words "the easiest field". With a full kit `zoneModelPick` (`internal/bot/zonemodel.go:257`) walks there. Before today A was picked.

Also: the hero's current zone can be dropped, so `cur` is nil: the stay hysteresis (`SwitchMargin`) and `MarkLeft` are skipped for it.

Knob: `bot.zoneModel` (default 0, row status dark). U: whether any sandbox runs it at 1; the card's own evidence is a hero that "sat in Dark Wood for a day" under the model. Recommendation: apply the band after judging, among eligible zones only, and never to `in.Current`.

Must-fail test sketch: `Choose` with A and B as above, B forced out by a recorded death; assert the decision is A and `Hole` is false. Second: current zone below the band and the only eligible one; assert `Stay`.

### 4. MEDIUM-HIGH: the planner's aura overrides the low-life and resting auras and the immunity skip (agency.planner 2)

`internal/combat/aura.go:296` `driveAura` prepends `AuraPref{Name: a}`, a plain entry. In `wantAura` a plain reachable entry returns at once (`:126`), so behind it the profile's `FightLifePct` aura (the low-life fight aura), the `RestLifePct` aura while resting, `NeedsCrowd` and the element-immunity skip (`:123`) are never reached. A Paladin driven to Holy Fire keeps it against a fire immune and at 20 percent life.

Knob: `agency.planner` 2 (default 0, owner go). Recommendation: let the Drive name re-order the ladder (move the matching profile entry to the front with its fields) instead of adding a bare one, and keep the life entries ahead of it.

Must-fail test sketch (Paladin default and hammerdin): Drive names the crowd aura; target immune to its element, then life under the profile's fight line; assert `wantAura` does not return the driven aura.

### 5. MEDIUM: move rows cast outside the mouth gate, are not counted as casts, and the escape trigger is AND where its source says OR (mode 2)

- `internal/combat/move.go` `escapeCell`, `gapClose` and `internal/combat/whirl.go` `whirlLeg` choose a landing with `f.valid` only. `f.move` refuses a point near a cave mouth (`fight.go:972`) and `swingIntoMouth` asks the mouth gate for left-hand clicks; Leap Attack, Charge and a Whirlwind leg are movement and ask neither. S: a leg or leap that ends in an exit's trigger area repeats the bounce the mouth fixes were written for.
- `escapeCell` measures distance only to hostiles within 30 subtiles; the farthest cell 18 away can be next to a pack just outside that ring. No hop cooldown.
- `escapePlan` (`move.go:226`): `contact < r.EscapeSurround || pct >= r.EscapeLifePct` returns "hold", so both conditions are required. The Sorceress rows' source reads "three in contact or under 75 percent life". A Sorceress at 20 percent with two in contact does not hop. Not a regression (no hop existed), but the SME tuned a rule the code does not implement.
- `moveTick`, `whirlTick` and `gapClose` call `f.cast` without `st.noteCast`, so a fight made of whirl legs counts no cast: the watchdog (`castfirst.go:87`) forces casts into it and `evadeOK` stays false.

Knob: `combat.deliveryUpkeep` 2. Test sketch: `whirlLeg`/`escapeCell` with a mover whose `NearMouth` is true for the best cell, assert another cell or none; `escapePlan` at life 20 percent with `EscapeSurround-1` in contact, assert the documented rule.

### 6. MEDIUM: corpse reservation is taken on aim, shared, and never released

`f.takeCorpse` runs after `f.cast` returns nil, whether or not the cast landed (`minions.go:248`, `aimcorpse.go:229`, `necro.go:266`, `:297`), and on a failed walk (`minions.go:304`). `necroState.tried` is created once (`necro.go:146`) and never cleared. Today's change makes Revive, Corpse Explosion, the minion driver and Death Sentry (`assassin.go:71`, was `nil`) read the same set, so one interrupted Raise cast removes that corpse for all four. `assassin.go:92` reserves id 0 when no corpse stands exactly on the lay point. S: unit ids start low in every game, so a long-lived Fighter refuses fresh corpses that reuse old ids; a summoner that cannot rebuild its army is the hero the `behindArmy` stance then leaves with no front. U: the Fighter's lifetime across games. The necromancer part predates today and is always on; the sharing is new.

Test sketch: aim a raise at corpse 7, cut the cast (no new minion, corpse still in the snapshot), assert corpse 7 is offered again after a timeout; new area or game resets the set.

### 7. LOW

- auraflap (`aura.go:120`, always on): a crowd aura already on is held for the whole fight and 30 s after, also against a single boss. Immunity, low-life and resting entries still win (V). Damage loss only.
- War Cry kind melee, range 6: `meleeGo` (`fight.go:745`) skips the line-of-sight test for melee kinds on the assumption that a left click walks the hero; a right-hand cry does not walk, so it is cast at targets behind a wall within 6 subtiles until the no-progress rule drops them. Only the profile band that requires War Cry.
- `lockHoldMinutes` 480: a zone left for cost reopens after 8 hours with no power change (`ledger.go` `LockedAt`); the death rule (`ForcedOut`) is separate and unchanged.
- necroapply, palapply, classapply table edits: no skill made unreachable. S: `internal/leveler/prereq_test.go` shows the leveler buys a missing prerequisite first; `internal/plandata` tests pass (V, `go test -p 1 ./internal/plandata/ ./internal/survival/`, both ok). Point timing shifts when a prerequisite is bought late.

## Read lightly or not at all

- Read lightly, nothing found: fieldstate (`errFieldCrowd` hand-over), holereach (unreachable drop, hole needs `fullKit`; an empty list returns to the plan tables), cellar3gate (`leavePocket` refusal is now named), admission shadow (acts only under `bot.admission`), nav `TeleportHopFn` (a refusal means no Teleport and a walk).
- Not reviewed: hostilespin, countessnav fixture, combattelem summaries, aim point rows (`aim.go`), mark and minion drivers beyond the corpse sites, the whirlscen drills.
- Nothing here was run against a game or the Proving Ground.
