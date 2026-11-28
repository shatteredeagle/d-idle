# The intent arbiter (shadow mode)

> status: amended 2026-11-27 by card fix7 (TASK-1096): survive reads the fighter's safety verdict (fact `safety`), yields to an emergency trip; controlled hooks call `Report`; recovery and the gold farm pass the low-life gate; a declined hook does not hold the quest step. Details and migration order: `docs/design/arbiter-target-and-migration.md`. Amended 2026-11-28 by card fix7b (TASK-1115): back-off per (source, kind, cause) and never for a running or emergency trip; survive needs a drink verdict with a hostile near; the watchdog heartbeat is `observer.beat` (same doc, "What fix7b changed").

> status: current. First slice of roadmap R4 (docs/design/fable-review-2026-11-19/06-roadmap.md, 05-architecture-and-overhaul-options.md section 3.1). SHADOW ONLY: nothing the hero does changes. Tags: V = a test or measurement in this branch, U = unverified.

## Why

`runGoal` (internal/bot/goal.go) is a fixed list of about twenty `if step() { return true }` hooks; each decides alone, and five guards (flail, ping-pong, explore gave up, border loop, arrival bounce) detect afterwards that two hooks undid each other. The arbiter is the place that knows what the hero is committed to and what a change of plan costs. It is built and tested here, run beside the chain, and moved over one hook family at a time (migration plan below).

## Model (internal/arbiter)

