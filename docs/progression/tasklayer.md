# Task layer: step graph with preconditions

Implements design-gaps D12. Constraint: `bot.Runner` stays small (V bot.md: 8 states, per-state timeouts, guard, watchdog ladder, death policy). Quests, act
hops, bosses, merc and difficulty all become **Steps** executed inside one Runner state; the Runner never learns the word "Duriel".
Goal types and campaigns come from `docs/goals` (the planner emits them); this layer executes what a goal expands to.

## 1. Where it sits

```
goals.Manager -> goal (quest|boss|reach_level|campaign child|...)
  bot.Runner.compile(goal) -> Plan            // the only place a goal becomes work (bot.md "Add a goal type")
    Plan = []Node with edges (requires)       // a small DAG, executed in topological order
  bot.Runner state = Working (or Clearing)     // existing states; the plan is the "what", states are the "where"
    r.do(step.Run) under guard + watchdog + death policy + ladder (unchanged)
```
Runner states do not grow: `Idle, InTown, Resupplying, Traveling, Clearing, Recovering, Paused, Error` stay (bot.md). A step *uses* them: `Travel` steps run in
Traveling, fights in Clearing, chores in Resupplying. The step executor only decides what to do next when the Runner asks "what now?" at a loop iteration.

## 2. Types (extends D12; names are proposals)

```go
type Kind int // fault classification, replaces ad-hoc sentinel matching (finding F14)
const (KindNone Kind = iota; KindPrereq; KindResource; KindNotFound; KindDanger; KindUI; KindTimeout; KindDeath; KindFatal)

type Evidence int // how sure a decision is (quests.md)
const (EvNone Evidence = iota; EvProxy; EvMemory; EvSave)

type Step interface {
    ID() string                                     // stable: "quest.radament/kill", "hop.a2-a3/talk-meshif"
    Need(w WorldView) (needed bool, ev Evidence)    // false = already done; never act on EvProxy alone to SKIP
    Ready(w WorldView) *Fault                       // preconditions: act unlocked, clvl, gear, potions, WP, quest state
    Run(ctx context.Context, d Deps) error          // idempotent; ends with a Checkpoint where possible
    Budget() time.Duration                          // per-step watchdog (hard cap; the Runner's watchdog is the progress timer)
    Constraints() Constraints                       // NoTownPortal, NoLeaveArea, NeedsTeleport, ExclusiveInput
}

type Fault struct{ Kind Kind; Why string; Retry bool; Insert []Step } // Insert: prerequisite steps the planner can splice in

type Node struct{ Step Step; Requires []string /* step IDs or predicate names */; Optional bool }
type Plan struct{ ID string; Nodes []Node; Created time.Time } // data, no goroutines; built by pure functions
```
`WorldView` is the read-only union of `d2.Snapshot` (V), `Observation` (tracking.md), progress from `saves.Progress` (T1) and inventory. One value, one tick
(finding F7): the executor hands the same `WorldView` to `Need/Ready` of every candidate in a loop iteration.

`Deps` carries the existing consumer-side contracts (bot/contracts.go): Mover (nav), Fighter, Townsman, Looter, Cycler, Input, Clock, Emit, plus the new
`Dialog` (acts.md), `Quests` (progress reader), `Merc`, `Difficulty` (launch). A step may only call what `Deps` gives it, which is how fakes work.

## 3. Execution

```
loop (inside Runner.Working, one decision per iteration, like bot.md "run loop"):
  w := world()
  pick := first Node in topological order where Need(w).needed && !Skipped && Requires satisfied
  if none: plan done -> report (goal progress comes from the manager, events unchanged)
  if f := pick.Ready(w); f != nil:
        switch f.Kind:
          KindPrereq/Resource: if f.Insert != nil -> splice them before pick (planner-approved depth <= 2), continue
          else mark blocked, emit needs_attention, pause plan (not the goal) after N
          KindDanger: ask planner for a level step; do not Run
          KindNotFound: skip if Optional, else escalate
  err := r.do(pick.Run, Budget)         // existing: guard cancels on death/leave/stuck/TP-needed
  classify err -> Kind; on KindDeath: Runner.Recovering handles; the plan resumes at the same step because Need is re-derived
  on success: Run's last Checkpoint should have made Need false; if Need is still true with EvMemory/EvSave, count a failure (3 -> skip + needs_attention)
```
Properties that make it testable and crash-safe:
- **No saved cursor.** Which step is next is a pure function of the world (quest bits, inventory, area, difficulty). Death, restart, ladder rung 3 (new game),
  and a restart of the whole bot all resume correctly. This is the reason `Need` exists.
- **Idempotent steps.** `FetchObject`: "item already in pack => skip". `Talk`: reading a changed menu (greeting) is handled in `Dialog`.
- **Budget** per step on top of the Runner's no-progress watchdog; exceeding it = `KindTimeout` (retry once, then escalate).
- **Constraints** are checked by the Runner when it starts the step: `NoTownPortal` suppresses `Fighter.Safety`'s TownPortal action and the "town trip due"
  guard cause for the duration (Ancients), `NoLeaveArea` makes the guard abort instead of walking home, `ExclusiveInput` is the future input lease (F6).
- **Evidence**: `Need` returns the tier. The executor logs it with the step event, the dashboard shows "verified/estimated" (tracking.md `Verified`).

## 4. Primitive steps (shared library, one file each)

From execution.md section 1, now with ownership. All take a spec struct and return a Step.

