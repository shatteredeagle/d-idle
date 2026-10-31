# KILLFIX: removing the two top kill-gap causes on d2d (2026-10-30)

> status: current. Tags: V = measured on hidden d2d (fresh uncheated Sorceress, 2 virtual hours per run, parallel slot runs, `tools/d2dloop/pm.sh`), U = inferred. Nothing was run on Game.exe. Follows docs/killgap.md.

## Method (V)

`tools/d2dloop/kfbuild.sh BIN` builds the worktree on the worker host, `pm.sh par PREFIX BIN 2 SEED...` runs one managed d2d per seed inside a slot of the pool (docs/process/worker-compute.md), all seeds side by side. Numbers are kills and XP at the last `minute` line divided by the 2 virtual hours. Runs are not deterministic (same seed, different game), the same binary gave 169 and 159 kills/h in two 4-seed rounds, so a difference under about +-10 % is noise. The load differed between rounds (other agents use the slots), which also moves the wall-clock-based timers of the bot.

## What the logs show (V, seeds 1-4, baseline binary)

- The cave shuttle is a fight, not a walk: a Fallen pack stands at the Den mouth, the combat approach (`approaches=12`, `ignoring target ... approaches`) walks to a point at the mouth, d2d fires the warp (8 ms between the approach and `Entered Den of Evil`), the bounce puts the character inside the trigger, nav steps 20 subtiles away, the fight walks back (log of seed 4, k3 round). `penalizeExits` leaves exits near the walk's goal alone, so a goal at the mouth gets no penalty.
- In Cave Level 1 (target of poverty farming) arrival is inside the up warp; `mouthStep` found no point 20 subtiles straight away (corridor walls, `nearestWalkable` within 6 failed) and returned nothing, so the first roam leg fired the warp again (40+ crossings in one 1063 s gap).
- The empty-area gaps (`clear killed nothing`, roam coverage 36 % of 1781 points, 23 kills in 43 minutes) occur in an area that still has monsters: the roam is long because Blood Moor has 1800 coverage points, not because it is empty.
- d2d moves at run speed (V, probe in a slot: 1.88 tiles per 25 frames = 9.4 subtiles/s with `cmd move`), so walking speed is not the lever; walking distance is.

## Changes

| change | where | gate |
|---|---|---|
| `Config.EmptyAreaSec` (default 0 = off): a sweep with no kill for that long in a non-quest area calls it cleared (planner: next area or fresh game). The check runs inside the roam (Explore done callback), not between roams. | internal/bot/goal.go, runner.go, config.go | knob, off by default |
| `nav.mouthStep` flood fallback: nearest cell within 40 hops at least 8 subtiles farther from the warp when the straight line is blocked | internal/nav/walk.go | none (only runs next to an unwanted exit; applies to Game.exe, needs a G1/G2 check) |
| `nav.NearMouth` + `Fighter.move` skip: combat moves to a point within 9 subtiles of an unwanted exit are skipped (the monsters come) | internal/nav/walk.go, nav.go, internal/combat/fight.go; `SetMouthGuard(cfg.MouthGuard)` in cmd/d2bot/d2dwire.go | `Config.MouthGuard`, d2d only, off by default |

## Results

Same seeds 1-4, 2 virtual hours, per round mean of 4 runs (V). Baseline = binary kg2 (master plus the killtime instrument), two rounds (`ba`, `bb`); variants are cumulative.

| binary | kills/h | XP/h | s/kill (walk+cast+wait) | unwanted border crossings per run |
|---|---|---|---|---|
| baseline, 8 runs (169 / 159 per round) | 164 | 3830 | 22.1 | 28.5 |
| k1: empty-area leave 150 s, check between roams (never fired) | 152 | 3459 | 23.8 | 20.8 |
| k2: + mouthStep flood | 151 | 3455 | 23.5 | 23.5 |
| k3: + check inside the roam, 150 s (16 areas abandoned) | 171 | 3913 | 21.6 | 31.2 |
| k4: same with 300 s (never fired) | 146 | 3507 | not computed | 33.5 |
| k5/k6: + combat mouth guard, 300 s, 8 runs | 154 | 3480 | 23.3 | 20.8 |

Reading: no change moved kills/h or XP/h outside the run noise (4-run means of one binary differ by 6 %, one seed round of k3 by +4 %, k5 by -6 %). Seconds per kill did not fall. The mouth guard cut the border crossings by 27 % (28.5 to 20.8, 8 runs each) and the flood step is exercised only next to an unwanted exit; both leave the kill rate where it was, so the crossings are not what limits kills/h.
Gap seconds by step (share of all gap seconds, baseline vs k5, 4 runs each): `Recovery: farming gold` 58-69 % vs 56-71 %, `Quest: fighting` 18-28 % vs 20-38 %, `Returning to the field` 0-21 % vs 2-12 %; the over-120 s gaps stayed 12-13 kills per run at 8-15 s per kill.
EmptyAreaSec 150 abandoned areas at 36 % roam coverage with the area still holding monsters, then paid a walk to town (no scroll) and a restart (five `d2d already running` failures each); 300 s never fired in 8 runs, because the long gaps are quest areas (exempt on purpose), town trips and fights, not silent roaming of a dead area.

## Verdict and defaults

All three changes ship OFF or inert: `emptyAreaSec` 0 and `mouthGuard` false (bot.Config keys, `config.json`), the flood step is always on but only acts where `mouthStep` used to return nothing. The 20 longest gaps of KILLGAP are real but the seconds per kill are spread over thousands of ordinary 10-120 s gaps (31 % of kills, about 11 of 22 s): walking between sparse monsters, and (U) the poverty walks to town without a town portal. Next candidates: the no-scroll town trips ("Returning to the field", `no town portal: walking back`), and the 1800-point roam order (nearest-first coverage visits about 36 % of a Blood Moor before a pack is found).


## Other findings

- A recovered panic `index out of range [0] with length 0` in `nav/astar.go:53 pop` (1 of 24 runs): two goroutines use one `Planner` (combat `PathTo` while a walk searches). Not fixed here (internal/nav planner, shared); a mutex in `Planner.search` or one planner per caller would do.
- Restarts show `d2d already running` x5 after `Save and Exit` (failsafe ladder step 1) in every restart: the leave/re-enter path races the old process exit.
