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

Flags: `-horizon` (minutes, comma list), `-mode plan|strict|bold|all` (danger floors, section 4), `-width` (beam width per class, default 60), `-reach` (horizon of the time-to-clvl search, default 480, 0 = skip), `-den-refarm`, `-fitted` (refit the supply parameters first), `-runs FILE` (extra runs added to the embedded ones), `-out DIR` (writes `best-strategies.json`). The full `-optimise -mode all` run takes several minutes on the 4-core box; use the worker host (`labrun <worker> -- go run ...`) for sweeps.

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

```
run  obs xp  pred xp  xp err   obs kills  pred kills  kills err  clvl pred/obs  per-minute error
1    17,032   15,434   -9.4%    528        533         +0.9%      6/6           (no minute rows)
4    13,883   13,805   -0.6%    494        521         +5.4%      5/5           2.9% of final xp (mean over 59 minutes)
2    no area sequence: 445 kills x 22..33 xp per kill = 9.8k..14.6k, inside the clvl 5 band (7,875..14,174)
3    no area sequence: 355 kills x 22..33 xp per kill = 7.8k..11.7k, inside the clvl 5 band
leave one run out (the fit uses the other run only, 2 runs have area data):
  run 1 held out: predicts 15,039 xp against 17,032 (-11.7%), kills -2.1%
  run 4 held out: predicts 13,353 xp against 13,883 (-3.8%), kills +1.8%
```

`levelsim -calibrate` prints this table. Error summary: on XP at minute 60 the fit is 0.6 and 9.4 percent off on runs 4 and 1; held out, 3.8 and 11.7 percent. The brief's target of 10 percent holds for run 4 in and out of sample and for run 1 in sample (9.4), and misses by 1.7 points when run 1 is held out. Runs 2 and 3 cannot be replayed (U).

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

## 5. Results (danger mode `plan` is the answer; `strict` and `bold` show what lower floors would buy)

Every XP number is a model prediction (I) with the calibration error of section 3 (about 10 percent) and the unverified floors of section 4 behind it. "Plan today" is the plan as intended (no bot failure modes); the logged hours reached 17.0k and 13.9k XP at 60 minutes against 19.5k simulated.

| mode | horizon | plan today (xp, clvl) | best found (xp, clvl) | gain | restarts in the best |
|---|---|---|---|---|---|
| plan | 60 min | 19511, clvl 6 | 20885, clvl 6 | +7% | 5 |
| plan | 120 min | 39696, clvl 8 | 53737, clvl 9 | +35% | 10 |
| plan | 240 min | 101704, clvl 12 | 175537, clvl 14 | +73% | 16 |
| strict | 60 min | 19511, clvl 6 | 37300, clvl 8 | +91% | 1 |
| strict | 120 min | 39696, clvl 8 | 91672, clvl 12 | +131% | 4 |
| strict | 240 min | 101704, clvl 12 | 232828, clvl 16 | +129% | 11 |
| bold | 60 min | 19511, clvl 6 | 44395, clvl 8 | +128% | 1 |
| bold | 120 min | 39696, clvl 8 | 102250, clvl 12 | +158% | 7 |
| bold | 240 min | 101704, clvl 12 | 248217, clvl 16 | +144% | 11 |

Minutes to reach a level on the best path (search horizon 480 minutes, same constraints; blank = not reached):

| mode | clvl 10 | clvl 15 | clvl 20 |
|---|---|---|---|
| plan | 121 | 236 | - |
| strict | 77 | 185 | - |
| bold | 64 | 170 | - |

### Findings (I unless marked)

