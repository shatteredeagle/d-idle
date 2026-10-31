# Task layer (internal/task) and how the bot wires it (agent TASKLAYER)

Implements docs/progression/tasklayer.md (design-gaps D12) for Normal Act 1. Code: `internal/task` (+ `internal/task/sim`).
Offline only; every behavioural claim about the live game is **U** (unverified), area ids are **V per repo docs** (levels.txt not re-read).
Tests: `go test -p 1 -count=1 ./internal/task/...` (stdlib only; imports `internal/goals` for `Progress`).

## 1. What exists

| file | content |
|---|---|
| `world.go` | `World` (one read-only value per tick), `Env`, `Actions`, `Area` ids, quest/NPC/boss/item names, `Stage`, `Evidence` |
| `step.go` | `Step{ID,Title,Phase,Need,Ready,Do,Fail,Budget,Est,Constraints,Repeat,Optional}`, `Node{Step,Requires}`, `Plan`, `Planner`, `Session` |
| `kind.go` | `Kind`, `Fault{Kind,Why,Insert}`, `Wrap/Errorf/Classify` |
| `steps.go` | primitives: `StepTravel/Waypoint/Clear/Kill/PickUp/Interact/Talk/GoHome/Chores/Level` |
| `act1.go` | handlers `Handler(quest) []Node` for den, burial (Blood Raven), cain (optional), tools (Smith), tower (Countess), andariel, act1_complete (Warriv); proxy stages |
| `campaign.go` | `NormalAct1(opts)` template and `Campaign.Plan(w, session)` (the planner.md `next`) |
| `executor.go` | `Executor.Tick`: one decision per call, retry/escape policy, hooks, no cursor |
| `progress.go` | `Campaign.Report(w, ex) goals.Progress` |
| `sim/` | fake Act 1 world implementing `Env` and `Actions`; `task_test.go` drives whole campaigns |
| `testdata/plans/act1.txt` | golden node-id lists (`TASK_UPDATE=1 go test -run Golden` after a deliberate change) |

## 2. Model in one page

