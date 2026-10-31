# d2d run management (change runfix)

> status: policy implemented and unit-tested; stamina model V live; the before/after A/B is NOT finished (see the last section).

The d2d bot used to walk everywhere: d2d's run toggle starts off and nothing sent it. The original character runs while stamina lasts. Engine speeds are identical (DECOMPREC, V by emulator oracle: walk 0.375 subtiles/tick, run 0.5625, ratio 1.5), so running is a 1.5x travel gain.

## Stamina model on d2d (V live, fresh uncheated Sorceress clvl 1, lab5, lockstep, probe `~/d2d-lab/runfix-probe.py`)

| fact | measured | source in the fork (S) |
|---|---|---|
| max stamina | 74 (state prints 8.8 fixed, 18944/256) | |
| drain while running outside town | 7.8 per 2 s, 3.9/s, 74 to 0 in about 19 s | `rules::stamina_drain`, `World::stamina_frame` (no drain in town) |
| walking | no drain, stays at max | |
| forced walk at 0 | run toggle stays on, character walks | `run = ... && stamina > 0` |
| regen standing | 0 to 18.2 in 2 s (about 7.2/s, full in about 10 s) | max >> 8 per tick |
| regen walking | only at stamina >= 1.0, max >> 9 per tick (about 3.6/s); at < 1.0 a walk never regenerates | `stamina_regen` |
| observed speed | walk 1.85 cells/s along a path, run about 2.8 cells/s (cam units, coarse: turns and stalls included); ratio about 1.5, matches DECOMPREC | |

Consequences: standing (fights, casts, waits) refills the bar in about 10 s, so a bot that fights between legs runs almost all of the time; a continuous walk gets about 19 s of running, then the toggle must be dropped BEFORE the bar hits 0, otherwise the walk is locked at 0 with no regeneration until the character stands still.

`debug players` pmode is -1 both walking and running (V live), and `state` does not report the toggle, so the controller keeps a belief (reset when a game starts).

## Control verb

There is no `cmd run` on the devctl channel (the fork parses `cmd::Run` only from the keyboard). The toggle is the key `r` (fork `kToggleRun` = 'R', S; V live: `key r` made the character drain stamina). A fork verb `cmd run 0|1` would remove the belief (not done: no fork change needed).

## Policy (internal/d2dinput/run.go)

Before every move command: read stamina; run off below 8 % of max, on again from 25 % (RunPolicy On/Off; env `D2D_RUN_ON`, `D2D_RUN_OFF` fractions; `D2D_RUN=off` restores the walk-only bot). Wired in `d2denv` only; the Game.exe backend is untouched. Stamina is a free read of the existing `state` line.

## Before/after (hidden d2d, 4 seeds x 2 virtual hours, same binary, `D2D_RUN=off` vs default)

NOT MEASURED. Two attempts:

1. On master (17fffea) all 8 queued runs, both arms, entered the same livelock at clvl 3 after about 35 virtual minutes: `leveling: everything worth farming is cleared, starting a fresh game` hundreds of times in a minute (`freshGameToLevel`, internal/bot/task.go: `r.loop.cleared` is not reset by the new game, so the next tick starts another one). Not related to running (the walk-only arm does it too). The killfix branch (empty-area default 300 s) does not reach it in the same time, so it only shows on master.
2. Re-run on a scratch tree = agent/killfix 2ea1c6e + the runfix diff (binary `~/d2d-lab/loop/d2bot.rf3`, script `~/d2d-lab/loop/rf.sh ARM SEED`, arm `off` sets `D2D_RUN=off`): started, but a mistake of mine left the first attempt's bots running and sharing the log file names, and the agent was not allowed to kill them, so these logs are contaminated and no numbers are quoted.

To redo: `~/d2d-lab/loop/rf.sh run|off SEED` inside `~/d2d-lab/d2dslot.sh run -w 3000 --`, 40 min wall, then compare the `killtime` lines (walk bucket), `minute` kills and xp.