1. **Under today's entry floors the first hour is nearly optimal in order and the gap is execution.** `plan` mode: 20.9k XP (clvl 6) against 19.5k for the plan at 60 minutes (+7 percent); the best first hour is Blood Moor to clvl 3, the Den at clvl 3 (the quest), then Cold Plains from clvl 4 with a game restart each time it is swept (5 restarts, 15 s each plus the walk back). The logged hours reached 17.0k and 13.9k: the distance from 19.5k to those is the bot's failure modes (the Den exit that did not fire, three Blood Moor re-sweeps, roaming), not the area order.
2. **From minute 60 the plan's row order costs 35 percent at 120 minutes and 73 percent at 240** (clvl 8 against 9, clvl 12 against 14). The plan enters Cave 1 at clvl 6 when Cold Plains is swept (385 XP per minute, 66 and 114 kills in two stretches) instead of starting a fresh game and re-clearing Cold Plains (about 600 per minute at clvl 6), keeps Stony Field and Dark Wood behind Burial Grounds and Cave 1, and never restarts while any row is open (the areamon finding B2). The optimum restarts whenever the best field is swept and a fresh game is cheaper than the next row (10 restarts in two hours).
3. **Time to level (plan mode, best path): clvl 10 at minute 121, clvl 15 at minute 236, clvl 20 not within 480 minutes.** With the `strict` floors: 77 and 185; with `bold`: 64 and 170 (clvl 20 is out of reach in all three: Act 1 Normal XP per kill falls below 150 at clvl 15 and later areas are outside the model).
4. **The floors dominate everything else.** At 120 minutes the optimum is 53.7k (`plan`), 91.7k (`strict`, clvl 12) and 102k (`bold`). `strict` sends a clvl 5 hero to Black Marsh and a clvl 7 hero to Tamoe Highland (about 1,000 to 1,200 XP per minute); whether that is survivable is exactly the drill X4 of areamon section 12.
5. **A restart is cheap and worth using: the measured cost is 15 s (V) plus the walk.** At 150 seconds the 120-minute optimum drops 17 percent, at 75 seconds 9 percent (section 6); the model's restart advantage vanishes only above about 4 minutes.
6. **Candidate rows for the plan v3 agent (mode `plan`, 120-minute best):** Blood Moor 1-3, Den 3-4 (quest, once), Cold Plains 4-7 with restarts, Stony Field 7-8 (enter 7 melee as today), Burial Grounds at 8 for Blood Raven (a 7 to 8 minute stop, 1,200 XP including her 181), Stony Field 8-9, Dark Wood 9-10 (today 9-13), then at 240 minutes Tamoe Highland 10-13 (no plan row today, 1,100 XP per minute), Black Marsh as a short stop, Catacombs 2-3 from 13 (as today). Cave 1, Cave 2, Crypt and Mausoleum never appear in the best path.

### Best strategy, mode `plan`

**60 minutes**: Blood Moor > Den of Evil > Blood Moor > Cold Plains (5 restarts). clvl by minute: 10:3 20:4 30:5 40:5 50:6 60:6

**120 minutes**: Blood Moor > Den of Evil > Blood Moor > Cold Plains > Stony Field > Burial Grounds > Stony Field > Dark Wood (10 restarts). clvl by minute: 10:3 20:4 30:5 40:5 50:6 60:6 70:7 80:7 90:8 100:8 110:9 120:9

**240 minutes**: Blood Moor > Den of Evil > Blood Moor > Cold Plains > Stony Field > Burial Grounds > Stony Field > Underground Pass. 2 > Stony Field > Dark Wood > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 2 > Catacombs 3 > Catacombs 2 (16 restarts). clvl by minute: 10:3 20:4 30:5 40:5 50:6 60:6 70:7 80:7 90:8 100:8 110:8 120:9 130:10 140:10 150:11 160:12 170:12 180:12 190:13 200:13 210:13 220:14 230:14 240:14

