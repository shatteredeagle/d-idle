# Fable review 2026-11-28, round 2 part B: the fixes for reviews 3 and 7 (fix3, fix7)

> status: review, read-only. Tree `829337379` (origin/main). Scope: merge `46365bb12` (fix3, TASK-1094) and merge `74ae9b205` (fix7, TASK-1096). No Go was changed, no game was used. Tags: **V** read at the cited line or run as named; **S** reasoned from the code, not run; **U** unproven. Paths are relative to the repo root; `bot/` is `internal/bot/`, `combat/` is `internal/combat/`, `arbiter/` is `internal/arbiter/`.

## 0. Verdict

Every finding the two cards name is closed in code as far as it was claimed (table in section 1). The fixes brought four new defects, two of them on safety paths, all behind `bot.arbiterControl` 1 or in the process watchdog:

1. **HIGH** N1: the new `Report` back-off can refuse a RUNNING EMERGENCY town trip and hand the tick to the recovery or gold farm hook (V at arbiter level, S at bot level).
2. **HIGH** N2: the new heartbeat lets facts the guard goroutine observes keep the process watchdog quiet while the acting goroutine is hung (V by reading).
3. **MEDIUM-HIGH** N3: the survive source lost its "hostile near" condition and answers a leave verdict (portal, chicken, crowd abandon) with a full `sweep`.
4. **MEDIUM** N4: the first-field floor is the first field of the hero's CURRENT act; for a lean hero in act 2 or later it is above the no-potion limit it is the floor of.
5. **MEDIUM** N5 to N7: band rule can pin the hero in the lowest zone; corpse reservation now retries an unusable corpse every 4 s; the stance leave rule makes a melee row kite.

Recommended before any family bit beyond 14 is set again: fix N1 and N3; N2 before the next unattended night whatever the knobs are (it is always on).

## 1. Are the reviewed findings closed?

| finding | state | evidence |
|---|---|---|
| R3-1 insidePack walks in hurt | closed for the low-life case (V `combat/stance.go:366` to 375: `low` no longer needs contact, hold when clear, `mayEvade` and `f.mv` at :300; rows shadow at knob 2 without `enabled`, :292; no row carries `enabled` in `combat/classdata/profiles.json`, V grep). Open: the surround leave is still followed by "inside" on the next tick at healthy life (N7b). |
| R3-2 no-portal recovery | closed as written (V `bot/broke.go:156` to 162, :203; `bot/resource.go:193`). Residual N4. |
| R3-3 band rule | closed (V `internal/survival/controller.go:241`, :391 to 408: after judging, current kept, all kept when nothing eligible is left). Residual N5. |
| R3-4 driveAura | closed for today's data (V `combat/aura.go:299` to 322; every `fightLifePct` row also has `restLifePct`, V grep of profiles.json). S: a future row with `fightLifePct` only lands behind the driven aura and never fires. |
| R3-6 corpses | closed (V `combat/aimcorpse.go:82` to 112, id 0 at :86). Residual N6. |
| R3-5 move rows, mouth gate, escape AND/OR | NOT in fix3 scope, still open (not re-read). |
| A1 low-life gate | closed (V `bot/arbiter_control.go:85`, :89). The gate equals the chain's; it declines with a hostile near or a potion in the belt, as in the chain. No new loop found (S): a failed way home counts `healFails` and ends in `noPortalLeave` or `lowLifeWait` exactly as on the chain path. |
| A2 survive | closed as claimed (V `arbiter/sources.go:78` to 95, `bot/shadow.go:137`, :158). New defect N3; the plain-trip case of A2 is still open by design. |
| A3 Report | closed (V `bot/arbiter_control.go:29` to 33, :38, :151). New defect N1. A3(c), the 40 min hold slice (`bot/goal.go:118`, `bot/broke.go:221`), is unchanged. |
| A4 heartbeat | half: heartbeat changed (V `bot/observe.go:389`), slice budget open (the card says so). New defect N2. |
| admission rows | closed (V `bot/earlyroute.go:62` to 111, `bot/povertyfarm.go:114`, `bot/admission.go:153` to 167). Brief question "does the 220 floor lock casters out of Act 2": no (section 3). |

## 2. New findings

