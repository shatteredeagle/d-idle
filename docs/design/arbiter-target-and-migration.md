# Arbiter: target architecture and migration order

> status: current design note (TASK-1096, card fix7). Source: `docs/design/fable-review-2026-11-28/07-arbiter-and-deciders.md` (sections 5 and 6), kept here as the one page to plan from, with what card fix7 already changed. Tags: **V** tested offline in this tree (test named), **S** reasoned, **U** not run live. Nothing here was run against a game.

## Recommendation (family bits)

`bot.arbiterFamilies` is a bit list: 1 survive, 2 player, 4 chores, 8 claim, 16 goldfarm, 32 recovery, 64 quest, 128 level.

| when | bits | why |
|---|---|---|
| now | 14 (player, chores, claim) | these run the chain's own hooks with the chain's own predicates: worst case is order, not a new behaviour |
| after A3 is merged and one control-0 night is scored with `-session` | add 64 + 128 | the veto is bounded: a declined hook reports and does not hold the quest step (fix7) |
| after A1 is merged and one more scored night | add 16 + 32 | the controlled farm hooks pass the low-life gate (fix7) |
| last | add 1 | survive reads the one safety verdict and yields to an emergency trip (fix7); needs the TASK-1034 lab rerun first |

The coordinator flips the knob; no card changes it on a sandbox.

## Target (one decider per concern)

```
            facts (one snapshot, one read per tick)
                         |
        +----------------+-----------------+
   SAFETY VERDICT                     ADMISSION
   Fighter.Safety, one function       admit.MayEnter / MayLeaveTown
   (drink, rejuv, portal, chicken,    (rows; route guard, Barracks, tower,
    retreat, heal in town)             boss floors, gear gate as providers)
        |   facts: safety=..., may_enter[area]=...
        +----------------+-----------------+
                         v
                     ARBITER  (one Decide per tick AND per guard look)
   sources: safety (reflex) | player (all player hooks) | upkeep (supply plan,
            heal, save point, checkpoint, claim) | objective (board, recovery,
            gold farm) | filler (level)
   each intent: class, urgency, area (from ONE PickArea), budget, resumable
                         |
                     EXECUTOR (one running step, budget <= 3 min, resumable)
   start / continue / abort(reason); always calls Report(intent, outcome)
                         |
                     GUARD = the same Decide, asked every watch tick
                         |
                     FIGHTER (target, skill, stance, aura: one wantX each, with dwell)
                         |
   FAILSAFE: one progress fact set, one ladder; the process watchdog reads the
             same progress facts and is the last rung
```

Rules:
1. The guard has no list of its own: "should the running step stop" is "would the arbiter now choose a higher class".
2. Safety is one verdict; no second life-line predicate anywhere.
3. "May the hero be there" is one function with providers; every caller reads it.
4. "Is a trip due and what does it buy" is the supply plan; cooldown, latch and reward hold become plan inputs or arbiter rules.
5. Every executor reports. No intent is chosen twice after a refusal without a changed wake fact.
6. No action longer than a third of the watchdog limit; long work is a resumable step.
7. A shadow lives at most one flip: the PR that flips a default deletes the old path.

## Migration order (12 steps)

