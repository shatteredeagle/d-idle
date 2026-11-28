# Fable review 2026-11-28, part 7: the arbiter and the deciders

> status: review, read-only (no code changed, nothing run against a game). Tree: `415b11f25` (origin/main at the time of the read). Tags: **V** read in this tree (file:line given) or in a named repo doc; **S** reasoned from the code, not run; **U** unproven. Paths are relative to the repo root; `bot/` means `internal/bot/`.

## 0. Verdict in ten lines

1. There is no single decider per concern yet. A tick passes through FOUR layers that each hold their own precedence list: the pre-goal hooks of `step`, the arbiter (when it acts), the `runGoal` chain, and the guard inside a running action. The arbiter sits in the second and sees the hero only at action boundaries.
2. `bot.arbiterControl` 1 with families 255 is NOT justified by the evidence the ledger row asks for (section 3). It is tolerable as a sandbox trial; it must not become a default, and three bits should be cleared or fixed first (survive, goldfarm, recovery).
3. Under control two safety paths can be starved by the arbiter itself: the low-life gate (skipped for a broke hero, finding A1) and a running trip home (outranked by the survive family, finding A2).
4. The arbiter is open-loop: nothing in `internal/bot` calls `Shadow.Report`, so the back-off never applies live, and a chosen family whose hook did nothing still blocks the quest step for up to a 40 minute slice (A3).
5. `bot.admission` 1 on Doofer (a melee hero in Act 1) is low risk because the control path is the old judges by construction, but for the same reason the 2 h window cannot fail and proves nothing. Two real differences exist (lean under try-anyway; overlapping road rows at area 73 for casters), and the second is a live bug in the OLD path today (section 4).
6. The 673/0 and 753/0 counters cannot carry a flip: the counter does not split "both open" from "both refused", and on those two heroes most rows could not fire at all.

## 1. Who decides what in a tick

### Layer 0: before the goal (`bot/loop.go`, `step`, lines 60 to 139), strict order

| # | decider | can do | line |
|---|---|---|---|
| 1 | game lost / relaunch | leave, restart | 70 |
| 2 | `stuckCheck` -> `escalate` (failsafe ladder: skip target, Esc, town reset, new game, restart, pause) | move, change area, leave game; clears `l.trip` (684) | 79 |
| 3 | death wait, corpse recovery, `corpseTownStep` | change area | 96 to 107 |
| 4 | `campaignStall`, `droughtCheck` | new game, change area | 98, 101 |
| 5 | `resetStep` (player reset) | leave game | 110 |
| 6 | `errorWait` (holds; `hazard` may drink or leave) | drink, chicken | 116 |
| 7 | `guardedExit` (route guard, tower, Barracks, the admission rows for the area the hero stands in) | walk out of an area | 119, `bot/routeguard.go:290` |
| 8 | pause (`hazard` or `pausedFight`, a 15 s Clear with no guard goroutine) | fight, drink | 127 to 137 |
| 9 | `runGoalShadowed` -> arbiter and chain | everything else | 139 |

### Layer 1: the arbiter under control (`bot/arbiter_control.go:19`)

`Decide` runs before the chain. Classes: reflex > player > upkeep > objective > filler (`internal/arbiter/sources.go:76` to 161). What the bits really do (V lines 58 to 82, 123 to 139):

| bit | family | what control does | what it does NOT do |
|---|---|---|---|
| 1 | survive | runs `sweep` in the current area when life is under `combat.drinkLifePct` with a hostile in the clear radius | does not drink, portal or chicken (the guard does); does not check `l.trip` |
| 2 | player | `playerActionStep` first | recall, errand, pause, vault keep have no source |
| 4 | chores | `advanceTrip` or `choresStep`; switches the sources to the one upkeep source with supply-plan facts (`bot/shadow.go:251`) | the old `needChores` still decides whether a trip starts |
| 8 | claim | `claimTripStep` | |
| 16, 32 | goldfarm, recovery | `goldFarmStep`, `recoveryStep` (a 40 min `levelSlice`, `bot/broke.go:215`) | skips every chain hook above them |
| 64, 128 | quest, level | only a VETO: `arbHoldsQuest` turns the quest step into a level slice when the arbiter chose another family | never runs `boardStep`, `taskStep`, `travel` or `sweep`; the chain still picks the area |

