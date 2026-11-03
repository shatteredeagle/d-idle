# Lockstep probe P3: unhooked control and k = 1, 2, 4 on the original game

> status: DONE for the planned runs (card `lockstepp3`, 2026-11-02). Continues docs/research/lockstep-p1p2.md. Result: the clock hook is clean and exact up to k=4 for the game's own frame counter (100 fps at k=4), the bot is usable at k=1, degraded at k=2 and not usable at k=4 without scaling the bot's own clocks. No hook source or addresses are in this repository (private hook repository, no remote).

## Authorisation, ledger, method

- User go 2026-11-02 ("run the lockstep work") for the hook set `d2-timehook proxy ijl11 IAT clock scaling` (instrumentation, ADR 0014). Throwaway character Booti on hidden instance i4. Run markers written with `d2cheat runstart`: `lockstep-p3-unhooked`, `lockstep-p3-k1`, `lockstep-p3-k2`, `lockstep-p3-k4`. No game memory was written by any run (reads only; the hook changes time, not state).
- Start protocol: the coordinator stopped, started and logged in the instance before every run (a restart per k; the hook is configured by a file next to the DLL, `offline=1`, `k=N`, `stat=10`). The original DLL was renamed, never written through the hardlinks; at the end it is restored (byte-identical) and the instance is stopped.
- Same bot arguments in all four runs: `d2bot --attach --goals 'farm:Blood Moor' --run-for 600s`. Frame rate: the game's frame counter (a static counter that matched 25 fps in P1; same static address in all four starts), read by `lockstepscan watch` every 5 s, started before the town phase and run through the bot phase.
- Town: character idle in Rogue Encampment, 10 min real (k=4: 5 min). Bot: 10 min REAL. Kills are `combat kill` lines of the bot log (including the repeat-collapsed ones), xp is the sum of the per-kill xp field (the level stayed 12 in every run), deaths are death events during the run (the bot prints its whole death history again at shutdown: those lines are not counted).
- Hook improvement (private repo, committed locally): a `STAT` line every 10 s real time (k, real and scaled elapsed ms, offset, per-API call counts, clamp count). It survives a kill: a stopped game leaves 130+ STAT lines per run (V), the exit `REHASH` is still never printed after `game.sh stop` (U whether a graceful exit prints it). The hook test suite (under Wine on the worker host) passes with the new STAT check: 145 checks, 0 failures.

## Results (V run, one run per row)

| | unhooked | k=1 | k=2 | k=4 |
|---|---|---|---|---|
| Hook log | none | `CLOCK k=1.0000`, 20 slots | `CLOCK k=2.0000`, same slot hash | `CLOCK k=4.0000`, same slot hash |
| Town frames per real s (ratio to 25 k) | 24.9992 (0.99997) | 24.9992 (0.99997) | 50.0 (0.99997) | 100.0 (1.00014, 5 min) |
| Hook STAT virt/real at the end | none | 1.0000 | 2.0000 | 4.0000 (offset grows as 3 x real) |
| Clamp events (nonzero wait forced to 1 ms) | none | 0 | 741 in 22 min | 670 in 18 min |
| Bot phase frames per real s / (25 k), stalls included | 0.99997 | 1.00003 | 0.986 | 0.988 |
| Bot: kills in 10 min real | 125 | 100 | 78 | 9 |
| Deaths | 0 | 0 | 0 (one chicken, save-and-exit at 21 percent life, re-entered by the bot) | 0 (two chicken events at 18 and 19 percent, one with save-and-exit and re-entry) |
| Kills per REAL minute | 12.5 | 10.0 | 7.8 | 0.9 |
| Kills per GAME minute (10 k game minutes) | 12.5 | 10.0 | 3.9 | 0.23 |
| Sum of per-kill xp | 1317 | 615 | 734 | 4 |
| `d2dsoak bands -bands 10-15` kills per hour (real) / xp per hour (real) | 802 / 11755 | 660 / 5211 | 493 / 5190 | 87 / 33 |
| Town chores failed | 0 | 0 | 8 (npc not found or not reachable, in the first 90 s of the run) | 16 (buying a potion: gold did not change) |
| Character at the end | alive, Rogue Encampment, no panel | same | same | same |

## What this shows