### N1 (HIGH, `bot.arbiterControl` 1 with chores and recovery or goldfarm bits). Back-off refuses a running emergency trip; the farm hook takes the tick

- `Report(false)` backs off the key (source, kind) (`arbiter/arbiter.go:176` to 205); `refuse` applies it to every class below player (`:312` to 318). With the chores bit the one upkeep source uses ONE key for "the plan would buy", "a trip runs" and "emergency chores" (`arbiter/upkeep.go:51` to 66).
- Decline streak (the A3 shape, common): the plan has an open line but `needChores` says no (cooldown, town latch, reward hold). `arbChores` declines, back-off 30, 60, 120, 240 ... up to 1800 s.
- Then the guard's safety portal starts an emergency trip (`bot/loop.go:441`, :1021) and the hero stands in town, hurt, belt empty. `Decide` refuses the upkeep intent ("backed off") and chooses recovery (a broke hero) or goldfarm. `arbAct` runs `lowLifeGate` (returns false in town while `l.trip != nil`, `bot/goal.go:1850`) and then `recoveryStep`: a leveling slice that walks out of town with the heal and the refill not done. `took` is true, so the chain, whose line 62 puts a running trip above everything, does not run.
- Same shape without a trip: emergency chores DUE (belt 0) start only in `choresStep`; with upkeep backed off and a farm hook taking each tick, the guard cancels each slice with `errTripDue` (`bot/loop.go:1232`) and the next tick starts the slice again. No trip until the back-off ends (up to 30 min).
- V: throwaway test in `arbiter/` (not committed): `UpkeepSources`, facts recovery 1, plan open; `Report(false)` at t = 0, 30, 90, 210; view at t = 300 with `trip` 1, `trip_emergency` 1, `in_town` 1 gives `chosen recovery/farm_gold`, rejected `upkeep/upkeep: backed off 150s more after 4 failure(s)`. S: the bot half (not run; `levelSlice` has no trip check, V `bot/broke.go:219`).
- Before fix7 this could not happen: nothing called `Report`, so upkeep always outranked the objective class.
- Do: (a) every controlled hook first honours the chain prefix: `l.trip != nil` goes to `advanceTrip`, and `choresStep` is asked before a farm hook, as `lowLifeGate` now is; (b) `refuse` never backs off an intent with `trip` or `trip_emergency` set (or give the running trip its own key); (c) cap the hook back-off far below 1800 s, a decline is not a failure after cost.
- Must-fail test (two classes, `bot/arbiter_control_test.go` style): families 255; make `needChores` false with an open plan line for four ticks; then set `l.trip = &trip{emergency: true}` in town at 30 percent life with recovery active; assert the next `runGoalControlled` runs `advanceTrip` (why code `WhyTownTrip`), not `WhyFarmGold`. Second: belt 0 in the field, upkeep backed off: assert a trip starts within one tick.

### N2 (HIGH, always on). The watchdog heartbeat now moves on facts a hung action cannot stop

- `observer.bump` stamps `realWall` (`bot/observe.go:389`), and `LastStep` is `max(lastStep, realWall)` (`bot/runner.go:801`). `bump` is called from `observe`, which the GUARD goroutine runs on every poll while `f(tctx)` runs on another (`bot/loop.go:394` to 405). The comment "a step that hangs moves nothing" is false:
  - `life < o.monLife` (`bot/observe.go:779`): `life` is the SUM of life percent over the monsters in the snapshot, so it falls whenever a monster walks out of range, or is hit by the mercenary, a minion, an aura, poison or thorns;
  - a potion leaving the belt (`:876` to 881): the guard's own reflex drink;
  - hero position changing by 3 (`:582`): knockback, a held click that still walks;
  - gold, a drop, an area change.
