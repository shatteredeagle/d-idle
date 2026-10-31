# bot: the core that ties the layers together (agent BOT)

Code: `internal/bot` (runner, planner, profiles, death policy), `internal/env/tuning_bot.go` (knobs), `cmd/d2bot` (the binary).
Everything is per instance (docs/net/design-now.md): one `env.Env`, one `bot.Runner`, one death log per instance, no package-level game state.

## Architecture

```
cmd/d2bot  flags/env -> game (attach | managed) -> env.Env -> behaviors -> bot.Runner -> goals.Manager -> api + dashboard
                                                                   ^ Mover Fighter Townsman Looter (consumer-side contracts)
bot.Runner:  Run goroutine (decide + one action per step)  +  guard goroutine per running action (watch, never acts)
```

- `contracts.go` holds the agreed method sets (docs/archive/wave2.md) and the argument/result types. At integration each type becomes an alias
  of the implementer's type or a thin adapter converts it; nothing else in the bot changes.
- **Run loop** (`loop.go`): each step polls a `d2.Snapshot` (observer emits events), reads the control state (goal, pause, stop), then makes one
  decision and runs at most one action under `runAction`. Pause/Resume/Stop/Start/Status are O(1), take a mutex, never touch the game or the Manager.
- **Guard** (`guard`, every `bot.watchMs`, only while an action runs): keeps the observer fed (kill/drop/level-up/area events are live), and cancels the action
  with a typed cause for: death, game gone (`launch.Monitor`), left the game (menus), pause due and safe, `Fighter.Safety` (drinks are left to the fighter while
  Clearing; TownPortal and Chicken always interrupt), town trip due, level changed (reach_level re-plan), watchdog (no progress for `stuckAfterSec`).
  The main loop then reacts (e.g. `Fighter.Act`), so input stays single-threaded.
- **Watchdog**: progress = moved 3 subtiles, area change, kill, drop, potion, level up, hostile life falling. Ladder (prior-art 07 section 7):
  close panels, reset in town, then a visible non-fatal Error state with a retry timer. Failed actions back off and escalate after `maxStepFailures`.
- **Events** are derived from snapshots (Kill with boss flag and XP split, Drop from new inventory/belt items in the field, Potion from items leaving the belt,
  LevelUp, AreaChange, Death with killer). If a behavior package emits the same events through `env.Emit`, list them in `disableObserve` to avoid double counting.
  The channel is long-lived (2048); when full the oldest event is dropped.
- **Fatal** (`Error{Fatal:true}`, `Run` returns the error, `Start` is refused afterwards): the game process is gone/crashed/hung, or game memory is unreadable with no
  health monitor. Everything else is a non-fatal Error event plus `Status.State == error` with a retry (`errorRetrySec`) or a new goal/fresh game.

## State machine

```
              Start                        Stop (finish in town)
   Idle <------------- any ------------------------------------+
    |  goal                                                    |
    v                                                          |
 InTown --chores due--> Resupplying --BackToField--> (field) --+
    |  target != here        ^  trip due / emergency TP        |
    v                        |                                  |
 Traveling --arrived--> Clearing --area cleared--> (new game via Cycler, or Error: waiting)
    |                        |
    +------ died ------------+--> Recovering --respawn--> chores / corpse policy --> back to the goal
 Pause (safe moment) -> Paused ; fatal/stuck ladder -> Error
```

`Status.State`: running while it plays (including idle goals: they accrue time as running), paused, idle (no goal), stopped (after Stop), error.

## Goals

- `idle`: rest in town (go home first, chores if due).
- `farm_area`: resolve area (id, `run:<key>` final area from runs.json, game-table name, areas.json name, goals catalog) at Start, travel
  (waypoint to the furthest waypoint area on the route when standing in a waypoint area, else exit by exit along `Mover.Route`; each hop verified by area id),
  then sweeps: Buff once, `Fighter.Clear` when hostiles are within `bot.clearRadius`, `Looter.PickUp`, `Mover.Explore(done = hostile near)`. When exploration runs out the
  area is cleared for this game; with a `Cycler` the bot asks for a fresh game, otherwise it waits visibly (Error state, resumes on a new game).
