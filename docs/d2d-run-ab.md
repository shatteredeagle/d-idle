# Livelock check and run A/B on d2d (change runab, 2026-10-30)

Tags: V = verified live (hidden d2d lab6, lockstep, fresh uncheated Sorceress "Loopy-ab", 2 virtual hours per run, `nice -n 10`, own slots via d2dslot).
Binary: integration of agent/livelockfix 5f864c2 + agent/runfix 9923e7c + one extra fix (below), built as `d2bot.runab`.
Runner: `tools/runab/one.sh TAG ARM SEED HOURS` (pid file = process group leader, stop with `kill -TERM -- -PID`), `tools/runab/ab.sh` (interleaved, at most 3 at once), `tools/runab/an.py` (per-run metrics, mean +- SE, Welch t-test).

## 1. Livelock check

- livelockfix 5f864c2 alone: **FAILED (V)**. At clvl 3 (about 35 virtual minutes) `leveling: everything worth farming is cleared, starting a fresh game` every ~10 s wall, 38 Save and Exits in 1.5 wall minutes, `restarts:0` in the minute lines. Cause: on d2d the restart quits and restarts the process without a menu poll, so `ob.newGames` never moves and `syncGames` never forgets the cleared set (the unit test faked the observer increment).
- Extra fix in this branch (internal/bot/task.go, `freshGameToLevel`): after a successful `Cycler.NewGame` reset `r.loop.cleared` directly. Not in livelockfix; the owner of internal/bot should take it.
- With it (run L2, seed 31): **PASS (V)**: 1 fresh-game line, clvl 3 at 30 virtual min, 4 at 1 h, 5 at 2 h (xp 9214, 399 kills, no death). The A/B runs: 1 to 2 fresh-game lines each.

## 2. A/B, D2D_RUN=off (walk only) vs default (run policy on)

Seeds 41, 43, 44, 45 (same seeds both arms, interleaved start order, up to 3 slots at once; runs are not deterministic, see d2d-loop.md). Seed 42 was dropped, see 3.
Per run (kills and xp from the last minute line, s/kill = mean of the `killtime` lines of the run, virtual seconds):

| run | arm | clvl | kills (/h) | xp/h | s/kill total | walk | cast | wait |
|---|---|---|---|---|---|---|---|---|
| 41 | off | 5 | 337 (168) | 4614 | 21.4 | 16.7 | 0.9 | 3.8 |
| 41 | on | 5 | 452 (226) | 6065 | 15.6 | 11.3 | 0.9 | 3.4 |
| 43 | off | 4 | 283 (142) | 3520 | 25.4 | 20.5 | 1.0 | 3.9 |
| 43 | on | 5 | 337 (168) | 4032 | 20.2 | 14.6 | 0.9 | 4.8 |
| 44 | off | 5 | 342 (171) | 4322 | 20.8 | 16.9 | 0.9 | 3.0 |
| 44 | on | 4 | 267 (134) | 3179 | 26.2 | 19.5 | 1.2 | 5.4 |
| 45 | off | 5 | 354 (177) | 4740 | 20.6 | 16.3 | 0.8 | 3.6 |
| 45 | on | 5 | 397 (198) | 5092 | 17.5 | 13.0 | 0.9 | 3.6 |

Mean +- SE over 4 seeds (Welch t, two-sided p):

| metric | off | on | p |
|---|---|---|---|
| s/kill walk bucket | 17.6 +- 1.0 | 14.6 +- 1.8 (-17 %) | 0.21 |
| s/kill total | 22.1 +- 1.1 | 19.9 +- 2.3 | 0.44 |
| kills/h | 164.5 +- 7.9 | 181.6 +- 19.9 (+10 %) | 0.47 |
| XP/h | 4299 +- 274 | 4592 +- 628 (+7 %) | 0.69 |

Reading: the direction matches the model (walk bucket down 17 %, kills/h and XP/h up 7 to 10 %), but 4 seeds do not separate the arms (all p > 0.2; one seed, 44, went the other way). Not significant: **running is not shown to help or to hurt; it is not a regression.** An extra on-arm run (seed 42, 457 kills, 4940 XP/h, walk 11.0) is not in the table because its off partner stalled.

`d2dsoak compare -gate v2 -hours 2 -baselines tools/d2dsoak/baselines-early.json` on the 4 on-arm logs: XP/h 4523 vs 6600 (-31.5 %, tolerance 1320): INCONCLUSIVE (too few seeds); deaths/h PASS (0 deaths in 8 h); the other metrics INCONCLUSIVE (not in the minute line); **OVERALL INCONCLUSIVE** (4 seeds under the minimum 5, 2 h under 3 h). Indicative only: still about a third below the early baseline.

## 3. New stall found: seed 42, walk-only arm (V, reproduced twice)

Seed 42 off (also a rerun) stops advancing at virtual 90 s, clvl 1, 6 kills: `combat pinned` / `combat crowd: stepping away` (crowd 9, life 32 %) / `combat move failed ... nav: no path: empty path` forever at 10 Hz, the pace line shows `steps 869, ticks 2249` constant (identical in both runs), so the combat loop never steps the lockstep game. Logs: `~/d2d-lab/runab/stall-S2off.log`, `stall-S2offb.log` on the worker host. The on arm with the same seed passed (different path). Owner: combat; both runs were stopped by me after 8 wall minutes.

## Not done / caveats

- 2 virtual hours per run, clvl 4 to 5: this is the opening, where the walk bucket is largest; a longer run could differ.
- Seeds are not exactly comparable between arms (non-deterministic loop); 4 pairs is a small sample.