- Scenario: the acting goroutine blocks in a call that does not return on cancel (a blocked helper write, a lock) in the field. Monsters hit the hero and drift in and out of the snapshot; the guard drinks. Each is a heartbeat. The in-process ladder is muted by the same facts (`stuckCheck` reads `lastProgress`) and can only cancel a context the hung call ignores. The process exit, the last rung, now comes only after the action timeout stops the guard (`:1124` to 1128) plus the limit: about 50 min for a 40 min slice instead of 10. Before fix7 only kills, XP and levels counted.
- A hero walking back and forth (the border loop, a ping-pong) is also "alive" for both judges now; only the drought and stall rules are left.
- Do: revert `realWall` to `bumpReal`, and give the watchdog a beat that only the ACTING goroutine can produce (the migration doc's step 3: `Beat` after each finished sub-step: a `GoTo` that returned, a cast that returned, a chore done). Drop the unqualified "monster life going down" (sum over a changing set) from any heartbeat.
- Must-fail test (`bot/runner_test.go`, fake clock): start an action whose `f` blocks on a channel and ignores `ctx`; feed the guard snapshots in which one monster leaves the list, then a potion leaves the belt; assert `LastStep()` does not advance. Run for a Sorceress and a Necromancer with a minion that damages a monster.

### N3 (MEDIUM-HIGH, survive bit). The danger fact has no hostile condition and its hook fights when the verdict says leave

- `inDanger` is `Safety(snap)` neither none nor a mana drink (`bot/shadow.go:158` to 164). The verdict asks for life actions with NO hostile near: "hurt, quiet and short of potions" returns `TownPortal` (`combat/safety.go:216`), a quiet life drink at :218 and :200. The old rule needed `hostile > 0`; the new branch at `arbiter/sources.go:84` does not (V throwaway test: facts `safety` 1, `hostile` 0, `trip` 1 chooses `survive/drink`).
- The hook is unchanged: `sweep` of the current area (`bot/arbiter_control.go:75`), a full pass that explores. So a quiet, hurt hero gets "find monsters here" in place of the chain's step (a plain chores trip walking home, the claim trip, rest). For `TownPortal`, `Chicken`, `CrowdAbandon`, `CrowdRetreat` the arbiter starts a fight and relies on the guard's first look inside it to do the opposite. S: where the guard suppresses the verdict (`noTP`, `bot/loop.go:1181`) the two disagree for the whole step.
- The fact flickers by construction: a life drink needs `HealReady` (`combat/fighter.go:289`), so after each reflex drink the fact is 0 for the potion gap while the hero is in the same danger; with an empty belt and no portal it is 0 from the drink line down to the chicken line. Under those conditions the survive family never fires (the brief's "never set" case; S: harmless, since the guard is the safety path, but then bit 1 does nothing a hero needs).
- `Safety` is not side-effect free (`noteDrinkSkip`, `noteCastRefused`, `mercDownLeave` log timers); the extra call per tick only shifts throttled log lines (V, low).
- Do: keep bit 1 clear. If it stays: the source returns an intent only for a drink verdict with `hostile > 0`; a leave verdict becomes an upkeep or reflex intent whose hook is the guard's own `handleSafety`, never `sweep`.
- Must-fail test (two classes): hero at 40 percent, no hostile in the snapshot, one potion, portal scroll, plain trip running; assert `runGoalControlled` advances the trip and never sets `WhyFight`. Second: verdict `CrowdAbandon`; assert no `sweep` starts.

### N4 (MEDIUM, always on). The floor of the no-potion rule is the current act's first field

- `starveStepEx` takes `first` = the first plan step of `s.Player.Act + 1` (`bot/resource.go:194` to 207) and returns it "whatever ban says" (:214). `homeBandArea` could pick a lower waypoint area, but for a lean hero (no portal, few potions: exactly this hero) `homeReach` is the first field only (:221 to 223).
- Scenario: Normal, act 2, clvl 15, belt empty, no scroll, broke. The rule's limit is monster level 12; every act 2 area is excluded; the floor returns the act's first field, whose monster level is above the limit (U: exact area levels come from `recoveryMLvl`, not read). Acts 3 to 5 are worse. The rule's own evidence row is a death at monster level 2 to 5 under these conditions.
- The rule is off outside Normal (`bot/povertyfarm.go:199`, unchanged).
- Do: when the floor itself is in `poor`, either go to the act 1 first field by the free town waypoint, or refuse with a typed need (sell, wait for the ban) instead of farming.
- Must-fail test (Sorceress and Paladin): act 2 town, clvl 15, no potion, no scroll, lean; assert the picked area's monster level is at most `clvl - noPotionGap` or the pick is refused.

### N5 (MEDIUM when `bot.zoneModel` is on). The band rule can pin the hero in the lowest zone

`inBand` keeps the current zone and drops the other below-band zones as soon as ANY kept zone is eligible (`internal/survival/controller.go:396` to 407). Scenario: clvl 15, `bandDrop` 6; current zone monster level 5 (eligible), zone A' level 8 (eligible, better pay), in-band zones locked or over the margin. Kept: current and the in-band zones; `safe` is true through the current zone; A' is dropped; the only eligible zone is the current one. The hero stays at level 5 until a lock reopens (`lockHoldMinutes` 480): the "sat in one field for a day" case the band was written against. S. Do: when the only eligible zone kept is below the band, keep every eligible below-band zone and let value choose. Test: `Choose` with those three; assert the decision is A'.

### N6 (MEDIUM, summoners; always on). A corpse the hero cannot use is retried every 4 s

Before fix3 a corpse was tried once per Fighter; now the reservation lapses after `corpseHold` (`combat/aimcorpse.go:82`, :104 to 109). A corpse that stays in the snapshot and cannot be used (walk fails, `combat/minions.go:304`; cast refused by the game) is offered again every 4 s for the whole fight: the summoner walks at it or spends the tick on it again and again. S; U whether `Corpses` filters unusable corpses. Do: count attempts per id and stop after two in one area. Test (Necromancer raise and Assassin Death Sentry): corpse 7 stays in the snapshot, the walk fails; advance the fake clock 20 s; assert at most two aims.

### N7 (MEDIUM, only when a row gets `enabled`)

- a. Under the life floor the hero leaves from ANY hostile within `standOff + 4` (`combat/stance.go:367` to 373). For `hammerdin_zeal` (a melee row, floor 50 percent) that is every target it attacks: below half life it alternates one swing and one step away. S. Test: zeal profile, life 45 percent, one hostile at 2; assert no "leave" while fewer than `leaveSurround` are in contact, or make the floor a per-kind field.
- b. Not fixed by fix3: at healthy life a surround leave is still followed by "inside" (contact 0 on the next tick), and with `standOff` 2 and contact radius 3 a pack of five is "in contact" as soon as the hero stands inside. In, out, in. Test: five hostiles in a 2 subtile cluster; run `packPlan` over three ticks; assert no "inside" directly after "leave".

### N8 (LOW)

- `earlyRouteCheck` asks level, life, kit per row in file order (`bot/earlyroute.go:96` to 109): for area 73 the tombs row's kit answer (`kitOnly` true) comes before the Duriel row's life floor, so a caster under 220 life and short of potions is told "kit", makes the trip and is then refused for life. Ask every row's level and life before any kit.
- The differ cell "words" has no counter (`bot/admission.go:160`); `Differ - OldOnly - NewOnly` gives it.
- Recovery and goldfarm declines back off to 1800 s too; harmless because the chain's own lines 111 to 115 still run them (V `bot/goal.go:111` to 116).

## 3. Does the caster floor at area 73 close Act 2? No

Road `act2_duriel_lair`: caster level 26, max life 220 (V `internal/plandata/admission.json`). Max life from the stat plans alone, no gear (S: class base life recalled, not read from game data): Sorceress plans 275 to 299 at clvl 26 (`build-sorceress_blizzard_endgame90` 229); Necromancer summoner plans 248 to 272; `build-necromancer_poison_nova_sme` 206 at 26 and 220 only at 28. So one build waits about two levels at the door, none is locked out. U: heroes whose points are unspent. The refusal is a level-type block although more levels are not what it waits for; worth a typed need "life".

## 4. Hold slice 40 min against the watchdog

Unchanged and still open: `arbHoldsQuest` hands out a 40 min slice (`bot/goal.go:118`), the lease is 120 s, and the arbiter decides only at action boundaries. fix7 made the watchdog tolerate the slice (N2) rather than shortening it, so the price of A4 moved from "false restart" to "late restart". The migration doc's step 3 (slice budget, beat from sub-steps) is the fix for both.

## 5. Not read (U)

`Plan.Pick` internals and `recoveryMLvl` values; `advanceTrip` beyond its first 30 lines; `admit.MayEnter` row order against the old judge (words cell); whether an exhaustive old-against-new admission test exists; the fix3 and fix7 tests themselves were not run here (only the throwaway arbiter test, `go test -p 1 -run RV2 ./internal/arbiter/`, deleted after).