1. The hook is exact for the game's own clock: the town frame counter is 25 k fps to 0.015 percent for k = 1, 2, 4. 100 fps works (the frame gate sleeps keep up on the shared host; clamp events, each a one-millisecond floor, were 741 and 670 and did not change the ratio). No stall in the town phase of any run. V run.
2. The bot is NOT unaffected by k. The bot's own timers, input holds and waits are real time (original-game-lockstep.md section 2.6, now measured): k=1 control is inside the unhooked noise (100 against 125 kills; fights, route and chores differ run to run, the unhooked and k=1 runs even took different routes: U what the noise band is, one run each), k=2 gets 78 kills in the same real time (3.9 per game minute: the bot does half the work per game minute), k=4 is broken: 9 kills.
3. Why k=4 broke, from the bot log (V log, cause U):
   - Town chores. At k=2 eight `shop` chores failed with "npc not found or not reachable" in the first minute and then went through; at k=4 sixteen failed, each after about 18 s of real time, with "buying <potion>: gold did not change (no room, not enough gold, or the click missed)". Gold was 395, so it is not lack of gold, and there were no failures at k=1 or unhooked. The vendor window opened every time. Most likely mechanism (U): the bot clicks and then waits a fixed real time before it reads the result; the game's UI and walking run per game frame, so a fixed real wait covers k times more game time (people walk away, targets move) and the bot's hold and settle times are k times too long or too short in game time. Not proven; the fix is the scaled bot clock, then re-measure.
   - Field. The bot arrived in Cold Plains with low potions, chickened (life 18 to 19 percent twice, the second 4 s after arrival), then spent 1 min 50 s with one kill in Blood Moor. The frame counter stalled in windows of 5 to 10 s, and twice for 70 to 100 s in the bot phase at k=4 (25 windows below half rate, against 1 unhooked, 3 at k=1, 4 at k=2 plus one counter reset when the bot left and re-entered the game). The stalls appear in all runs, so they are not caused by the hook, but they get longer with k (U: area loads and menu transitions are real-time work, so their share of game time grows with k).
   - Bot timeouts that misfired under k (real-time constants): the 18 s purchase and chore timeouts above; the nav "stuck" and "stalled" recoveries (counts: 31, 16, 20, 17 for unhooked, 1, 2, 4: no k trend, not a clean signal); the 10 s input-flow waits at menu transitions. A systematic list needs the scaled clock.
4. Character state is NOT identical across runs (gold 312 to 466 at the start, potion stock drifts; Booti stays level 12 but is not reset). One run per row; differences of about 20 percent in kills (unhooked against k=1) are inside what this data can not separate from noise (U). The k=4 collapse (9 against 100 or more) and the k=2 halving per game minute are far outside it.
5. `d2dsoak bands` reads these bot logs (minute lines) and prints the per-band table above, but a verdict needs a baseline of the same bot over hours; the existing baselines are d2d-backend runs in virtual hours, 10-minute runs have a single open band. So the bands result is a table, not a pass or fail (U). Per game hour at k=2 the kill rate is half of the k=1 rate; at k=4 it is about 3 percent.

## What Tier A supports (conclusion)

- k=2 for the game: clean frame rate and no desync seen, no crash in either k=2 run (two runs, 5 and 10 min). The bot at k=2 is usable but yields half the work per game minute and chickened once: unfit for a soak or baseline measurement without the scaled bot clock.
- k=4: the game runs at 100 fps without trouble, but the bot is not usable (9 kills, chores failing). Do not use k above 2 for any measured run until the bot's own clocks are scaled.
- Not covered by the hook, still U for the real game: `GetSystemTimeAsFileTime`, waitable timers, delay-loaded imports, DirectDraw timing. The frame counter ratio of 0.99997 says the paths that matter for pacing are covered.

## Next steps

1. A scaled bot clock (rate read from the hook's `STAT`/cfg, or a bot knob `timeScale`) for waits, input holds and timeouts; then repeat k=2 and k=4 with the same arguments and require kills per game minute inside the k=1 band.
2. A noise band: three unhooked and three k=1 runs from a reset character state.
3. Graceful exit once to capture the exit `REHASH` line.
4. Cause of the long frame stalls in the bot phase (U): read what the counter does during area loads.

## State left

i4: original DLL restored (byte-identical to the saved copy), no proxy, no cfg, hook logs removed from the game folder (kept locally for this report), character Booti in Rogue Encampment, no panels, instance stopped by the coordinator. The hook repository has the STAT change as a local commit (no remote).