- A `Source` proposes zero or one `Intent` per tick: priority class (reflex > player > upkeep > objective > filler), urgency (compared inside a class), where it must happen (`town`, an area name, or `*`), estimated cost, reason, resumable, steps.
- `Arbiter.Decide(View)` picks one. Rules: strict classes; ties go to the running intent, then lower cost, then name. The chosen intent holds a lease of `bot.arbiterDwellSec` virtual seconds; only a higher class preempts it. An intent that fails (`Report(..., false)`) backs off 30 s, 60 s, 120 s ... up to 30 min, per (source, kind, `Intent.Cause`); an intent with `NoBackoff` (a running town trip, emergency chores) never does. An area change that would undo the previous one inside `bot.arbiterTravelSec`, with no progress in between (kill, XP, gold, belt), is refused and logged with both reasons (the intent's own and the budget). Reflexes and player orders are never refused. When an upkeep intent for town wins, every source that implements `TownVisitor` is asked what it wants in the same visit (`Decision.Batch`).
- Sources read only `View`: world facts and the arbiter's previous decision (`View.Prev`, its own trace). They never read each other's state.
- Pure and deterministic: no goroutines, no clock, no I/O. Time and progress come in through `View`.
- `Shadow` (internal/arbiter/shadow.go) wraps it for a live loop: recovers any panic (counted), one short mutex, no I/O, compares the arbiter's hook family with the family the chain acted for.
- `StdSources` (sources.go) are the first nine sources (survive, player, trip, claim, resupply, recovery, goldfarm, quest, level). Build data (belt line, flee line, gold need) arrives as facts, so no class, skill or build literal lives in the package.

## Temporary intents and the lost-control hook

- `Intent.TTL` (virtual seconds) and `Intent.Until` (a condition over the facts in outcome-rule syntax, `potions>=potions_min`) make an intent TEMPORARY. `Arbiter.Post` adds one; it is proposed beside the sources' own until the TTL ends, `Until` holds, or the executor reports it done (`Report(..., true)`). V: `TestRefusalBecomesTemporaryIntent`, `TestTemporaryIntentEndsOnReportDone`.
- A refusal with a wake fact becomes one: `Arbiter.Refused(need, wake, now)` looks the need up in `RefusalIntents` (data: `needs_potions` becomes an upkeep refill, `needs_gold_farm` a recovery farm, `too_weak` a filler level-up; `route_closed`, `area_banned` and `not_ready` stay a pause only). The bot calls it from `handleRefused` (internal/bot/outcome.go, typed `Refused{Need, Wake}`), so a source reads the refusal and does not re-derive the facts (V `TestRefusalBecomesArbiterTempIntent`, two needs). Only with `bot.arbiterShadow` 1.
- `LostControl` (optional interface on a Source): `Lost(in, by, why, starved)` is called once when the intent the source proposed loses to another, and once more when it has been outranked for `StarveSec` (600 s). The same event is `Decision.Starved` and, in the bot, the soft invariant `arbiter-starved` (docs/invariants.md). V `TestLostControlAndStarvation`: a source that stopped asking is not starved.

## Control path (dark, `[dark-launch]`)

Knobs `bot.arbiterControl` (0, off) and `bot.arbiterFamilies` (a bit list, 3 = survive and player; bit order in `arbiter.Families`), needs `bot.arbiterShadow` 1. With control on, `runGoalControlled` (internal/bot/arbiter_control.go) takes the arbiter's decision BEFORE the chain. If the chosen family is under control its hook runs and the chain's lower hooks do not start: survive fights where the hero stands (no chore, claim or quest step starts under the drink line with hostiles near; the guard's reflex drink does the drinking), player runs the item actions first. If the arbiter chose nothing, a source panicked, or the hook did nothing, the old chain runs unchanged (the fallback). Minute line attribute `arbctl=N` counts the ticks the arbiter acted. Soft invariant `arbiter-starved` (outranked 600 s or more). V: `TestArbiterControlSurvive` (two classes, plus knob off and bit not set must not act). Chores and claim (card arbfam, entry 1): bit 4 runs a trip in progress or starts one when the chores are due (`choresStep`), bit 8 runs the claim trip (`claimTripStep`); both are the chain's own hooks, extracted so the arbiter calls them directly; when the hook finds nothing due the chain runs. V: `TestArbiterControlChores`, `TestArbiterControlClaim` (two classes, knob off and bit not set must not act); the outcome verdicts `chores` (4 cases, soft 3 vs 0) and `claim` (1 case, soft 1 vs 0) pass in `d2scenario arbiter-replay` (`TestChoresClaimOutcomes`, two refill lines, must-fail with nothing wanted; the stranded-walk and claim cases are MODEL cases, facts U). Goldfarm and recovery (entry 2, bits 16 and 32): `goldFarmStep` for the player's Farm gold order and `recoveryStep` through `recoveryDue` (the chain's own test, extracted), both campaign goals only, the order keeps its place above recovery (urgency 0.9 against 0.8, as the chain orders them); the resupply source stays quiet while recovery is on and the refill price is unknown (recovery means the purse cannot pay). V: `TestArbiterControlGoldFarmAndRecovery` (two classes, bit not set must not act), `TestGoldFarmRecoveryOutcomes` (two refill lines, must-fail without the recovery fact); verdicts `recovery` (2 cases, soft 2 vs 0) and `goldfarm` (2 cases, soft 1 vs 0) pass; the poverty spiral, the quest walk while broke and the unpayable trip under the order are MODEL cases, facts U. Quest (entry 3, bit 64): the quest source reads the typed `Refused` through `arbRefused` (need and the wake facts it was logged with; fact `quest_refused` while those facts are unchanged) and stays quiet, so a refused step is not asked again; `ObjectiveArea` is wired from the planner's step area (U); the quest hook defers (`arbHoldsQuest`): when the arbiter chose another family whose own hook found nothing to do, the quest step does not start and the hero runs a leveling slice instead. V: `TestArbiterControlQuestHeldWhileRefillIsWanted` (two classes; bit not set and knob off must not act), `TestQuestSourceReadsTheTypedRefusal` (a level change is a wake), `TestQuestOutcomes` (two refill lines, must-fail without the refusal fact); verdict `quest` (2 cases, soft 2 vs 0) passes, MODEL cases. Level (entry 4, bit 128): the same hook (`arbHoldsQuest`) when the arbiter chose the filler, which happens when the quest source is quiet behind a standing refusal or a `too_weak` refusal posted a leveling intent: the hero levels instead of asking the refused step again. V: `TestArbiterControlLevel` (two classes, bit not set must not act), `TestLevelOutcomes` (hard row xp-frozen, chain 1 vs arbiter 0, two refill lines, must-fail without the refusal fact); verdict `level` passes (MODEL case, facts U: the log never says how long the hero stood idle). The flail-guard 7-day rule of the migration table needs a sandbox run (U). Enable rule per family: ledger row `bot.arbiterControl` in data/darklaunch.json (verdict passes, incident rows green on a lockstep run, one sandbox flip, 24 h watch with auto-revert, owner go). U: nothing here ran against the game.

## Shadow knobs (docs/knobs.md)

| knob | default | meaning |
|---|---|---|
| `bot.arbiterShadow` | 0 | 1 runs the arbiter beside the chain. 0: nothing runs, nothing is logged |
| `bot.arbiterDwellSec` | 120 | lease length |
| `bot.arbiterTravelSec` | 600 | travel budget window |
| `bot.arbiterLogGapSec` | 60 | minimum gap between two divergence lines of one (chain, arbiter) pair |

To enable on the sandbox: set `bot.arbiterShadow` to 1 in the sandbox's tuning (the same place any `bot.*` knob is set), restart not needed beyond the usual knob reload. To disable: set it back to 0.

## The adapter (internal/bot/shadow.go)

`runGoalShadowed` replaces the one call of `runGoal` in `loop.go`. With the knob 0 it IS `runGoal`. With it on it collects world facts from the snapshot and loop state BEFORE the chain acts (in town, life, hostiles near, a trip running, claim pending, belt potions, gold, recovery, gold farm order, camp goal, player item actions pending), runs `runGoal` unchanged, then asks the shadow. The chain's family is read from the whyline code of the activity the chain started (`chainFamily`). The hooks are not wrapped one by one yet: their predicates have side effects (`needChores` clears `forceTrip`), so the sources use pure facts instead (U: how closely they match each hook's own predicate is exactly what the shadow log measures).

