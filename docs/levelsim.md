# levelsim: a simulator for Act 1 Normal leveling strategies

> status: current (2026-11-14, TASK-413, agent levelsim). Offline: no game, no d2d slot, nothing posted. Code: `tools/levelsim/` (package `d2lod-bot/tools/levelsim`, CLI `tools/levelsim/cmd/levelsim`), stdlib only. Consumer: the level plan v3 card (agent levelplan3) reads the committed summary [tools/levelsim/summary/best-strategies.json](../tools/levelsim/summary/best-strategies.json) and section 5 below.
>
> Tags: **V** verified (read from a log line, a table or a test, the source is named), **S** secondary (an earlier doc of ours, named), **I** inferred (arithmetic or a fitted model), **U** unverified. A number from the model is I unless it says otherwise.

## 1. What it does and how to run it

The simulator answers one question: given the areas of Act 1 Normal, what order of areas, entry levels, exit rules and game restarts gives the most XP in 60, 120 and 240 minutes, and the least time to clvl 10 and 15, for a Paladin at the bot's real pace?

```
go run ./tools/levelsim/cmd/levelsim -calibrate                     # replay the logged hours, fit, leave-one-out
go run ./tools/levelsim/cmd/levelsim -optimise -mode all -out tools/levelsim/out   # best strategy per horizon, JSON for plan v3
go run ./tools/levelsim/cmd/levelsim -sens -mode plan -horizon 60,120              # sensitivity table
go run ./tools/levelsim/cmd/levelsim -strategy my.json -horizon 60,120,240         # simulate a policy file
go run ./tools/levelsim/cmd/levelsim -extract bot.log -run-id 5 -start 2026-11-15T10:00:00Z   # derive a runs.json entry
```

Flags: `-horizon` (minutes, comma list), `-mode plan|strict|bold|all` (danger floors, section 4), `-width` (beam width per class, default 60), `-reach` (horizon of the time-to-clvl search, default 480, 0 = skip), `-den-refarm`, `-fitted` (refit the supply parameters first), `-runs FILE` (extra runs added to the embedded ones), `-out DIR` (writes `best-strategies.json`). The full `-optimise -mode all` run takes several minutes on the 4-core box; use the worker host (`labrun orion -- go run ...`) for sweeps.

A policy file (`-strategy`) is `{"name": "...", "restart": true, "rows": [{"area": 2, "enter": 1, "leave": 4, "min_f": 0}, ...]}`: rows in priority order, the first row whose clvl window contains the hero and whose area is still alive (more than `min_f` of it, default 0.08) is farmed; with `restart` true a fresh game starts when no row is open. Area ids are the Levels.txt ids of `internal/plandata/areas.json`.

## 2. The model

State: minute, XP, clvl, location, kills per area in the current game, activated waypoints, Den done, Blood Raven done, games started. Actions: travel to an area (graph walk, town portal, waypoint hop), farm an area until an exit rule fires, save and exit (every area respawns, the quest and waypoint state stays).

| piece | model | tag |
|---|---|---|
| XP per kill | mean over all units of the area (plain, uniques and minions) at the hero's clvl, linearly interpolated between the clvl 1/3/5/8/12/16 columns of [areamon-table.csv](research/areamon-table.csv) (agent decomp18, game.exe room populations, 12 seeds, level-gap table of xp-gold-trace.md); copied as `tools/levelsim/data/areas.csv`, no game data | V run (decomp18) |
| level thresholds | 500 / 1,500 / 3,750 / 7,875 (clvl 5) / 14,175 (clvl 6) / 22,680 ... 176,132 (clvl 15) | V (`internal/telemetry/exp.go`, xp-gold-trace.md Q6) |
| fresh kill rate r0 | Blood Moor 10.0, Cold Plains 17.0, Den 10.8 kills per minute (V: hour 4 five-minute windows 8 to 11 and 14 to 19, hour 1 Den first pass 400 XP per minute at 37 per kill); Stony Field, Dark Wood, Black Marsh, Tamoe: areamon's 14.6 x units / 198 scaled by 17/14.6 so that Cold Plains agrees; dungeons: areamon's `rate` column unchanged | V (three areas), I (rest) |
| supply | the area holds `units` monsters (mean of 12 seeds); kills so far in this game `k`, remaining share `f = 1 - k / (yield x units)`; rate = r0 x scale x (floor + (1 - floor) f^decay); no kills beyond `units`; a re-visit in the same game continues from the same `k` (re-clears at the floor rate, as in hour 1: 171 against 337 XP per minute) | I (fitted, section 3) |
| walking | graph of neighbouring areas with walking minutes (town to Blood Moor 0.2 min, V 12 s in the logs; Blood Moor to Cold Plains 2.0 min and every other edge U: `tools/levelsim/data/graph.json`), a town portal hop 0.35 min from anywhere, a waypoint hop 0.35 min from town to an activated waypoint (Cold Plains, Stony Field, Dark Wood, Black Marsh, Outer Cloister, Jail 1, Inner Cloister, Catacombs 2; a waypoint activates when the hero walks through it) | U |
| game restart | Save and Exit to the next town arrival: 15 s (V: 10:09:47 to 10:10:02 and 10:46:26 to 10:46:41 in hour 4), plus the walk back; the bot's own restarts are the "everything worth farming is cleared, starting a fresh game" and save-point lines | V |
| chores | a town tax of 6.5 percent of game time in policies and the optimiser (S: first-hour.md "5 to 8 percent"); in a replay of a logged run the town time is the logged one and the tax is off | S |
| Den of Evil | the skill point is given when 88 percent of the Den is killed (a clear); the hero needs clvl 3 (campaign floor); the quest is mandatory in every strategy; farming the Den again after the claim is off by default (levelplan.md change 1) | S, U (the 88 percent) |
| Blood Raven | clvl 8 (campaign floor, boss1-trace min safe level 8); done when 70 percent of Burial Grounds is killed, +181 XP (V boss1-trace) and 1 minute for the Rogue; mandatory in a strategy that ends at clvl 9 or more | V, U |
| survival | not simulated. Danger enters only as entry floors (section 4); deaths, potions, gold, items, merc, shrines and the Den skill point's damage effect are outside the model | U |