| primitive | spec | built on | status of the building blocks |
|---|---|---|---|
| `Travel{Area}` | area id | `Navigator.ToArea` + hop table (acts.md section 5) | ToArea V (nav), hops U |
| `Clear{Area, Until}` / `KillUnique{Class}` | | `Explore` + `Fighter.Clear/Kill` + `Looter.PickUp` | V pieces, boss scripts new |
| `Talk{NPC, Pick}` | npc class, Choice, expected effect | `Dialog` (acts.md) | open V, pick U |
| `FetchObject{Preset, Interact}` | | `GoToUnitOf(Object)`, `Interact`, verify by pack/object mode | object interact V (waypoint), chests U |
| `UseItem{Class}` | | right click in pack | TP scroll V path, books U |
| `Cube{Recipe}` | | stash/cube UI | new |
| `Equip{Item,Slot}` | | `hand.Transfer` | pack->slot U |
| `Checkpoint{Pred}` | quest bit / WP bit / inventory | `Quests`/`WorldView` | reader U |
| `Chore{Plan}` | | `town.Chores` | V mostly |
| `HireMerc`/`ReviveMerc`/`EquipMerc` | | merc.md | U |
| `EnterDifficulty{D}` | | difficulty.md | U |
| `ActivateWP{}` | | `ActivateWaypoint` | V (Cold Plains, V nav2) |

Quest handlers are **data plus a few lines**, not state machines: a handler returns `[]Node`. Example: Radament (execution.md, quests.md index 9):

```
quest.radament:
  talk-atma        Need: !questStarted(9)          Talk{Atma, "quest"}            // start, flag 0x11 edge checked
  to-sewers        Need: !killed(Radament) && !done(9) Travel{Sewers3(49)}          Ready: act==2, clvl>=?, potions
  kill             KillUnique{Radament}           Constraints{}                   Budget 6 min
  take-chest       FetchObject{HoradricScrollChest, interact}                       // Horadric Scroll
  take-book        FetchObject{ground "Book of Skill"}  (Looter always takes quest/skill books)
  use-book         UseItem{BookOfSkill}           Need: bookInPack                // stat 5 +1 checks
  turn-in          Talk{Atma, "reward"}           Need: !done(9) (EvSave|EvMemory)
  spend-point      -> existing Leveler build table (quest-points.json policy)
```
Edges: `kill` requires `to-sewers`, `use-book` requires `take-book`, etc.; all other nodes are ordered by list position unless they declare `Requires`.

## 5. Graph building (planner side)

- `quest.X` expands via a Go function `Handler(q QuestID, diff Difficulty) []Node` (table of handlers); the planner (planner.md) composes handlers across
  quests with the `requires` JSON as the *outer* graph. The two graphs are the same shape at two levels: quest-level requires (planner.md machine form) and
  step-level requires (inside a handler). One `Requires []string` mechanism serves both, so `quest.duriel` lists `quest.horadric_staff` as a quest
  dependency and the executor treats it as "all nodes of that handler Done".
- A `Plan` for `campaign:normal_playthrough` is NOT built at once: the planner appends the *next* quest-handler plan when the current finishes (planner.md section 4
  "next(o, h)"). That keeps plans small, and re-plans are cheap because there is no state.
- Plans are logged (`plan_built` event with node ids) so the dashboard can draw sub-steps (`Progress.Steps`, tracking.md): the node's ID/Title map to `Step{ID,Title,State}`.

## 6. Failure classes (Kind) and what each does

| Kind | examples | action |
|---|---|---|
| KindPrereq | act locked, clvl below floor, quest bit missing, no TP | splice prerequisite steps or block with needs_attention |
| KindResource | no potions, merc dead, broke, pack full | `Chore` step first (already in Resupplying), then retry; broke => gold farm goal |
| KindNotFound | NPC not in unit table, preset missing, boss not found | rescan once, walk to the Kolbot-style fallback spot, then skip Optional / escalate |
| KindDanger | unsafe affix, resist < 0 after penalty | planner inserts level/gear; never bypass |
| KindUI | dialog row missing, panel stuck | close panels (ladder rung 1), re-approach, 3 tries then skip |
| KindTimeout | budget over | retry once; then skip/escalate |
| KindDeath | died during the step | Runner's Recovering; resume by `Need` |
| KindFatal | game gone | Runner fatal (unchanged) |

(`Kind` replaces the per-package sentinel matching in finding F14 only for steps; existing errors map via one function `Classify(err) Kind`.)

## 7. Tests (the point of the design)

- `Need/Ready` are pure: table tests over fake `WorldView` (inventory, quest bits, area, clvl).
- A `stepworld` fake (like `rig`/`world` in bot's runner_test.go, V bot.md) simulates NPCs, talk effects, bit changes and an inventory; per handler one test goes from
  a fresh state to Done, one from "crash at every node" (kill the executor, rebuild, assert same result), one for "bit already set at start => zero actions".
- Constraint tests: Ancients plan never emits TownPortal; NoLeaveArea aborts on a walk home.
- Golden plan tests: `Handler(q, d)` node-id lists are stored under `testdata/plans/`; a diff means a deliberate change.
- Live tests are handler-scoped and use the cheat route (verification.md): prepare state by save edit, run one handler, compare bits.

## 8. Migration

1. Introduce `internal/bot/steps` with `Step/Plan/Executor` and the fakes; wire `quest`/`boss`/`act` goal types in `compile` (bot.md "Add a goal type" steps 1-4 each).
2. Port existing `farm_area` and `reach_level` paths as trivial plans (Travel + Clear) behind a flag; both paths keep passing `runner_test.go`.
3. Add handlers in the order of verification.md ranks; every handler ships with its `stepworld` test and a live checklist entry.
4. Remove nothing else from the runner. Total new Runner code: compile hook, `Classify`, constraint switches (~200 lines).