Read-only data: `Runner.Arbiter()` returns the last `Decision` (for the watch API; not wired into a page in this slice).

## How to read the shadow log

- Divergence line (rate limited): `arbiter shadow diverge="chain chose quest, arbiter would choose resupply/refill (belt 1 under 4) because quest/step: class objective below upkeep; ..."`. Format: chain family, then the arbiter's intent as `source/kind (reason)` with ` [lease]` when held, then the first two reasons it rejected something.
- Minute line attribute `arb="chores=68/103 quest=258/310 panics=0"`: per chain hook family `diverged/ticks` since the previous minute line, then the panic count. Families: chores, claim, quest, goldfarm, recovery, player, survive, level. Ticks where the chain family is unknown (no whyline code) are not counted.
- A high divergence rate for a family means either the arbiter source is wrong (fix the source) or the chain's hook is (the reason is in the line). Never both unread.

## Replay (offline)

`go run ./cmd/d2arbiter [-follow] [-trace] LOG.jsonl` feeds a JSON-lines log (`minute` lines, or `tick` lines: area, kills, xp, chain, objectiveArea, facts) to the arbiter. Without `-follow` the arbiter sees the logged hero (divergences and counters). With `-follow` the simulated hero obeys the arbiter, facts and progress still come from the log, and an intent that ran its estimated cost in place with its reason still holding is reported failed.

Recorded sessions: the bot writes these `tick` lines itself, at most one a second (always on since 2026-11-28, card grad1; it was the knob `bot.tickLogSec`); a log with tick lines replays from them only and its minute lines are skipped (docs/design/ticklog-t4.md, card ticklog).

Fixture `internal/arbiter/testdata/darkwood_pingpong.jsonl`: the Dark Wood, camp, Dark Wood, Cold Plains cycle of 2026-11-18 over 12 minutes. V: the area sequence and gaps are the real ones copied from the sandbox log into `internal/bot/flail_test.go`. U: facts (belt 1 under 4, 350 pack gold, a quest step in Dark Wood, no progress) are assumed from docs/probes/treefix3-2026-11-19.md, not logged. Result (V, `TestReplayPingPong`): the chain made 412 area changes; the arbiter, obeying itself, makes 2 (one to town, one back after the back-off), refusing 359 undo trips, each with both reasons in the trace (`testdata/pingpong_replay.golden`). Limit: the log never says whether the town visit helped; a chore that stays wanted is treated as failed after its cost.

## Measured (V, this branch)