So "255" means six executing hooks and one veto. The level and quest steps, and with them every area choice, stay in the chain.

### Layer 2: the chain (`bot/goal.go:24` to 134), first match wins

prepare 30; points 39; checkpoint 42; player pause 45; recall, player items, errand, vault keep 48; idle goal returns 52; trip in progress 62; repair trip 65; belt fill 68; flail hold 71; chores 74; low-life gate 77; gear and claim in town 80; save point 85; claim trip 88; difficulty 91; farm runs 100; board preempt 103; board holds area 106; gold farm order 111; recovery 114; arbiter hold 116; board step 120; task step 122; target, rest, travel, sweep 124 to 134.

Area choice inside `target` has its own stack: `target0` (level plan, zone model, poverty farm), `farmReturn`, the route-ban loop (976), `holdTarget` (3 and 10 min, 1007), the zone dwell (`bot/zonemodel.go:19`, 3 min), the flail guard, the admission rows through `areaGuardWhy`.

### Layer 3: the guard inside an action (`bot/loop.go:1143` to 1236), first match cancels

game event 1148; menu 1155; fight stall 1163; death 1167; pause 1171; SAFETY verdict of `Fighter.Safety` (drink, rejuvenation, portal, chicken, crowd abandon; suppressed by `noTP`, by `walkHomeNoScroll`, and while a portal cast is in flight; a life drink goes out as a reflex without cancelling) 1173 to 1196; chores outside town 1200; field crowd on the return leg 1204; drought 1208; replan 1213; player order 1218; checkpoint 1222; points 1226; trip due 1230; stuck 1234.

The guard exists only while `runAction` runs (`bot/loop.go:368`). Actions run 40 minutes at most (`bot/questboard.go:1007`, `bot/task.go:235`, `bot/broke.go:215`). Inside one action the guard's list IS the precedence; layers 0 to 2 are not consulted.

### Layer 4: inside the fight