| area | clvl in -> out | minutes | kills | xp | restarts before |
|---|---|---|---|---|---|
| Blood Moor | 1 -> 3 | 0-8 | 69 | 1514 | 0 |
| Den of Evil | 3 -> 4 | 8-16 | 71 | 2626 | 0 |
| Blood Moor | 4 -> 4 | 16-17 | 4 | 95 | 0 |
| Cold Plains | 4 -> 7 | 17-65 | 640 | 18452 | 5 |
| Stony Field | 7 -> 8 | 65-84 | 191 | 10248 | 2 |
| Burial Grounds | 8 -> 8 | 84-91 | 29 | 1223 | 1 |
| Stony Field | 8 -> 8 | 91-101 | 115 | 6017 | 1 |
| Underground Pass. 2 | 8 -> 8 | 101-106 | 24 | 1914 | 0 |
| Stony Field | 8 -> 9 | 106-110 | 45 | 2342 | 1 |
| Dark Wood | 9 -> 10 | 110-126 | 238 | 13308 | 1 |
| Tamoe Highland | 10 -> 10 | 126-139 | 146 | 12243 | 0 |
| Black Marsh | 10 -> 11 | 139-142 | 37 | 2229 | 1 |
| Tamoe Highland | 11 -> 13 | 142-181 | 487 | 40607 | 2 |
| Catacombs 2 | 13 -> 14 | 181-213 | 153 | 28118 | 1 |
| Catacombs 3 | 14 -> 14 | 213-239 | 128 | 33973 | 1 |
| Catacombs 2 | 14 -> 14 | 239-240 | 3 | 628 | 0 |

Today's plan under the same model: 10:3 20:4 30:5 40:5 50:6 60:6 70:7 80:7 90:7 100:7 110:8 120:8 130:8 140:9 150:9 160:9 170:10 180:11 190:11 200:11 210:11 220:11 230:12 240:12. Entries below this mode's floors: none.

Ranked alternatives at 240 min (distinct area orders; the beam finds them within a fraction of a percent of each other, so the order inside a band is not sharp):

1. 175537 xp, clvl 14: Blood Moor > Den of Evil > Blood Moor > Cold Plains > Stony Field > Burial Grounds > Stony Field > Underground Pass. 2 > Stony Field > Dark Wood > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 2 > Catacombs 3 > Catacombs 2
2. 175519 xp, clvl 14: Blood Moor > Den of Evil > Blood Moor > Cold Plains > Stony Field > Underground Pass. 2 > Burial Grounds > Stony Field > Dark Wood > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 2 > Catacombs 3 > Catacombs 2
3. 175389 xp, clvl 14: Blood Moor > Den of Evil > Blood Moor > Cold Plains > Stony Field > Burial Grounds > Stony Field > Underground Pass. 2 > Stony Field > Dark Wood > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 2 > Catacombs 3 > Catacombs 4
4. 175384 xp, clvl 14: Blood Moor > Den of Evil > Cold Plains > Cave 1 > Cold Plains > Stony Field > Burial Grounds > Stony Field > Dark Wood > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 2 > Catacombs 3 > Catacombs 2
5. 175380 xp, clvl 14: Blood Moor > Den of Evil > Blood Moor > Cold Plains > Stony Field > Underground Pass. 2 > Burial Grounds > Stony Field > Dark Wood > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 2 > Catacombs 3 > Catacombs 4

### Best strategy, mode `strict`

**60 minutes**: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland (1 restarts). clvl by minute: 10:3 20:4 30:5 40:6 50:7 60:8

**120 minutes**: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Burial Grounds > Dark Wood > Black Marsh > Tamoe Highland (4 restarts). clvl by minute: 10:3 20:4 30:5 40:6 50:7 60:8 70:8 80:9 90:10 100:10 110:11 120:12

**240 minutes**: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Burial Grounds > Catacombs 3 > Catacombs 2 > Catacombs 3 (11 restarts). clvl by minute: 10:3 20:4 30:5 40:6 50:7 60:8 70:9 80:9 90:10 100:11 110:11 120:12 130:12 140:13 150:13 160:13 170:14 180:14 190:15 200:15 210:15 220:15 230:15 240:16