- `kill_boss`: boss name -> run in runs.json -> same as farm_area on its final area (boss scripting is not done; the manager counts the kills).
- `reach_level(n)`: planner (`plan.go`) picks the area from progression-<difficulty>.json for the current difficulty and level: among reachable-act, non-town, not cleared,
  not avoided steps with `enter + levelMargin <= level`, the last whose window still contains `level - leaveSlack`; past all windows the last eligible; re-planned on level up.
  Plan data is embedded (`internal/plandata`, shared with the leveler, kept equal to docs/plan/data by a test) and overridable per file with `planDir`. The loader ignores unknown fields, accepts aliases
  (`area`/`area_id`, `enter_clvl`/`min_clvl`, `leave_clvl`/`target_clvl`) and skips broken entries with `Plan.Warnings`.
- `collect_item`: not supported (Start returns an error, the goal shows as failed).
- Resupply trips: healing/rejuv potions in the belt below `bot.resupplyPotions` (0 potions is an emergency, shorter cooldown), inventory occupancy at `bot.invFullPct`, after an emergency
  town portal, and in town at goal start. Repair: always requested with potion/sell trips; the bot cannot read durability yet (the Townsman decides).

## Death (PLAN.md "Death is normal")

Detected from the snapshot stream (player mode 0/17 or life 0; `diedTransition(prev, cur)` is the pure rule, replace it with COMBAT's `Died` when it lands). `Death` event carries killer name
(nearest hostile at the last living snapshot, UNVERIFIED heuristic), area, level-independent. A `DeathRecord` stores area, corpse position, killer + enchants, life curve (last 5 s),
estimated corpse value (base price x quality factor, crude), cost afterwards (gold, items, seconds), decision text.
- **Recovering**: wait for the respawn (`recoverTimeoutSec`), then decide: retrieve only if corpse value >= `bot.corpseValueMin`, not killed by a boss, area not avoided, character level fits the plan window,
  fewer than `spiralDeaths-1` recent deaths there, attempts left (`bot.corpseRetries`), softcore. Chores first; one attempt (`Corpse` hook or click-the-spot default, UNVERIFIED); a second death
  on the way aborts and never retries. Otherwise abandon and rebuild: `ChorePlan{Poverty: true, GearTier: bot.povertyTier, GoldFloor: bot.goldFloor}` (potions first, cheapest viable weapon/armor; the Townsman implements it).
- **Spiral breaker**: `bot.spiralDeaths` deaths in one area within `bot.spiralMinutes` -> the area is avoided for `dangerBanSec` (planner picks elsewhere; farm goals wait in an Error state),
  the killer's danger weight is raised (`DeathReport().Danger`, for the fighter to read later), and a non-fatal Error event "Death spiral ..." is emitted.
  `bot.pauseDeathsPerHour` deaths in the last hour -> the runner pauses itself (`Status` paused, message says why; Resume continues).
- **Persistence/display**: the death log (records, per-area and per-monster counters, heat map cells of 8 subtiles, danger weights) is saved atomically to `deathLogPath`
  (default `<data>/instances/<id>/deaths.json` (paths.Layout; older `state/instances/<id>/` files are moved on start)). `Runner.DeathReport()` and `GET /api/v1/bot/deaths` expose it. Not done: monster level and affix names in the record, recommendations text, gear insurance.

## Tuning and handicap profiles

Knobs live in `internal/env/tuning_bot.go` (tick, watch, reaction, resupply threshold, inventory-full %, level margin, overstay, corpse retries, settle time, clear radius, the death
knobs, leveling, and the merc and cube chore switches; `noHandicap` knobs, the unverified switches included, are never moved by a profile). The knobs `bot.starveGold` and `bot.starveRecoverGold` were removed (no reader); old config.json `tuning` entries naming them still load and are ignored (`env.RetiredKnobs`). Defaults are optimal. `bot.Profile` degrades a Tuning copy: each knob moves from its default toward the worse end of its documented range by a strength in 0..1
(`Level` for all, `Per` for a key or a `group.` prefix). Built-ins: optimal, relaxed, casual, sloppy, rookie. Tests: every knob resolves to a field (so other agents' knobs are checked
the moment they merge), no profile leaves a range or beats the default, profiles are monotonic. `SetKnob`/`GetKnob`/`ApplyOverrides` work by dotted key via reflection.
Settings from the dashboard (`chickenLifePct`, `potionLifePct`) are applied to `combat.chickenLifePct` / `combat.potionLifePct` if COMBAT defines those knobs.

## Config reference

Flags and environment (every flag has a `D2BOT_*` variable): `--data/D2BOT_DATA`, `--config/D2BOT_CONFIG` (default `<data>/config.json`), `--addr/D2BOT_ADDR` (default 127.0.0.1:8420, other
addresses need `--token/D2BOT_TOKEN`), `--helper/D2BOT_HELPER`, `--gamedir/D2BOT_GAMEDIR`, `--instance/D2BOT_INSTANCE`, `--log-level`, `--allow-host`, and exactly one of
`--sim` (simulated runner, no game), `--attach` (dev: already running game via mem.FindGame), `--managed` (launch game and helper through internal/launch, subreaper). `--dry` (with attach/managed):
read-only, prints what it sees each second, `--once`, `--dry-for`, only `/healthz`. `--stub-behaviors`: failing combat/town stubs and a fake input controller (nothing reaches the game).
Endpoints besides the API: `/healthz` (503 when the game is gone or the runner is in Error), `/api/v1/bot/deaths`, `/api/v1/bot/state`.
Logs: JSON slog on stdout with `inst`. Shutdown (SIGINT/SIGTERM): `Manager.Stop` (finish in town), wait, save, close the API; `--stop-game` also stops a managed game.

`config.json` (or its `"bot"` object), strict (unknown keys are errors): `profile`, `custom` {Name, Desc, Level, Per}, `tuning` {"bot.clearRadius": 20}, `planDir`, `acts` (default [1]; act changes need quests),
`hardcore`, `deathLogPath`, `disableObserve`, and seconds: `stuckAfterSec` 120, `travelTimeoutSec` 300, `sweepTimeoutSec` 240, `townTimeoutSec` 180, `stopTimeoutSec` 60, `recoverTimeoutSec` 60,
`menuGraceSec` 8, `resupplyCooldownSec` 240, `errorRetrySec` 300, `maxStepFailures` 5, `dangerBanSec` 3600.

## Add a goal type

1. `goals` package: add the type, params and completion rule (the manager owns progress).
2. `Runner.compile`: resolve names and validate (the only place `Start` fails); add fields to `goalSpec`.
3. `Runner.runGoal`/`target`: return the area to work in, or handle the type before the trip/travel logic (like `idle`). Use `r.do(...)` for actions so guard, errors and the ladder apply.
4. Add a fake-world test in `runner_test.go` (see `rig`, `world`, virtual clock) and update this file.

## Failsafes (PLAN.md "Failsafes", layer 1)

- **Watchdog** (`stuckCheck`, runs in the guard during actions and at every loop step): progress is a monotonic counter (`HealthReport.progressCounter`) fed by position change, area change, level up, experience, gold,
  kills, damage dealt, items. Two clocks: per state (since the last progress or state entry; `stateStuckSec`, defaults recovering 180, resupplying 240, traveling 240, in_town 300, else `stuckAfterSec` 120) and global
  (`globalStuckSec` 600, catches state flapping). Idle, Paused and Error states and the idle goal resting in town are exempt: doing nothing is their job.
- **Ladder** (`escalate`), one rung per expired window, each rung re-arms the watchdog: 1 close panels (dead: Esc x3); 2 reset to town (skipped when already there or dead); 3 leave and re-enter the game through the
  `Cycler` (skipped when none); 4 `Deps.Restart` game restart hook (skipped when none; not wired in cmd/d2bot yet); 5 pause itself (`Status` paused, non-fatal Error event "Failsafe: ...", Resume continues).
  Only experience, kills, damage, items and gold restart the ladder (movement and area changes are caused by the ladder itself).
- **Death overlay**: while dead the bot presses Esc 2 s after the death, every 3 s for 5 presses, then every 15 s (never while alive: an Esc in the field opens the game menu) and logs the dead state
  (`dead` lines: mode, life, UI flags). A dead character that never comes back goes up the same ladder.
- **Health**: `GET /api/v1/bot/health` (same token as the API) returns `{state, status, activity, sinceSeconds, lastProgressSeconds, progressCounter, stuckReason, lastError, ladderStep, paused, selfPaused, goal, fatal}`.
- **Drills** (`failsafe_test.go`, fakes, virtual clock): travel with no progress, clearing with no progress, a town trip that never returns, a panel left open (pickup keeps failing), a death overlay nobody dismisses, game gone
  (fatal), ladder reset by real progress. Each ends in a visible pause (or the fatal error), never a hang.

## Integration status

- Wired: NAV, COMBAT, TOWN, LOOT through adapters in `cmd/d2bot/adapters.go`; the fighter's Leave hook is `launch.Flow.LeaveGame`; `combat.Portal` is left at COMBAT's default (UNVERIFIED geometry; the Townsman opens its own
  portals in `ToTown`, and an already open portal next to the player is reused). Combat emits Kill/Potion and loot emits Drop, so `disableObserve` is set to kill, drop, potion automatically.
  The Cycler (`launch.Flow` Start/LeaveGame) is wired but UNVERIFIED end to end. Hotkeys are bound at startup (`EnsureHotkeys`).
- Live display: `/live/` shows overview, timeline, metrics (telemetry fed from the runner's snapshots through `Deps.OnSnapshot`), character and inventory. The world panel is not wired (nav owns the map tracker).
- Mismatches found (owners): TOWN `loot.minValuePerCell` and NAV `nav.exploreTimeoutSec` knob keys have no matching field (skipped with a warning by `BuildTuning`).
- Leveling (docs/leveler.md): points, gear, starvation, runs and difficulty rules are wired (see "Leveling runs"). Not done: respec, real quest claim verification, act changes, kill_boss scripting, collect_item, restart hook, danger weights to combat, combat.Died instead of the own
  detection, world panel.

## First live runs (instance i4, character Booti, level 1 softcore Sorceress, 2026-10-28)

All runs: `tools/live.sh bot d2bot --attach --goals ... --run-for ...`, the worker host, Xvfb :104, Fire Bolt bound to F8 by `EnsureHotkeys` (VERIFIED live). The stats in the final `summary` log line accumulate in the data dir, so the numbers
below are counted from the `combat kill` log lines of each run.

| run | goal | length | result |
|---|---|---|---|
| 3 | farm Blood Moor | 360 s | 16 kills (Quill Rat, Zombie, Fallen), 273 XP = about 2,700 XP/hour at level 1, 3 combat potions, kiting 7 times, 1 target given up, 1 resupply trip (potions low: portal home in 1 s, chores 16 s, portal back, 27 s total), 0 deaths, 0 stuck |
| 7 | farm Blood Moor | 150 s | 9 kills, level 2 reached, the "clear killed nothing" rule fired 5 times (see below), 0 deaths |
| 9 | reach_level 3 | 9 min | 4 kills, 1 death (Fallen, Cold Plains, level 2, corpse value 168 gold), then 7+ minutes stuck on the death overlay (see below) until the run ended |
| 10 | reach_level 3 (new binary) | 8 min | helper `127.0.0.1:47016` died mid-run: the failsafe ladder ran rungs 1 to 5 in 25 s (Esc, town (no portal scroll), leave and re-enter through the OOG Cycler (worked: new game in 1.5 s), restart (no hook), pause) and left a visible paused state with the error text; 0 kills |

Level 3 was not reached (level 2 at about 500 XP, level 3 needs 1,500; the character has no gold for potions or portal scrolls, so every run ends in resource starvation).

Failures by cause (and owner):
- **Death overlay not dismissed (bot, fixed).** The first death left "You have died. Press ESC to continue." on screen and the bot waited in Recovering for 7 minutes. Now: Esc after 2 s, repeated, verified by respawn; a dead character climbs the failsafe ladder.
  Verified live afterwards: one Esc through the helper respawned Booti in the Rogue Encampment (area 1), alive. The dead-state samples (`dead` log lines) are recorded by the new code but no second death has happened yet; the first death's mode and UI flags were not captured.
- **Level-up HUD buttons counted as open panels (d2, fixed).** After level 2 the red stat/skill point buttons set UI flags 6 and 7, `AnyPanelOpen` said a panel was open and every pickup failed with "hand: panels still open". Both flags are ignored now.
- **Routing (bot, fixed).** `levels.txt` Vis data gave no neighbors for the Rogue Encampment and none from Cold Plains to Blood Moor, so every travel failed with "no route". Fixed with an undirected graph plus an embedded Act 1 fallback graph (R, UNVERIFIED). SUPERSEDED: the bot no longer has a graph; `Mover.Route` (nav.Route, seen exits then levels.txt then its own Act 1 hints) is the only one, `graph.json` is no longer loaded (the file was deleted with the plandata unification F15).
- **`nav: nothing left to explore` (bot, fixed).** It was treated as a failure and walked up the ladder; it now means the area is cleared.
- **Fighter that kills nothing (bot, fixed).** 3 hostile units within 30 subtiles that COMBAT never engaged (unreachable or avoided) made the sweep loop in place for 2 minutes. Now the bot moves 40 subtiles away before reacting to them again.
- **Exploration leaves the target area (bot, fixed).** Explore walked from Blood Moor into Cold Plains; the guard now walks back.
- **NAV: leaving the Den of Evil.** `ExitTo(2)` from area 8 returns "no exit to that level" (dungeon exits are not border gaps), so the planner's move from the Den back to the Blood Moor failed 5 times (rung 3 of the ladder re-entered a fresh game).
- **NAV/TOWN: this character's town is at different coordinates** than Testy's (Rogue Encampment around x 5150, y 4230 against x 6080, y 4914); `navlive town` gets stuck at (5501, 4569) on i4, the Akara click failed once ("pointer never hovered the target").
- **TOWN: nothing can be bought with 0 gold** ("gold did not change"), and the "low on mana potions" advice fires forever for a character that never has any (`ignoreTownAdvice` works around it).
- **INFRA: the d2input helper of i4 (47016) died twice** (after run 9 and during run 10) with no bot involvement that I could find; `tools/game.sh helper i4` revives it. Worth a look by MULTI.
- **Knob keys** `loot.minValuePerCell` (TOWN) and `nav.exploreTimeoutSec` (NAV) do not match a field; skipped with a warning.

Not exercised live: corpse retrieval, poverty rebuy, spiral breaker, chicken exit, town portal in a fight.

## Leveling runs (instance i4, Booti, 2026-10-28, `d2bot --attach --goals reach:10`)

Details of the mechanisms: docs/leveler.md. Numbers are counted from the run logs.

| run | length | result |
|---|---|---|
| A | 6 min | Den of Evil at level 2 (NAV exits worked: Blood Moor to Den and back), 11 kills, +290 XP (621 to 911), 1 chicken exit (life 19/51, 9 hostiles) with re-entry through the OOG flow in 9 s, gear pass in town (budget 147, nothing better to buy), then the d2input helper of i4 died: ladder rungs 1-5 ran, paused |
| B | 10 min (helper dead after 7.5) | Blood Moor, 43 kills, level 3 reached at 1,500 XP (exp 1844 at the end), points spent IN THE FIELD 0.4 s after the level-up: 5 vitality + Charged Bolt in 6 s, verified (`statLeft 0 skillLeft 0`), 1 death (Zombie, Den of Evil, level 3) while the helper was down and the chicken exit could not be sent, 4 abandoned town trips (no town portal scroll, 0 gold), 7 ladder rungs |

Rates: B gained 1 level plus 1,220 XP in about 7.5 live minutes at level 2-3, about 8 levels per hour extrapolated at this level (not meaningful beyond level 5); deaths 1 in 16 minutes of play; stuck events 2 (both the dead helper); failures by cause:
- **d2input helper of i4 dies after 5-8 minutes of any bot run** (3 times in this session, `connection refused 47016`; `tools/game.sh helper i4` revived it twice and then reported "did not start"): INFRA, it caused the death in run B and both ladder runs. Biggest blocker for unattended play.
- **Town table offset** (TOWN): Booti's Rogue Encampment sits at another origin, so chores walked out of town (sell, shop skipped). `calibrateTown` in cmd/d2bot is a workaround, not exercised in a full run yet.
- **No town portal scroll/tome and 0 gold** (resource starvation): trips abandoned; starvation rule (farm Blood Moor until 150 gold) is unit tested, not seen live (the character found potions and gold instead).
- Den of Evil at level 3 with few potions: the potion gate (fewer than 2 healing potions skips the Den) was added after run A; run B still entered the Den after picking up potions, and died there.
- Not exercised live: Den quest claim (the Den was never cleared), difficulty switch, gear shopping with the bot loop (only through levelerlive).

## Integration follow-ups (CORE, 2026-10-28)
- Death uses the verified facts (`combat.IsDead`, `CanRespawn`, `EscRetryEvery`, `GoldPile`; V, docs/combat.md "Death"): Esc only from mode 17, every 400 ms; default corpse retrieval walks onto the corpse then onto the gold pile (class from `Catalog.Item(c).GoalType == "gold"`), no clicks. Tests replay the recorded `internal/combat/testdata/death/*.jsonl`.
- `input.Game` has `Clock` and `Ctx` (wall clock / no cancel when nil). The bot failsafe tests use `input.Fake`, so their flakes were test-side races (state flips before the collector goroutine sees the event): fixed by waiting for the event.
- Panel flags: one table in `internal/d2/ui.go` (`UIKindOf`, `BlockingPanelOpen`, `EscClosableOpen`, `UnknownOpen`); `input.Reader.UIOpen` uses it. Still own copies (not CORE's directories): `internal/town/hand.PanelOpen` (can become `u.BlockingPanelOpen()`), `internal/sentinel/observe_linux.go openPanels` (use `Indexes(d2.UIPanel)` minus `UIEscMenu`).
- `cmd/d2bot --version` prints `d2bot <version>`; `var version = "dev"` is stamped with `-ldflags -X main.version=...`.

## Data paths: migration status (PATHS)
Done: cheat ledger `<data>/ledger/cheats.jsonl` and backups `<data>/ledger/backups/`; sentinel log `<data>/logs/sentinel.jsonl`; d2watch token `<data>/tokens/d2watch`; game locks `paths.Instance.GameLock` (`$XDG_RUNTIME_DIR/d2bot/game-<id>.lock`) taken TOGETHER with the legacy `/tmp/d2lod-game[-<id>].lock` (launch.LockInstance, sentinel, tools/live.sh, helper.sh; drop the legacy one when no old binary is deployed); data tables `data.OpenCached` -> `<data>/cache/data/v<schema>-<stamp>/` with legacy dirs copied in; launch/cheat game roots via `paths.Layout.GameRoot` (new `<data>/prefixes/<id>`, or dev-era `~/Games/d2lod[-inst/<id>]` kept in place while it exists: a live Wine prefix is not moved automatically). Shell mirror: `tools/lib-paths.sh`. Old locations are read as fallback and moved on first start (tests: internal/paths, internal/cheat, internal/data).
Not done (other owners): cmd/d2bot live.go still uses `<data>/cache/tables` and `~/Games/d2lod-inst`; tools/rigs/levelerlive, internal/town/cmd/townlive, internal/combat/cmd/combatlive and internal/town/sim use hard-coded Games/ and /tmp cache paths (dev tools); moving the Wine prefixes into `<data>/prefixes` (stop fleet, move, by hand).

## Integration: resource policy, campaign goal, LIVEBOT follow-ups (INTEGRATE, 2026-10-28)

What is wired (details: docs/resource.md, docs/task.md section 6, docs/quest.md, docs/launch.md section 9): the resource policy in combat/bot/town and 28 `resource.*` knobs in the Tuning view; the `campaign` goal (`--goals campaign`, Normal Act 1) with `taskStep` inside the usual `r.do`, a save-fed quest reader and a town dialog; `--autostart`; hotkeys bound by the runner after login (`Deps.Prepare`); shutdown walks home and saves before the game stops; healer retry; sell chore capped (`town.sellMaxS`).

Live run (i4, Booti, clvl 6, `d2bot --managed --stop-game --goals campaign`, 31 minutes, 2026-10-28 19:56-20:27, the worker host load about 20 on 16 cores). Cheated: 500 gold once (d2cheat, ledger reason "empty TP tome") so the town portal path and chores could be tested.
| metric | value |
|---|---|
| XP | 2,186 to about 6,100 (+3,900; 7,600 XP/h over the 31 active minutes, about 10,000 XP/h while fighting in the Den, lower while the leveling step walked in circles) |
| kills / deaths / chicken | 203 / 1 (Dark Hunter, Cold Plains, corpse retrieved) / 1 chicken exit (Blood Raven) |
| potions drunk, town trips | 12 healing/mana, 5 trips (all by portal after the tome fix) |
| ladder rungs | 13 (all rung 3, a fresh game, from the same cause: `nav: no exit to that level`, see below); no pause, no fatal, no watchdog stall |
| campaign | Den of Evil: walk, 14 min clear (hunted leftovers 4 times), home by portal: worked. Reward claim: not achieved (dialog, below). Burial Grounds: reached, Blood Raven fight ended in a chicken exit at clvl 6 |

Findings (owner):
- **Empty Tome of Town Portal read as quantity 1 (town, fixed).** An empty tome has no stat list; `town.qty` defaulted to 1, so every portal attempt right-clicked an empty tome ("portal did not appear") and a character deep in a dungeon could not get out. Now 0 for `tbk`/`ibk`; scrolls bought later go into the tome (stat 70).
- **`ErrExplored` is not "all monsters dead" (task, fixed):** the Den was declared cleared with 11-19 monsters alive; the campaign clear hunts leftovers.
- **Level-up event ended the campaign** (goals, fixed).
- **NAV (request for NAVFIX):** `ExitTo` fails from the Den (8) to Blood Moor and from Cold Plains (3) to Cave Level 1 (9) with `no exit to that level`: dungeon entrances and exits are not border gaps. The campaign avoids an area it cannot enter for 15 minutes; leaving a dungeon works only by portal.
- **Akara dialog (FIXED by TOWNFIX, see docs/quest.md "Akara dialog by quest state"; the text below is the old finding):** the submenu is TALK (title) / Introduction / Gossip / Den of Evil / [Sisters' Burial Grounds once started] / Cancel (screenshots); the line count changes with the quest states, so the click position does. The first version clicked at once and the pointer sat on the row with the menu still open: `town.PickLine` now hovers 450 ms first, `quest.Menus` has the Burial row, and the row set comes from the save's states. Two live attempts after the fix opened the menu but the Den row was not confirmed (the menu flag stayed set after the click; Introduction showed as the highlighted line). Needs a screenshot-guided session next to a character with a cleared Den (cheat `quest set` pending is U).
- **Leveler shop (FIXED by TOWNFIX, docs/leveler.md "Full-pack purchases": no free hole for the item; old finding follows):** every gear purchase failed ("gold or pack did not change", pack 80-85% full of junk, 4 s each, 1-2 minutes per town visit): the pass now stops after two failures per vendor. Cause not found.
- **Resource policy live:** rest before packs fired 5 times; regeneration 7 mana in 21 s (45 max), about 130 s to full, not the U 100 s; melee fallback reach fixed (see resource.md).

## INTEGRATE2 (2026-10-28): boss scripts, recycler, class-aware start (V = offline tests, live facts in docs/boss.md section 6, docs/questmem.md, docs/soak.md)
- Campaign `KillBoss` runs the boss scripts through `TaskDeps.Boss` (cmd/d2bot/boss.go). A clean run counts as a kill (proxy evidence); an escape becomes `ErrPortalOpened`, a chicken exit `ErrLeftGame`.
  A town trip that falls due while a boss run is going cancels it (V live: "equipment at 15% durability" interrupted a Blood Raven fight three times and she healed in between).
- Quest reader: memory before the save (docs/questmem.md). Recycler between goals (docs/soak.md). Leveler pause (docs/leveler.md).
- Class-aware start (`cmd/d2bot/class.go`): `Deps.Prepare` first reads the logged-in character's class from memory and sets `combat.Fighter.Build` so the combat profile and the leveler's table agree
  (Sorceress on the leveler's "fallback" table fights with the Lightning profile; Paladin: profile `paladin`, table `paladin-normal30`); the choice is logged as `class-aware start`.
  An explicit `levelerBuild` in the config still wins. COMBATCLASS (profile per snapshot) and LEVELERCLASS (table per class) were already per class; this joins them and makes the choice visible.
- Debug: `--probe-states` logs changes of the player StatList (how the state bit array was found).