- `go test -race -count=20 ./internal/arbiter` passes; the prototype scenarios are all here: first-match chain 180 area changes in 60 virtual minutes vs arbiter 9, broke with stash gold, preemption and resume, starvation and back-off, golden decision trace.
- `BenchmarkShadowTick`: about 2.6 microseconds and 19 allocations per tick with the standard sources, against a goal tick of about a second.
- Shadow on, off: `TestShadowOffByDefault`, `TestShadowOnLogsAndKeepsTheChain`; panics: `TestShadowSurvivesPanicAndKeepsCounting`.

## Not done (U)

- Live: no sandbox run yet. Whether the divergence rates are sane on a real log is the first thing to read.
- `ObjectiveArea` is empty in the adapter (where the quest step must be done is not available as a pure fact), so the travel budget bites only for the trip and replay cases live. Wiring it is the first adapter follow-up.
- The chain's family is the whyline code and can lag one activity (U).
- No executor: the arbiter chooses, it does not run or abort steps; `EstCost` is a tie-break and trace field.

## Exit rule (amended 2026-11-25, user decision)

Agreement with the old chain is NOT the bar: the live divergences are mostly cases where the arbiter is right and the chain is wrong (Fable logic review 2026-11-25, section 7.3), so the better the arbiter, the further it is from passing the percent rules below. A family moves behind its knob on ONE sandbox when, judged by outcome:

- on recorded sessions replayed with `d2arbiter -follow` and on the incident scenarios (docs/probes, ADR 0020 amendment 2026-11-25), the arbiter-led run has fewer invariant-monitor hits than the chain-led run and no new kind of hit;
- every remaining divergence line is explained (arbiter right, chain right, or fact missing), and the "earliest" dates below still hold.

The "divergence under N percent" figures in the table are kept as a reading aid, no longer as the exit condition.

## Outcome harness (offline, report-only)

Questy 2026-11-26 cases (`tools/provingground/arbiter-outcomes/questy-2026-11-26.json`, `TestQuesty1126Outcomes`): death 15 (chain quest step against a hard potion-not-drunk, the arbiter picks survive), the poverty sell (broke hero in the camp with `sellable` items: the resupply source now plans the sell first, family chores), the dry area (recovery with the `new_game` step when fact `area_dry` is set). New facts `sellable` and `area_dry`; soft rows `no-earn`, `guard-stall`, `fight-stall`, `stuck-in-pack` listed in `cmd/d2scenario/invariants.json`. MODEL cases, facts U.

`go run ./cmd/d2scenario arbiter-replay [-strict] tools/provingground/arbiter-outcomes/*.json` decides each case with the standard sources and scores BOTH the recorded chain family and the arbiter's family with the same data rules (a rule says: this family, under these facts, hits that invariant row). Hard rows decide, soft rows break ties; a hit the chain did not have is a new kind and fails the family. The rules are a model of the monitor (U until a lockstep run confirms a row); the three Questy incident cases (V, `TestQuestyOutcomes`, two drink lines in `TestSurviveOutcomeTwoBuildsAndMustFail`): survive, chain 1 hard hit (potion-not-drunk) vs arbiter 0; chores, chain 2 soft hits (field-entry-no-kit) vs arbiter 0; the ping-pong first tick scores no hit for either. Limit: one decision point per case, not a whole session; whole sessions go through `d2arbiter -follow`.

## Outcome over a recorded session (stage 4)

The case files score one decision point each; a night of the bot's tick lines (docs/design/ticklog-t4.md) is scored tick by tick with the same rules:

`go run ./cmd/d2scenario arbiter-replay [-strict] [-follow] -session LOG.jsonl tools/provingground/arbiter-outcomes/*.json`