| area | clvl in -> out | minutes | kills | xp | restarts before |
|---|---|---|---|---|---|
| Blood Moor | 1 -> 2 | 0-3 | 24 | 520 | 0 |
| Cave 2 | 2 -> 3 | 3-9 | 23 | 1552 | 0 |
| Underground Pass. 2 | 3 -> 4 | 9-19 | 44 | 3678 | 0 |
| Den of Evil | 4 -> 5 | 19-28 | 71 | 2626 | 0 |
| Black Marsh | 5 -> 7 | 28-52 | 301 | 20158 | 1 |
| Tamoe Highland | 7 -> 12 | 52-112 | 733 | 61656 | 4 |
| Catacombs 3 | 12 -> 12 | 112-130 | 64 | 16993 | 0 |
| Catacombs 2 | 12 -> 13 | 130-139 | 48 | 8976 | 0 |
| Catacombs 3 | 13 -> 13 | 139-150 | 52 | 13820 | 1 |
| Catacombs 2 | 13 -> 14 | 150-160 | 60 | 11051 | 1 |
| Catacombs 3 | 14 -> 14 | 160-184 | 116 | 30804 | 1 |
| Catacombs 2 | 14 -> 15 | 184-188 | 24 | 4405 | 0 |
| Burial Grounds | 15 -> 15 | 188-195 | 29 | 345 | 1 |
| Catacombs 3 | 15 -> 15 | 195-206 | 52 | 13815 | 0 |
| Catacombs 2 | 15 -> 15 | 206-214 | 48 | 8802 | 1 |
| Catacombs 3 | 15 -> 16 | 214-240 | 127 | 33629 | 1 |

Today's plan under the same model: 10:3 20:4 30:5 40:5 50:6 60:6 70:7 80:7 90:7 100:7 110:8 120:8 130:8 140:9 150:9 160:9 170:10 180:11 190:11 200:11 210:11 220:11 230:12 240:12. Entries below this mode's floors: Den of Evil at clvl 3 (floor 4); Cold Plains at clvl 4 (floor 5).

Ranked alternatives at 240 min (distinct area orders; the beam finds them within a fraction of a percent of each other, so the order inside a band is not sharp):

1. 232828 xp, clvl 16: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Burial Grounds > Catacombs 3 > Catacombs 2 > Catacombs 3
2. 232722 xp, clvl 16: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 4 > Burial Grounds > Catacombs 3 > Catacombs 2 > Catacombs 3
3. 232635 xp, clvl 16: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3
4. 232421 xp, clvl 16: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds > Catacombs 3
5. 232264 xp, clvl 16: Blood Moor > Cave 2 > Underground Pass. 2 > Den of Evil > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2

### Best strategy, mode `bold`

**60 minutes**: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Dark Wood > Tamoe Highland > Pit 2 (1 restarts). clvl by minute: 10:3 20:4 30:6 40:7 50:8 60:8

**120 minutes**: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Burial Grounds > Tamoe Highland (7 restarts). clvl by minute: 10:3 20:5 30:6 40:7 50:8 60:8 70:9 80:10 90:10 100:11 110:12 120:12

**240 minutes**: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds > Catacombs 2 > Catacombs 3 (11 restarts). clvl by minute: 10:3 20:5 30:6 40:7 50:8 60:9 70:9 80:10 90:11 100:11 110:12 120:12 130:13 140:13 150:14 160:14 170:14 180:15 190:15 200:15 210:15 220:16 230:16 240:16

| area | clvl in -> out | minutes | kills | xp | restarts before |
|---|---|---|---|---|---|
| Cold Plains | 1 -> 2 | 0-3 | 17 | 506 | 0 |
| Stony Field | 2 -> 3 | 3-6 | 19 | 1032 | 0 |
| Den of Evil | 3 -> 4 | 6-15 | 71 | 2626 | 0 |
| Dark Wood | 4 -> 6 | 15-31 | 234 | 14055 | 1 |
| Black Marsh | 6 -> 7 | 31-37 | 67 | 4472 | 0 |
| Tamoe Highland | 7 -> 8 | 37-51 | 178 | 15042 | 0 |
| Black Marsh | 8 -> 9 | 51-58 | 100 | 6723 | 1 |
| Tamoe Highland | 9 -> 11 | 58-84 | 330 | 27773 | 1 |
| Catacombs 3 | 11 -> 11 | 84-102 | 64 | 16201 | 0 |
| Catacombs 2 | 11 -> 12 | 102-111 | 48 | 8972 | 0 |
| Catacombs 3 | 12 -> 13 | 111-134 | 110 | 29270 | 2 |
| Catacombs 2 | 13 -> 14 | 134-147 | 78 | 14338 | 1 |
| Catacombs 3 | 14 -> 14 | 147-171 | 116 | 30804 | 1 |
| Catacombs 2 | 14 -> 15 | 171-179 | 48 | 8835 | 0 |
| Catacombs 3 | 15 -> 15 | 179-190 | 52 | 13815 | 1 |
| Catacombs 2 | 15 -> 15 | 190-198 | 48 | 8802 | 0 |
| Catacombs 3 | 15 -> 15 | 198-209 | 52 | 13815 | 1 |
| Burial Grounds | 15 -> 15 | 209-216 | 29 | 345 | 0 |
| Catacombs 2 | 15 -> 16 | 216-219 | 15 | 2811 | 1 |
| Catacombs 3 | 16 -> 16 | 219-240 | 105 | 27981 | 1 |