- **No cursor.** `Tick` reads `World`, asks `Planner.Plan(w, session)` (pure), picks the first node whose `Need` is true and whose `Requires` are satisfied (done or skipped), checks `Ready`, runs `Do` under `Budget`, re-reads the world and checks `Need`. A restart between any two ticks loses nothing (test `TestRestartAtEveryNode` rebuilds the executor before every tick).
- **Quest stage** `None < Started < Cleared < Done`. `Cleared` = objective met, reward unclaimed (Den cleared, Blood Raven dead, Malus in pack, Andariel dead); turn-in steps run on `Cleared`. The stage comes from the quest reader (`World.Quests`, EvMemory/EvSave) or, where the reader knows less, from proxies (`World.Quest`: kills this game, items in pack, areas cleared, act number; EvProxy). A proxy never yields `Done` except where the world itself proves it (Act 2 reached; Countess dead).
- **Evidence.** `Need` returns the tier. With no reader (today) a turn-in step that ran cleanly but whose `Need` stays true on proxy evidence is marked *assumed* for the session (`Executor.Assumed`), not failed; `Report.Verified` is then false. With a reader, a step whose `Need` stays true counts a failure; 3 failures escape.
- **Escape policy** (`Policy`, defaults 3/1/3/2): NotFound/UI count `MaxFails`; Timeout retries once; Death does not count as a failure (Runner recovers) but 3 deaths in one step block the group; Prereq/Resource/Danger faults from `Ready` splice `Fault.Insert` steps in front (depth 2; the first runs in the same tick so a restart cannot lose the splice) else count and block. Optional steps are skipped on escape; other steps **block their quest group**, and the campaign planner routes around blocked groups (planner.md rule 4). When nothing else is runnable `Result{Outcome: OutBlocked, Stuck: true, Attention: "needs_attention: ..."}`.
- **Level gates** live in the campaign (floors: Den 1, Blood Raven 6 (U), Smith 8 (V), Countess 14 (U), Andariel 22 (U, runs.json)); when no unfinished quest qualifies the plan is a `level.N` step (`Repeat`: `Actions.LevelTo` is one bounded slice of the bot's existing reach_level loop). `Ready` repeats the Andariel/Countess/Smith floors and Andariel's potion minimum (inserts a Chores step) as a second line of defence.
- **Constraints** (`NoTownPortal`, `NoLeaveArea`, `NeedsTeleport`, `ExclusiveInput`) are reported through `Executor.Hooks.OnStart/OnEnd`; no Act 1 step needs one yet.
- **Progress.** `Report` fills `Steps` (one per campaign quest, states queued/active/done/failed, plus a `level.N` step while levelling), `Phase` (of the next step), `ETASeconds` (sum of `Est` of needed steps, U), `Source` ("memory" | "mixed" | "events"), `Verified` (all evidence memory/save), and `Current/Target/Percent` = quests done of total.

## 3. Integration plan (exactly what the bot must call)

DONE by INTEGRATE (2026-10-28), see "Integration status" at the end of this file. The text below is the original plan; the code is `internal/bot/task.go` (Env, Actions, `taskStep`), `cmd/d2bot/task.go` (save reader, dialog) and `goals.Campaign`.

### 3.1 goals (owner GOALS, small)
1. Add `goals.Campaign Type = "campaign"` (params: `template` = `"normal_act1"`, `cain` bool) to `allTypes`/validation/catalog. `Params.Template string`, `Params.Cain bool`. Target = number of quests; `Progress` fields already exist.
2. Optionally `Quest Type = "quest"` (`params.quest`) later: `task.Handler(q)` already serves single quests; wrap in a one-entry `Campaign{Entries: [{q,..., minLevel}]}`.

### 3.2 internal/bot (new file `task.go`, plus 3 hooks)
```go
// Deps gets one optional field:
Task *TaskDeps   // nil: campaign goals fail at Start with "task layer not wired"

type TaskDeps struct {
    Dialog   Dialog     // Talk(ctx, npc, pick string) error  (town/dialog, acts.md; does not exist yet)
    Objects  Objects    // Interact(ctx, name string) error   (nav GoToUnitOf(TypeObject) + input.Interact)
    Quests   QuestReader // optional: Stage(quest string) (task.QuestState, bool)  (internal/progress, T2 reader)
}
```
- **`compile`** (runner.go): `case goals.Campaign:` build `task.NormalAct1(task.Act1Options{Cain: p.Cain})`, store it and a fresh `task.NewExecutor(envAdapter, actAdapter)` in `goalSpec` (`spec.camp *task.Campaign`, `spec.ex *task.Executor`); reject unless the character is in Act 1 or `actAllowed`. `spec.label = "Normal Act 1"`. Drop the executor on goal change (`Reset()` on retry of a needs_attention goal).
- **`runGoal`** (goal.go): after the `trip`/`pointsStep`/`gearStep`/`ensureDifficulty` checks and before `r.target(...)`, add
  ```go
  if c.spec.camp != nil { return r.taskStep(ctx, snap, c) }
  ```
  `taskStep` does: `res, err := ex.Tick(ctx, c.spec.camp)` **inside** one `r.do(ctx, c, state, activity, timeout, "task step", f)` so guard, watchdog, death policy and the ladder apply unchanged. Because `Tick` runs a step to completion inside `f`, set `state` per phase before calling (peek with `ex.Peek(plan, w)` or use a small two-phase: `Peek` outside, `do(...Tick...)` inside): `travel`->`StateTraveling`, `fight`->`StateClearing`, `turn-in`/`chore`->`StateResupplying` (or `StateInTown`), `level`->the existing leveling path. **No new Runner states.** Map `res`:
  - `OutDone` -> `finishGoal` via the event path (emit `EvGoalDone` through the existing manager route: send a final `LevelUp`-style progress event or return a new `Runner` callback; simplest: manager polls `Status()` and the Runner calls `goals` done through its event channel, same as `reach_level` completion).
  - `OutDeath` -> return true; `Recovering` already runs; next tick resumes.
  - `OutBlocked && Stuck` -> `r.selfStop`-style pause with `res.Attention` as `Status.Message` (needs_attention); non-stuck `OutBlocked` -> log and continue.
  - `OutInterrupted` -> return false (guard cancelled; the loop re-reads control).
- **Constraints**: `ex.Hooks.OnStart = func(s task.Step){ l.noTP = s.Constraints.NoTownPortal; l.noLeave = s.Constraints.NoLeaveArea }`, `OnEnd` clears them; `handleSafety` ignores `TownPortal` when `l.noTP`, `tripDue`/`errTripDue` is suppressed likewise. (No Act 1 step uses them.)
- **Replace `Quester`/`claimStep`** for campaign goals: `Actions.Talk(npc,"reward")` is the claim; keep the old path for reach_level.
- **Status**: `Runner.Status()` stays as is; the manager copies `camp.Report(world, ex)` into `Goal.Progress` (Steps, Phase, ETASeconds, Source, Verified, Current, Target, Percent) every few seconds from the runner side: add `Runner.Progress() (goals.Progress, bool)` returning the last `Report` (computed at the end of each `taskStep`), and have the manager merge it where it already updates `Progress` (goals/manager.go `refreshProgress`).

### 3.3 Env adapter: `World(ctx) (task.World, error)` from `d2.Snapshot`
| field | source |
|---|---|
| `Act, Area, InTown, Level, Gold, Dead` | `snap.Player` (`Area`, `InTown()`, `Level`, `Act()`, gold stats, death) |
| `Difficulty` | `snap.Player.Difficulty` |
| `Potions` | `r.beltAndInv(snap)` healing count (goal.go) |
| `HasTP` | existing TP check in `town` / pack scan |
| `WPs` | waypoint reader (`internal/d2/waypoint.go`, tracking.md; U) or `nil`: `StepWaypoint` then always runs once (Optional, cheap) |
| `Quests` | `Deps.Task.Quests` when present (EvMemory) and, after Save and Exit, `saves.Progress` (EvSave); **nil map** otherwise |
| `Killed` | from `observeKills` (observe.go): unique/boss kills by name for the current game; cleared on `syncGames` (new game) |
| `Items` | pack scan by name: "Horadric Malus", "Scroll of Inifuss", "Deckard Cain's Scroll"; plus pseudo item "Tristram portal" when a town-portal-class object of Tristram appears in Stony Field |
| `Cleared` | the `loop.cleared` map that `sweep` already keeps (`ErrExplored` => cleared); reset in `syncGames` |
Build `World` once per tick and keep maps fresh copies (the executor re-reads after `Do`).

### 3.4 Actions adapter: each method is a thin wrap, errors tagged with `task.Wrap`
| `Actions` method | calls | error mapping |
|---|---|---|
| `Travel(a)` | `r.travelTo(ctx, int(a))` | `nav.ErrNoPath` -> `KindNotFound`; stuck -> `KindUI` |
| `ToTown()` | `Town.ToTown` | no TP -> `KindResource` |
| `Clear(a)` | `Explore(done=hostile near)` loop + `Fighter.Clear`, as `sweep` does; returns nil once `nav.ErrExplored` (and records `loop.cleared[a]`) | `ErrExplored` is success |
| `KillBoss(name)` | find unit by name/superunique in `snap.Monsters`, `Mover.Explore` until seen, `Fighter.Kill(id)`, `Looter.PickUp` | not seen after full explore -> `KindNotFound`; `safetyInterrupt` stays an interrupt |
| `Talk(npc,pick)` | `Dialog.Talk`; first version may only do the NPC click + Esc and rely on `Quests` to prove the effect | row missing -> `KindUI` |
| `Interact(object)` | `GoToUnitOf(TypeObject, name)` + `input.Interact`; `"cairn-stones"` = brute force over the 5 stones (U) | not found -> `KindNotFound` |
| `PickUp(item)` | `Looter.PickUp` with a name filter | none -> `KindNotFound` |
| `ActivateWaypoint(a)` | `nav.ActivateWaypoint` (V for Cold Plains) | |
| `Chores()` | `Town.Chores(r.fullChores("task"))` | broke -> `KindResource` |
| `LevelTo(n)` | one pass of the existing reach_level step (`target`, `travel`, `sweep`) with `spec.level = n` | |
Deaths: wrap `errDied`/`errLeftGame` with `task.Wrap(task.KindDeath, ...)`; guard causes (`errStuck` ...) already cancel `ctx`, which the executor reports as `OutInterrupted` (not counted).

### 3.5 cmd/d2bot
1. Build the adapters in `adapters.go` (`taskEnv`, `taskActions` over the Runner's Deps) and pass `Deps.Task`.
2. `sim.go`/`dry.go`: wire `internal/task/sim.Game` through the same `TaskDeps` so `d2bot --sim` can run a campaign goal end to end.
3. Web/API: allow `POST /api/goals {"type":"campaign","params":{"template":"normal_act1"}}`; the dashboard card already draws `Progress.Steps` and `Verified` (tracking.md).

### 3.6 Order and acceptance
1. goals type + sim wiring (campaign goal completes in `d2bot --sim`): acceptance = a `runner_test` rig with `task/sim.Game` goes clvl 1 -> Act 2.
2. Env + Actions adapters over fakes in `bot` tests; `KillBoss`, `Talk` stubs return `KindFatal`-free errors so a missing capability blocks the quest with needs_attention instead of hanging.
3. Live, one handler at a time, on a cheated throwaway (verification.md): Den (`Clear` V pieces exist), Blood Raven, then Smith/Countess/Andariel, each with the checklist: run `taskStep` from a prepared save, compare the quest word before/after.

## 4. Live checklist (not run; needs the game)
- L1 `Talk(Akara,"quest")` and `"reward"`: row index per NPC (V-04/V-05), effect proven by quest word 0x14F block or HUD flag 0x11 edge. Until then `Talk` effects are U.
- L2 Den cleared signal: does `sweep`'s `ErrExplored` match "all monsters dead" (quest bit)? Compare with the quest word.
- L3 Blood Raven location: superunique name in the monster list in Burial Grounds (17)? `Fighter.Kill` kiting.
- L4 Smith: Malus drop without talking to Charsi first? (handler talks first, Optional). Imbue item chooser (`"imbue:best"` is a placeholder pick).
- L5 Andariel: poison, potion floor 4, antidotes; Warriv option text `"go east"` and act change read (`Player.Act()`).
- L6 Tristram chain (Cain): stone order, portal object proxy `"Tristram portal"`.
- L7 `World.Killed` fed from `observeKills` by name: confirm "The Countess"/"The Smith"/"Blood Raven"/"Andariel" spellings in `data` monster names.

## 5. Interface decisions others must know
- `internal/task` imports only `goals` + stdlib; nothing in `bot` imports it yet (no cycle risk: `task` never imports `bot`).
- Errors from adapters must be tagged (`task.Wrap`/`Errorf`) or they classify as `KindUI` (3 tries then skip). Context deadline = timeout, context cancel from the guard = interrupted.
- `World.Quests == nil` means "no reader"; do not fill it with zero values.
- Dead ends / not done: Acts 2+ handlers, cube/merc/difficulty primitives, `Dialog` (does not exist), real `Quests` reader, bot wiring, golden plans beyond node ids, ETA numbers are rough (U).

## 6. Integration status (INTEGRATE, 2026-10-28)

Done: `goals.Campaign` (`campaign[:TEMPLATE]` in `--goals`, `POST /api/goals {"type":"campaign"}`), `goals.ProgressReporter` (the manager merges `Runner.Progress()` every tick and ends the goal when `Current >= Target`),
`bot.TaskDeps{Dialog, Objects, Waypoint, Quests}`, `taskStep` inside one `r.do` (state from the next step's phase), `taskEnv`/`taskActs`, a quest `Reader` over the save, a `Dialog` over town `openNPC` + `PickLine`.
Offline tests: `internal/bot/task_test.go` (Den sequence, no redo after a restart, leftovers, refusal without TaskDeps), `cmd/d2bot/task_test.go` (save mapping), `internal/goals/campaign_test.go`.

Decisions and findings the plan did not have:
- **Save evidence lags.** The .d2s only changes on Save and Exit, so a save-fed reader cannot confirm a step in the session that ran it. `World.SaveOnly` (set by the bot unless `TaskDeps.QuestsLive`) makes `evOf` report proxy evidence: a clean Do whose Need stays true is *assumed* (not 3 failures and a blocked quest). The cmd reader reports only what the save shows started/done (prior knowledge); an untouched quest is left to the proxies. This is what keeps Den of Evil from being redone after a restart (the save says Done).
- **Save and Exit before the game stops.** `d2bot --stop-game` now walks home (`Runner.Settled`), then runs the Cycler's `LeaveGame` so the file carries the quest words and waypoints.
- **`ErrExplored` is not "every monster is dead" (LIVE, i4).** The Den was declared cleared with 11-19 living monsters in the table because the fighter sets targets aside. `taskActs.Clear` therefore counts living hostiles after the exploration ends and walks to the nearest leftover (up to 25 times) before it reports success.
- **A level-up event must not set `Current` of a campaign** (it ended a 6-quest campaign at level 6): `goals.Handle` only does that for `reach_level`.
- **Dialog (live-checked result in section 7).** Akara is Talk, then the quest row ("Den of Evil"); the old `questAdapter` only clicked Talk. The effect is verified by the unspent skill point.
- Sentinel errors from actions (`ErrPortalOpened`, `ErrLeftGame`) are re-raised from the `r.do` closure so the usual trip/ladder logic runs: an opened portal travels as `KindFatal` (an interrupt, no failure counted), a chicken exit and a death as `KindDeath` (three in one step block the quest: a boss that is too strong must not loop through fresh games). A step that fails (retry or blocked) also returns its error to `r.do`, so the bot's own failure ladder (close panels, town, a fresh game) runs when the executor alone cannot get the character out of a place (LIVE: stuck in the Den). A blocked quest never spins the loop.
- Not done: `d2bot --sim` campaigns (the sim runner is `goals/sim`, not the bot); Acts 2+; `Objects` (Cain chain), Warriv/Kashya/Charsi rows are U.

## Quest state across a game restart (PLANRESTART)

What a restart of the game keeps and loses, as the planner sees it (V: `TestCampaignGameRestartAtEveryTick`, restart after every action, stride 1, plus `TestCampaignRestartAfterStaffPlaced`):

- Kept: real pack items (Cube, Staff pieces, scrolls), quest words (the reader), waypoints, level.
- Lost: `Killed`, `Cleared`, and the adapter's pseudo items (`Horadric Staff placed`, `Book of Skill read`, `Orb smashed`, `Tristram portal`, Anya/forge items).
- Rule: a Need/Ready predicate that gates a later step on a pseudo item must also accept the persisted quest word. Bug found: `staffPlaced` read only the pseudo item, so after a restart past the orifice the Duriel door step was "not ready", no entry had work left and `Campaign.Plan` returned Done in Act 2. It now also accepts `Quest(QStaff) >= Done`.
- `Campaign.Plan` returns Done when no entry has a Need; a Ready gate that goes false by losing state therefore reads as "finished". Gate on something that survives a restart.
- U: with no quest reader (reader=false) the placed-staff state cannot survive a restart; the restart test runs with the reader only.