## 3. Calibration

Inputs (derived rows only, `tools/levelsim/data/runs.json`; the raw logs are not committed): run 1 (area entries from hour1-analysis section 3, final row 17,032 XP clvl 6, 528 kills), run 4 (59 per-minute rows and 19 area entries from the bot log, final 13,883 XP clvl 5, 494 kills, Den hunts that killed nothing listed as dead minutes), runs 2 and 3 (summary rows only: clvl 5, 445 and 355 kills; the container log of those hours was rotated before this work, so there is no area sequence). The `hero14` log is a parked hero and not used.

Method: `Replay` feeds a run's logged area sequence and restart times through the kill model (field time in each logged area, no chores, no walking: those are in the log), and compares XP and kills at minute 60 and, for run 4, XP at every logged minute. The free parameters are decay, floor and yield (343 grid points, `Calibrate`); the fresh rates of the three observed areas are fixed from the logs and the rate scale stays 1. A weak prior (yield 0.82, floor 0.4, decay 1: hour 4 section 2.3, Cold Plains cleared at 163 of about 198 units, tails under half the fresh rate) breaks ties because two runs cannot identify three supply parameters.

CALIBRATION_TABLE

Reading (I): both runs are inside the 10 percent target on XP and kills at the default parameters, except run 1 which is 9.4 percent low on XP; run 4 follows its own minute curve within 3 percent of the final XP on average. Run 2 and run 3 have only a consistency check: observed kills times the lowest and highest XP per kill of the starter fields puts the XP inside the observed clvl 5 band (7,875 to 14,174). The old AREAMON route model had 13.1k against 13.9k on hour 4 (6 percent); this model is at 0.6 percent on the same hour, but it was fitted on that hour, so the held-out numbers are the honest ones.

The fit is flat: 12 of 343 grid points are within 25 percent of the best loss and they span decay 0.25 to 0.5, floor 0.3 to 0.7, yield 0.95 to 1.0. The data prefer a rate that stays near the fresh value until the area is nearly empty and then falls to about half; the "supply tail under 8 per minute" of hour 4 is mostly the end of an area, not a smooth decay. The sensitivity table (section 6) includes a decaying-supply variant (decay 1.5, floor 0.3, yield 0.85) to show how much the answer moves.

How to re-calibrate after new hours: extract the hour with `-extract LOG -run-id N -start <the hero's first moment, RFC3339>` (prints a runs.json entry of minute rows, arrivals and restart times; a Den hunt that killed nothing can be added by hand to `dead`), append it to `tools/levelsim/data/runs.json` (or pass it with `-runs FILE` to try it first), run `-calibrate`, and copy the fitted parameters into `DefaultParams` when the held-out errors stay under 10 percent. `TestCalibrationRegression` pins the two replays at 12 and 10 percent.

## 4. Optimiser and constraints