| # | step | size | unblocks | state after fix7 |
|---|---|---|---|---|
| 1 | sandbox to families 14; ledger rows to `trial`; record the admission flip | config | honest state | coordinator |
| 2 | `Report` on decline and on hook result; bound the veto (A3) | S | bits 64, 128 | DONE for the six executing hooks (V `TestArbiterControlDeclinedHookReportsAndDoesNotVeto`, `TestArbiterRecoveryChoiceDoesNotVetoTheChunkTheChainAllows`). OPEN: the hold slice is still 40 min, not the dwell |
| 3 | slice budget for board, task and level slices; watchdog beat from sub-steps (A4) | M | every later step | PART: heartbeat only (below). The slice budget is open |
| 4 | one recorded night, control 0, shadow 1, tick log on; score with `-session` | run | the first real verdict | open (live) |
| 5 | a `heal` source from the low-life gate predicate; recovery fact equal to `recoveryDue` (A1) | S | bits 16, 32 | PART: the controlled farm hooks call the gate itself (V `TestArbiterControlFarmHonoursTheLowLifeGate`). The source and the equal fact are open, so the shadow still names recovery where the chain declines it |
| 6 | admission: area 73, lean under try, exhaustive equality test, `fillEarly` on the function, counter split | S | `bot.admission` default | PART: both differences fixed in the old judges, counter split (V `TestAdmissionDurielLairAsksEveryRoadRow`, `TestAdmissionLeanRowIsExemptUnderTry`, `TestAdmissionShadowCountsTheMixedCells`). Open: the exhaustive test, `fillEarly` reading the function, deleting the old judges |
| 7 | safety verdict as a fact; survive reads it and yields to an emergency trip (A2) | M | bit 1 | DONE offline (V `TestArbiterSurviveReadsTheSafetyVerdictAndYieldsToATrip`); U live, TASK-1034 rerun open |
| 8 | one player source for pause, recall, errand, vault keep, repair; save point and checkpoint as upkeep intents (A5) | M | chain lines for those hooks | open |
| 9 | the guard asks the arbiter, cause by cause, behind one knob | L | the guard list | open |
| 10 | one `PickArea` feeding `NeedsArea`; delete `holdTarget`, zone dwell, then flail, ping-pong, border loop (A6) | L | area deciders | open |
| 11 | the supply plan as the only trip judge | L | old trip deciders | open |
| 12 | aura: one `wantAura` with dwell, then `agency.planner` 2 (A7) | S | planner drive | open |

Steps 2, 3, 5, 6 are independent and offline; 4 needs one night. Nothing after 4 flips without a control-0 night scored.

## What fix7 changed, in behaviour terms

- **A1.** Under control, recovery and the gold farm order ask `lowLifeGate` first, as the chain does (`internal/bot/arbiter_control.go`). A broke hero below the resume line with an empty belt and nothing hostile near walks home. Not build-specific.
- **A3.** `runGoalControlled` calls the arbiter's `Report` for every family with an executing hook: acted clears the back-off, declined backs the intent off (30 s doubling to 30 min, `arbiter.Config`). A family whose own hook declined this tick holds nothing back, and a recovery or gold farm choice never vetoes the quest step (the chain asked `goldFarmOn` and `recoveryDue` just above and said no: the `boardChunkNeedsNoGold` rule stands). An uncontrolled family (a refill wanted with bit 4 off) still holds the quest step: that is the 0319 case the quest bit exists for.
- **A2.** New fact `safety` (1 while `Fighter.Safety` asks for a life action; a mana drink is not danger). The survive source reads it and nothing else; `survive_below` is read only for tick logs recorded before this change (replay). Survive does not fire while an EMERGENCY trip runs. It still outranks a plain chores trip: the incident case questy-0251 (a chores leg with a potion not drunk) scores that way, so "yields to every trip" would fail the outcome model. S: the walk home of a hero with no scroll is not an emergency trip and is not covered.
- **A4.** `observer.bump` stamps the heartbeat `Runner.LastStep` reads, so the d2bot watchdog (10 min, `cmd/d2bot/watchdog.go`) sees the same progress facts as the in-process stuck check: walking, an area change, gold, a potion, a drop, monster life going down, next to kills and xp (TASK-1030). A 40 min slice with ten quiet minutes of walking or town chores is no longer judged hung; a step that hangs moves nothing and is still caught (V `TestKillsAndXPAreAHeartbeat`). S risk: a hung acting goroutine while a mercenary or minions keep hurting monsters delays the exit until they stop; kills already had this property.
- **Admission.** The old road judge asks every row that holds the area (Duriel's Lair is on the tombs road and has its own row: the caster max life 220 floor is now enforced with `bot.admission` 0, and `fillEarly` blocks it for the task layer too). The lean entry row is exempt under the owner's try anyway, like every other row. `ops status --why` gains `agreeRefused`, `oldOnly`, `newOnly`; the log line `admission shadow` carries `differ=old-only|new-only|words`. A window with `agreeRefused` 0 exercised no row and proves nothing. Road floors are data (`internal/plandata/admission.json`), per archetype; nothing here is build-specific.

## Live checks (not run)

1. Control 0, shadow 1 night on one sandbox; score with `-session`.
2. Families 14 window: `arbiter-starved` count, deaths, chicken cycles.
3. A caster below 220 max life at the tombs: `admission shadow` shows no differ line and the hero is held (both refuse).
4. A board slice with a long quiet walk: no watchdog exit; a deliberately hung step still exits after the limit.