`combat` picks target, skill, stance, aura (`internal/combat/aura.go:150`: profile ladder, band and rest rules, plus the planner's `Drive` prepended at `agency.planner` 2, line 296) and the upkeep driver (`combat.deliveryUpkeep`: mode 2 replaces the "is it up" read, lines 160 to 196). Mana drinks in the Clearing state are the fighter's, not the guard's (1186).

### Outside the process

The d2bot watchdog exits the process after 10 min without a finished loop iteration, a kill or an XP gain (`cmd/d2bot/watchdog.go:16`, `bot/runner.go:799`).

## 2. Where deciders still disagree, or starve a safety path

Ranked. "Blocker" means: blocks making arbiter control a default or putting it on a played hero; none of them stops the sandbox trial tonight.

### A1 (blocker). Recovery and gold farm under control skip the low-life gate

- Chain: `choresStep` 74, then `lowLifeGate` 77, then (much later) recovery 114. The gate sends a hero below the resume line with no potion and no hostile near to town for the free heal (`bot/goal.go:1840`).
- Control: when the arbiter picks recovery or goldfarm, `arbAct` calls `recoveryStep` / `goldFarmStep` directly (`bot/arbiter_control.go:72` to 79). Neither checks life (`bot/broke.go:207` to 227; V `belowResume` has no caller there). No source proposes "heal": survive needs a hostile, upkeep needs a running trip or a payable plan line (`internal/arbiter/upkeep.go:216` to 239: short of gold with nothing to sell returns nil).
- So a broke hero, belt empty, below the resume line, in the field: chain walks home; arbiter starts a 40 min farm slice. S (code path, not run).
- Evidence that fits (cause U): the row `poverty.recoveryEvidence` in `internal/plandata/admission.json` records "Doofer clvl 4, 15 gold, belt 0, killed ... while Recovery: farming gold (2026-11-26 23:35 EST)". Families 255 went on at 2026-11-26T23:25Z, about five hours earlier (V `data/darklaunch.json`, row `bot.arbiterFamilies`). The fix was a new data floor from one death (cause class G); nobody recorded whether the gate had been skipped.
- Do: clear bits 16 and 32 on Doofer until an offline test (two builds) shows the hero below the resume line goes home under control. The right fix is a source, not a call: a `heal` intent (reflex or top upkeep) built from the same predicate as `lowLifeGate`.

### A2 (high). The survive family outranks a running trip, and is a second safety rule

- `arbAct` survive (`bot/arbiter_control.go:59` to 65) has no `l.trip` check; reflex beats the trip's upkeep class. The chain puts a running trip (62) above everything that fights.
- V lab incident: TASK-1034 result, 169 chicken and re-enter cycles in a virtual hour "survive hook fighting mid-leg". The fix covers only the leg back out of town (`bot/goal.go:747`). The field legs stay: the walk home of a hero with no scroll (the case `walkHomeNoScroll` protects from portal cancels, `bot/loop.go:489`) is replaced tick by tick by a sweep. AC 2 of that card (the lab rerun) is still open.
- The survive predicate (`life < drinkLifePct`, hostile near) is not the safety verdict (`Fighter.Safety`). Two rules for "the hero is in danger", no hysteresis: life crossing the line flips trip and sweep every tick. S.
- The hook is `sweep`, a full pass (buff, fight, pick up, explore, `bot/goal.go:1496`), not "fights where it stands" as `docs/arbiter.md` says.
- With an empty belt the survive intent holds the hero in the fight while the retreat for potions (trip due, cancelled by the guard at 1230) is outranked each time it comes back. Bounded only by the chicken line. S.
- Do: clear bit 1 until survive is driven by the one safety verdict and yields to a running emergency trip. The guard already drinks in every state; the bit adds only "no new step starts", which the guard's cancel gives anyway.

### A3 (high). Open loop: no Report, and a hook that did nothing still vetoes the quest

- `grep '\.Report(' internal/bot` finds no arbiter call (V). Back-off, "failed after its cost" and success never reach the arbiter live; replay has them (`-follow`), the bot does not. The replay verdicts therefore describe a different machine.
- When the chosen controlled hook does nothing, `took` is false and the chain runs with `r.arb.chose` set (`bot/arbiter_control.go:30` to 41). At 116 `arbHoldsQuest` then returns true for any family but quest, and the hero gets "Leveling: the arbiter holds the quest step", a 40 min slice.
- Concrete regression (S): `recoveryDue` is `recoveryActive && !gearGated && !boardChunkNeedsNoGold` (`bot/goal.go:195`). The chain, when a board chunk needs no gold, skips recovery and runs the chunk. Under control the fact is only `recoveryActive` (`bot/shadow.go:133`), recovery wins (urgency 0.8 over quest 0.5), its hook declines, and the veto replaces the chunk with leveling. The rule the chain wrote on purpose is undone.
- Same shape: upkeep says "plan would buy" while `needChores` says no (cooldown, town latch, reward hold: two deciders for "is a trip due", `bot/goal.go:344` and `internal/arbiter/upkeep.go:214`); a player order pending with a hostile near; a claim pending with a hostile near.
- Do: (a) call `Report(in, now, false)` when a controlled hook declines, so the intent backs off; (b) the veto only when the chosen family's hook ACTED this visit or holds a lease that was earned; (c) the hold slice no longer than the arbiter dwell.

### A4 (high). Forty minute actions make every decider above the guard blind, and two judges answer "is it hung"

- V TASK-1048 result: one 40 min board slice outlasted the 10 min watchdog. The loophang change (TASK-1030, kills and XP as heartbeat) hides it while the hero kills; a slice with ten quiet minutes (town chores, a hold, a long walk) still ends in a process exit that bypasses the ladder.
- In-process `stuckCheck` counts walking, gold and potions as progress (`bot/observe.go:371` to 394); the process watchdog counts loop iterations, kills and XP. Different facts, different limits, different reactions. Class A between failsafe rungs.
- Arbiter control is nominal while a slice runs: lease 120 s, slice 2400 s.
- Do: a slice budget of at most a third of the watchdog limit for every `do` (resumable steps exist: the trip, the board park); the watchdog heartbeat from the acting goroutine's own sub-steps (`Beat`, `bot/runner.go:792`), not from kills.

### A5 (medium). Player commands other than item actions lose their rank under control

Points 39, checkpoint 42, player pause 45, recall, errand and vault keep 48, repair trip 65, save point 85 have no source. Whenever the arbiter acts (a long gold farm order, a trip) they wait. "The player ranks right behind safety" holds in the chain and in the guard, not in the arbiter. A save point never fires during a controlled gold farm slice chain (XP at risk). S. Do: one `player` source that covers every player-origin hook, and `checkpoint`/`savepoint` as upkeep intents, before any more bits.

### A6 (medium). Area choice is outside the arbiter and has five hysteresis mechanisms

`holdTarget`, zone dwell, arbiter lease and travel budget, flail guard, route-ban avoid list. The travel budget never binds for leveling (`NeedsArea: Any`, `internal/arbiter/sources.go:160`) and `ObjectiveArea` is the last planner step area (U, `bot/shadow.go:151`). The ping-pong replay (412 against 2 area changes) is a property of the model, not of the control path as wired. With `bot.zoneModel` on (the brief says it is on both sandboxes; the ledger row still says `dark`, V) a third chooser joins. Do: area choice becomes ONE intent field (`NeedsArea` from one `PickArea`), and the travel budget is the only anti-ping-pong rule; then delete the others one by one.

### A7 (medium). The aura has three writers and no dwell

V audit `docs/probes/sandbox-postdeploy-audit-2026-11-27.md` problem 3: 95 and 97 swaps in 70 min on Questy. `wantAura` reads the profile ladder, the band and rest rules and, at `agency.planner` 2, the planner's aura prepended (`internal/combat/aura.go:296`). The planner shadow's `agree` ignores the aura. Do before mode 2: one `wantAura` result with a reason and a minimum dwell; a test that one fight never swaps twice (two builds); `agree` includes the aura.

### A8 (medium). The measurements cannot see the above

- Under control, agreement is by construction (`bot/arbiter_control.go:105`) and the tick line's `chain` is the arbiter's family (docs/arbiter.md says so itself). A Doofer night under 255 cannot feed the session verdict.
- `arbiter-starved` fires as a standing fact (audit problem 7).
- The admission counter lumps both-open with both-refused (`bot/admission.go:146`).

### A9 (low)

- `escalate` drops any trip, an emergency one too (`bot/loop.go:684`).
- Fight stall is checked before safety in the guard (1163 before 1173): one guard look without a safety verdict.
- A portal "in flight" returns from the guard before any other check (1184); a stuck flag would mute the guard (U).
- Idle goal: the chain returns at 52 before any trip; `arbChores` has no idle check (S, not traced through `needChores`).

## 3. Is families 255 on Doofer justified?

No, by the ledger's own text; yes, as a labelled experiment.

| the row asks for (V `data/darklaunch.json`, `bot.arbiterControl`) | what exists |
|---|---|
| verdict passes per family | passes offline on MODEL cases whose facts are U (docs/arbiter.md) |
| incident rows green on a lockstep run | none run (docs/arbiter.md, "U: none run yet") |
| 24 h watch | windows of 1800 s; row says `enabled` with `criteria_met: no` |
| owner go per family | one bundle flip |
| a recorded night scored with `-session` | none (TASK-973 result) |

Live evidence: 70 min, 0 deaths, 5 `arbiter-starved` (audit). Lab evidence against: the arbiter255 soak ended in a watchdog exit (TASK-1048; explained by a binary without loophang, not rerun on record) and the chicken loop under families 3 (TASK-1034, rerun open). The admission counters 673/0 and 753/0 say nothing about the arbiter.

The 40 min slice against the 10 min watchdog is not an arbiter bug, but it removes the premise of control (A4).

Recommended state on Doofer now: `bot.arbiterFamilies` 14 (player, chores, claim). These three run the chain's own hooks with the chain's own predicates, so the worst case is order, not a new behaviour. Clear 1 (A2), 16 and 32 (A1), 64 and 128 (A3: the veto with no Report). Set the ledger rows to `trial`.

Safe next flips, in order, each after its fix and one control-0 recorded night scored with `-session`:
1. 64 + 128 after A3 (Report wired, veto bounded by the dwell).
2. 16 + 32 after A1 (heal source) and the `recoveryDue` fact made equal to the chain's.
3. 1 last, after survive reads the safety verdict and yields to a running trip (A2).

## 4. Admission: shadow to control

### What the control path is

`areaGuardWhy` asks the route guard, Barracks and the Countess-level guards first, then `admitEnter` (`bot/routeguard.go:178` to 188). With `bot.admission` 1 the words of `admit.MayEnter` replace the words of the four old judges; the old judges still run and the differ counter still counts (`bot/admission.go:79` to 93, 125 to 131). `admitLeave` takes only the quest-walk row; the lean exit line is log-only (95 to 122).

### What can differ from the old judges (V by reading both sides)

1. **Lean under try-anyway.** `leanEntryWhy` has no `TryActive` exemption (`bot/povertyfarm.go:112` to 121); `admit` exempts every row (`internal/admit/admit.go:90`). With the owner's try on, a lean hero below 60 percent life is held by the old judge and let through by the function. The function's answer is arguably the intended one; it is still a change and has no test that names it.
2. **Overlapping road rows.** Area 73 is in `act2_tombs` and in `act2_duriel_lair` (`internal/plandata/admission.json`). The old judge takes the FIRST row that holds the area (`bot/earlyroute.go:61` to 71); the function checks ALL (`admit.go:128` to 144). A caster with max life 150 to 219 is let into Duriel's Lair by the old judge and refused by the function. `TestCasterLifeFloorsSewersAndDuriel` asserts the function's answer; the live path (`bot.admission` 0) does not enforce the 220 floor today. That is a live bug in the old path, and under `bot.admission` 1 `fillEarly` still builds `EarlyBlocked` from the old judge (`bot/routeguard.go:512`), so the task layer says open and the hop refuses: the "refused at the hop, counted as too weak" case `fillEarly` exists to prevent. Melee numbers are equal, so Doofer is not affected.
3. Unreadable rows fall back to the old judges (fine).
4. `View.Knobs` and `View.Legacy` are never filled (`bot/admission.go:44` to 54): the `den_of_evil` knob floor in the data is applied by neither path through `admit`. Not a difference, but the data file's comment claims it.
5. `MayEnter` is always asked with purpose `level` or `corpse-run`; farm, boss and quest-walk purposes are unused.

### What the counters prove

Both heroes are Paladins (melee) at clvl 17 and 16 with 0 deaths in the window (audit). The quest-walk row ends below clvl 15, so `MayLeaveTown` could not refuse; corpse-run, try, caster rows, Act 2 roads and (likely) poverty never fired. The counter counts "both open" as agreement. So 673/0 and 753/0 prove: the kit-floor row and the open case agree for a melee hero in Act 1. Nothing more.

### Was the 21:51 flip on Doofer safe?

Yes, low risk: melee, Act 1, equal by construction, the differ counter keeps running and shows in `ops status --why`. Watch: `differ` stays 0; `field-entry-no-kit` and `field-entry-hurt` do not rise. It is not evidence: a window that cannot fail closes clean. The ledger row still reads `dark` in this tree; record the flip.

### Before `bot.admission` 1 on a played hero or as default

- Fix difference 2 (one row per area, or first-match in both) and decide difference 1; test both by name.
- An exhaustive equality test old against new over area x archetype x level x life x kit x try x corpse-run (pure functions, cheap), not sampled.
- The counter split into open/open, refused/refused, and the two mixed cells, per row.
- `fillEarly` reads the function too, so the task layer and the guard cannot split.
- A hero that exercises the rows: clvl 5 to 14 (quest-walk), a caster, one corpse run, one Act 2 road. Not Doofer at 17.
- Then delete the old judges in the same PR as the flip to default; a shadow that stays forever is a second decider.

## 5. Target architecture (one page)

```
            facts (one snapshot, one read per tick)
                         |
        +----------------+-----------------+
        |                                  |
   SAFETY VERDICT                     ADMISSION
   one function, Fighter.Safety       admit.MayEnter / MayLeaveTown
   (drink, rejuv, portal, chicken,    (rows; route guard, Barracks, tower,
    retreat, heal-in-town)             boss floors, gear gate as providers)
        |                                  |
        |   facts: safety=..., may_enter[area]=...
        +----------------+-----------------+
                         v
                     ARBITER  (one Decide per tick AND per guard look)
   sources: safety (reflex) | player (ALL player hooks) | upkeep (supply.Solve plan,
            heal, save point, checkpoint, claim) | objective (board, recovery,
            gold farm) | filler (level)
   each intent: class, urgency, area (from ONE PickArea), budget, resumable
                         |
                     EXECUTOR  (one running step, budget <= 3 min, resumable)
   start / continue / abort(reason) ; always calls Report(intent, outcome)
   typed outcome: done | progress | refused{need, wake} | failed | preempted
                         |
                     GUARD = the same Decide, asked every watch tick:
   cancels the step iff the arbiter would now choose a higher class
                         |
                     FIGHTER (target, skill, stance, aura: one wantX each, with dwell)
                         |
   FAILSAFE: one progress fact set, one ladder; the process watchdog reads the
             executor's sub-step beat and is the last rung of the same ladder
```

Rules that make it one decider per concern:
1. The guard has no list of its own. Its fifteen cancel causes become sources; "should the running step stop" is `Decide(view).Chosen.Class > running.Class`.
2. Safety is one verdict, read by the reflex source and by the reflex drink; no second life-line predicate anywhere (the survive source's `drinkLifePct` rule goes).
3. "May the hero be there" is one function with providers; `guardedExit`, `target0`, the board readiness floors and the task layer's early blocks all read it.
4. "Is a trip due and what does it buy" is `supply.Solve`; `needChores`, the cooldown, the latch and the reward hold become plan inputs or arbiter rules (lease, back-off), then go.
5. Every executor reports. No intent is chosen twice after a refusal without a changed wake fact.
6. No action longer than the watchdog limit divided by three; long work is a resumable step.
7. A shadow lives at most one flip. The PR that flips a default deletes the old path.

## 6. Order of migration steps

| # | step | size | unblocks |
|---|---|---|---|
| 1 | Doofer to families 14; ledger rows to `trial`; record the admission flip | config | honest state |
| 2 | Wire `Report` on decline and on hook result; bound the veto slice by the dwell (A3) | S | bits 64, 128 |
| 3 | Slice budget for `boardStep`, `taskStep`, `levelSlice`; watchdog beat from sub-steps (A4) | M | every later step; closes the TASK-1048 cause |
| 4 | One recorded night with control 0, shadow 1, tick log on; score with `-session`; compare with the monitor lines | run | the first real verdict |
| 5 | `heal` source from the `lowLifeGate` predicate; `recovery` fact equal to `recoveryDue` (A1) | S | bits 16, 32 |
| 6 | Admission: fix area 73, decide lean under try, exhaustive equality test, `fillEarly` on the function, counter split (section 4) | S | `bot.admission` default; delete four judges |
| 7 | Safety verdict as a fact; survive source reads it and yields to an emergency trip (A2) | M | bit 1 |
| 8 | Player source covers pause, recall, errand, vault keep, repair; save point and checkpoint as upkeep intents (A5) | M | chain lines 39 to 48, 65, 85 can go |
| 9 | Guard asks the arbiter (rule 1), cause by cause, behind one knob | L | the guard list goes |
| 10 | One `PickArea` (zone model or plan) feeding `NeedsArea`; delete `holdTarget`, zone dwell, then flail, ping-pong, border loop after 14 quiet days (A6) | L | cause class A for areas |
| 11 | `supply.Solve` as the only trip judge (town.plan 4 and 5, then rule 4) | L | old trip deciders go |
| 12 | Aura: one `wantAura` with dwell, then `agency.planner` 2 (A7) | S | planner drive |

Steps 2, 3, 5, 6 are independent and offline; 4 needs one night. Nothing after 4 should flip without a control-0 night scored.

## 7. Not read (U)

`internal/survival` controller internals, the town package behind `town.plan` (only the bot seams `bot/routeguard.go:441`, `bot/broke.go:335`, `bot/observe.go:1166` were seen), `internal/outcome` (does not exist in this tree; the typed outcome is `bot/outcome.go`), the invariant monitor, `Fighter.Safety` and the fighter's own drink loop, the sandbox's live knob values (taken from the brief and the audit doc). No test was run; every S above wants an offline test on two builds before it is acted on as fact.