`Optimise` is a beam search over decision points: from a state, every farm slice (area x exit rule: leave at 50, 25 or 8 percent of the area left, or at the next level-up) and a game restart; slices end at the horizon. States are bucketed by the minute they end, pruned to the best `width` per (Den done, Blood Raven done) class by XP, with near-duplicate states (same place, XP bucket, kills pattern) dropped. It maximises XP at the horizon; the time to clvl 10 and 15 is the earliest level-up time over every expanded state (a longer search, `-reach`). It is deterministic (no random numbers). `TestOptimiserFindsToyOptimum` pins it on a two-area table; the beam is not an exact dynamic program, so the optimum is a lower bound on the model's true optimum (ranked alternatives within 0.2 percent of each other at 240 minutes suggest the beam is converged on the shape).

Hard constraints: the Den quest done, Blood Raven done when the hero ends at clvl 9 or more, the Den not farmed again, and an entry floor per area in one of three danger modes (all U; the danger side is the largest uncertainty):

| mode | entry floor of an area | source |
|---|---|---|
| `plan` | today's melee entry level of the plan row (BM 1, Den 3, Cold Plains 4, Cave 1 6, Stony Field 7, Dark Wood 9, Black Marsh 10, Burial 8, Jail 1 13, Catacombs 2 13, ...); areas without a row: mean monster level + 2.5, rounded up (the rows run +2.4 to +4.3) | S (SME numbers in levelplan.md), I |
| `strict` | the larger of mean monster level - 1 (areamon's safety proxy) and the boss floor of the area: Den 4 (Corpsefire), Cold Plains 5 (Bishibosh), Stony Field 6 (Rakanishu), Cave 1 6 (Coldcrow), Dark Wood 8 (Treehead), Burial 8 (Blood Raven), Tower Cellars 11 (Countess), Barracks and the Jails 14 (Smith) | I (boss1-trace.md min safe clvl) |
| `bold` | mean monster level - 2; only the quest floors Den 3 and Blood Raven 8 | I |

`plan` is the default and the one the plan v3 agent should treat as "what is safe today"; `strict` and `bold` show what lowering the floors would buy. Today's plan is simulated as a policy (`PlanToday`): the farm rows of `progression-normal.json` in file order, the Den at clvl 3, Burial Grounds at melee 8, a restart when no row is open (`TestPlanRowsMatchPlandata` keeps the data file and the plan in step).

RESULTS_SECTION

## 7. Using it for the level plan v3

- Machine-readable: `tools/levelsim/summary/best-strategies.json` (committed) and, after a local run, `tools/levelsim/out/best-strategies.json` (ignored). Per danger mode: `plan_today` and `best` per horizon (XP, clvl, area order, `bands` = area, enter clvl, leave clvl, minutes, kills, XP, restarts before, clvl by 10-minute mark), `ranked_at_longest_horizon`, `minutes_to_clvl`, `gain_vs_plan`, and `sensitivity`.
- Reading the bands as plan rows: a band is "enter area X at clvl a, leave at clvl b, with this many game restarts between the stretches". The simulator's exit rule is "leave when the area is 92 percent killed" (or 50 or 25 percent in the first rows where a later area pays more), which is the plan's `leave_clvl` replaced by a depletion rule; the row order in the optimum is by pay, not by area number.
- Never copy a band below the entry floors of the plan you consider safe; the `plan` mode already respects them.
- What the model cannot say: whether any floor is right (section 8), how many chores a deeper area costs, and what a Den skill point is worth.

## 8. Limits and what is unverified

- U: every danger floor. The `strict` and `bold` results are what the plan could gain if the floors were lowered; whether a clvl 5 Paladin with the kit survives Black Marsh or Tamoe is not known. Run the hidden-instance drill of areamon section 12 (X4) before moving a floor.
- U: supply shape (decay, floor, yield): two runs with an area sequence cannot identify it; section 6 shows the answer's dependence.
- U: all walking minutes except town to Blood Moor; the waypoint rule (a waypoint activates on the way); the restart walk back to the field.
- I: fresh kill rates of every area except Blood Moor, Cold Plains and the Den: areamon's time model, which was calibrated on those three only (areamon section 14). Dungeon levels beyond Cave 1 are extrapolated. Small second floors (Underground Pass 2, Hole 2, Pit 2) pay high XP per kill because they hold a high share of uniques and minions; the optimiser likes them whenever the floor allows.
- I: XP per kill is the mean of all units; uniques and minions come in packs (a clear is not a random draw), and one Blood Raven kill is the only boss in the model.
- Not modelled: death and recovery, potions and gold, item finds, shrines, the merc, Zeal at clvl 12, Act 2 and later, bot failure modes (the Den exit tile of hour 4, nav stalls: `plan_today` is the plan as intended, 19.5k XP at 60 minutes in `plan` mode against 17.0k and 13.9k in the two logged hours with area data), and d2d quirks (`docs/d2d-deviations.md`).
- The beam is not an exact optimum; the model's true optimum is at least as good as every number printed.
- d2d monster population per area was never compared with game.exe's (areamon D1, U).