Today's plan under the same model: 10:3 20:4 30:5 40:5 50:6 60:6 70:7 80:7 90:7 100:7 110:8 120:8 130:8 140:9 150:9 160:9 170:10 180:11 190:11 200:11 210:11 220:11 230:12 240:12. Entries below this mode's floors: none.

Ranked alternatives at 240 min (distinct area orders; the beam finds them within a fraction of a percent of each other, so the order inside a band is not sharp):

1. 248217 xp, clvl 16: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds > Catacombs 2 > Catacombs 3
2. 247580 xp, clvl 16: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds
3. 247515 xp, clvl 16: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Burial Grounds
4. 247475 xp, clvl 16: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 4 > Catacombs 3 > Burial Grounds > Catacombs 3
5. 247410 xp, clvl 16: Cold Plains > Stony Field > Den of Evil > Dark Wood > Black Marsh > Tamoe Highland > Black Marsh > Tamoe Highland > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Catacombs 3 > Catacombs 2 > Burial Grounds > Catacombs 3

## 6. Sensitivity (mode `plan`, beam width 40)

```
scenario                                           | opt@60   plan@60   gain | opt@120  plan@120  gain
baseline                                           | 20885    19511       +7% | 53738    39697      +35%
kill rate -20%                                     | 16743    15847       +6% | 40243    31325      +28%
kill rate +20%                                     | 25781    23123      +11% | 73454    45620      +61%
walking and chores -50% (walk, hop, town tax)      | 22069    20693       +7% | 59921    41689      +44%
walking and chores +100%                           | 18158    17081       +6% | 44380    33317      +33%
restart 30 s                                       | 20905    19435       +8% | 53128    39432      +35%
restart 75 s                                       | 20080    19202       +5% | 49187    38615      +27%
restart 150 s                                      | 19272    18809       +2% | 44844    37186      +21%
XP per kill -15% (fewer uniques)                   | 17456    17456       +0% | 41633    33425      +25%
XP per kill +15%                                   | 23547    22279       +6% | 67041    47883      +40%
supply decays (decay 1.5, floor 0.3, yield 0.85)   | 13020    12133       +7% | 32191    24739      +30%
Den may be farmed again                            | 20885    19109       +9% | 53738    39374      +36%
```



Reading the sensitivity table: the answer is stable in shape (Cold Plains with restarts, then Stony Field, Dark Wood, Tamoe, Catacombs 2-3) under every scenario but the gain over the plan moves from +21 to +61 percent at 120 minutes. The largest moves of the optimum at 120 minutes: supply decay (decay 1.5, floor 0.3, yield 0.85) minus 40 percent, kill rate +20 percent plus 37 percent and minus 20 percent minus 25 percent, XP per kill plus 25 or minus 22 percent, a 150-second restart minus 17 percent, halved walking and chores plus 12 percent. The supply-decay scenario is **rejected by the calibration** (run 1 error minus 35 percent, run 4 minus 16 percent at those parameters), so it is a bound, not a candidate; the plan loses the same share there (+30 percent gain stays). "XP per kill minus 15 percent" at 60 minutes makes the plan equal the beam (the optimiser result is seeded with the plan, so it never reports below it).

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