- The log is replayed through the shadow (`arbiter.Replay`, so lease, back-off and travel budget apply); each tick gives the chain's logged family and the arbiter's family on the same logged facts (`ReplayResult.Points`). `-follow` lets the simulated hero obey the arbiter as `d2arbiter -follow` does; the rules read facts only, so it changes the verdict only through the arbiter's own decisions.
- The rules of all given files are merged by name (`arbiter.MergeRules`); the files' cases are not scored in this mode.
- A hit counts once per EPISODE: on the tick it starts for that side, not again while it holds (a model of the monitor's once-per-change line, U). A tick counts for the arbiter's family when the two differ and the arbiter chose something, else for the chain's; ticks where neither chose anything are skipped. A new kind is an arbiter hit whose kind the chain never hit in the whole session.
- Verdicts as for cases (`arbiter.Judge`). A family with no hit on either side is printed as "no evidence" and does not fail `-strict`.
- V: `TestSessionOutcomeTwoBuilds` (drink lines 50 and 70, refill lines 2 and 6: two stretches under the line are two hard episodes for the chain, 0 for the arbiter), `TestSessionOutcomeMustFailWithoutDrinkLine` (no drink line: no survive verdict, quest does not pass), `TestScoreSessionNewKindAndEpisodes`, `TestArbiterReplaySession`. U: no real night scored yet.

Reading a Doofer night (arbiterControl 1, families 255): with control on, the tick line's `chain` is the family of the hook that ran, which is the arbiter's whenever it acted (U), so a control-on night mostly measures agreement; score a night with `bot.arbiterControl` 0 and `bot.arbiterShadow` 1 (the chain led) and compare the hit counts per family with the night's real monitor lines.

## Migration plan per family

Order is by risk. Each family moves from the chain to the arbiter behind its own knob only when its exit rule is met; until then the chain's hook stays. The flail guard, ping-pong and border-loop breakers stay until the shadow log shows they never fire for 14 days after the last family moved. Under the project freeze only shadow and scenario-backed work merges.

| order | family | exit rule by outcome (all of) | earliest |
|---|---|---|---|
| 1 | survive, player | `d2scenario arbiter-replay` verdict `survive` passes (fewer hard hits, no new kind) on the incident case of 0251 and on every recorded survive divergence added since; `d2arbiter -follow` on recorded sessions shows no new hit kind; one sandbox, 24 h watch | 2026-11-26 |
| 2 | chores, claim | verdict `chores` passes on the 0319 and 0056 cases and the ping-pong replay (`TestReplayPingPong`) shows no area change the chain did not make; one sandbox, 24 h watch | 2026-12-03 |
| 3 | goldfarm, recovery | a case for the stash-gold broke hero exists and passes; verdict `goldfarm` passes; one sandbox, 24 h watch | 2026-12-10 |
| 4 | quest (board and task) | `ObjectiveArea` wired and the quest source reads typed `Refused{need, wake}`; verdict `quest` passes; no `refusal-storm` in a 7-day sandbox run | 2026-12-17 |
| 5 | level (filler) | last: everything else moved; a 7-day sandbox run with the arbiter acting shows no flail guard hit | 2026-12-24 |

Each row also needs the scenario rows of its incident (docs/probes/incident-scenarios.md) green on a lockstep run (U: none run yet) and the owner's go. The cases are data (`tools/provingground/arbiter-outcomes`); a verdict needs at least one case, a family with none has no verdict and does not move.

If a date passes without the rule met, the family stays on the chain and the reason goes into this table; the dates are earliest, not promises.

## Upkeep source wiring (card townwire)
- With `bot.arbiterShadow` 1, `bot.arbiterControl` 1 and bit 4 (chores) set in `bot.arbiterFamilies`, the shadow runs `arbiter.UpkeepSources()` (the trip and resupply sources collapsed into `upkeep`) and every View carries the supply plan's facts (`arbiter.PlanFacts`, profile `field`, from the town's `SupplyPlan`). With the defaults (control 0, families 3) the sources and the facts are exactly as before (V `TestUpkeepWiredBehindChoresBit`, a Sorceress and a Paladin).
- Not done: the old trip deciders (`NeedsTrip` reasons, `trip.go`, the resupply knobs) still run in the chain and in `arbChores`; collapsing them is a follow-up after a live run of the controlled chores family. U: not run live.
- `GET /api/townplan` is mounted in both backends (cmd/d2bot `bindTownPlan`); it answers 503 until the route is bound and the "no plan yet" view until a visit solved one (knob `town.plan` >= 1; the Town keeps the last plan and the route stops, `Town.PlanView`). The Town plan card stays opt-in (manifest `optIn`): a layout names it to show it; it is not in the default layout.
