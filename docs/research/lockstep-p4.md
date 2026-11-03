# Lockstep probe P4: k = 2 and k = 4 with the bot's own clock scaled

> status: DONE for the planned runs (card `lockstepp4`, 2026-11-02). Continues docs/research/lockstep-p3.md (table of k = 0 control, 1, 2, 4 with the bot clock NOT scaled). The scaled bot clock is `internal/gclock` (branch `agent/botclock`, docs/timescale.md). No hook source or addresses are in this repository (private hook repository).

## Setup

- User go 2026-11-02 for the hook set `d2-timehook proxy ijl11 IAT clock scaling` (instrumentation, ADR 0014). Throwaway Booti, hidden instance i4. Run markers `lockstep-p4-k2` and `lockstep-p4-k4` (`d2cheat runstart`); no game memory was written by any run.
- Same bot arguments as P3 (`d2bot --attach --goals 'farm:Blood Moor' --run-for 600s`, 10 min REAL) plus the scale. The bot binary is `agent/botclock` (45d50cf1) with ONE local harness patch that is not on that branch: `tools/live.sh` cannot forward environment variables to the bot, so the patch also accepts a leading `--timescale=K` argument and prints `gclock: timescale K` as the first log line (V: the line reads 2 and 4 in the two logs). `D2BOT_TIMESCALE=K` itself is the documented route and was not used. A second confirmation that the scale is live: the log's own `ms` fields and the `npc visit` timings run K times faster than the host stamps (V log).
- Coordinator restarts the game per k (stop, start, login); the proxy was installed with the original DLL renamed (never written through hardlinks) and restored at the end. Frame rate: the game's frame counter read every 5 s by `lockstepscan watch`, same static counter as P3.
- Host load was shared (Gate 2 at nice 19 and other lanes). One run per row (U noise band, see P3).

## Results (V run)

P3 columns are copied from docs/research/lockstep-p3.md; kills there are `combat kill` lines including the repeat-collapsed ones, so the P4 row "combat kill incl. repeats" is the comparable one.

| | P3 unhooked | P3 k=1 | P3 k=2 unscaled | P3 k=4 unscaled | P4 k=2 scaled | P4 k=4 scaled |
|---|---|---|---|---|---|---|
| Frames per real second, steady windows | 25.0 | 25.0 | 50.0 | 100.0 | 49.96 | 99.5 to 100.1 (town 5 min: 99.99) |
| Kills, `combat kill` lines incl. repeats, 10 min real | 125 | 100 | 78 | 9 | 154 | 271 |
| Kills, observer counter (`killtime` lines, one per monster death seen in snapshots) | n/a | n/a | n/a | n/a | 142 | 204 |
| Kills per real minute (combat kill / observer) | 12.5 | 10.0 | 7.8 | 0.9 | 15.4 / 14.2 | 27.1 / 20.4 |
| Kills per GAME minute (10 k game minutes; combat kill / observer) | 12.5 | 10.0 | 3.9 | 0.23 | 7.7 / 7.1 | 6.8 / 5.1 |
| Sum of per-kill xp | 1317 | 615 | 734 | 4 | 2174 | 1442 |
| Deaths | 0 | 0 | 0 | 0 | 0 | 0 |
| Chicken events (life under 25 percent) | n/a | n/a | 1 | 2 | 2 (one save-and-exit, re-entered) | 2 (two save-and-exits, re-entered) |
| Shop chores failed ("gold did not change") | 0 | 0 | 0 | 16 | 0 | 0 |
| Chores "npc not found or not reachable" | 0 | 0 | 8 | 0 | 0 | 0 |
| `nav stuck` lines | 31 | 16 | 20 | 17 | 9 | 44 |
| `nav stalled` lines | n/a | n/a | n/a | n/a | n/a | 29 |
| Frame counter resets (game re-entry) | n/a | n/a | 1 | n/a | 1 | 2 |
| Character at the end | alive, town | same | same | same | alive, Rogue Encampment, no panel | same |

P3's nav counts (31, 16, 20, 17 for unhooked, k=1, 2, 4) are its "stuck and stalled recoveries" total, not the `nav stuck` line alone; the two P4 `nav stuck` counts and the k=4 `nav stalled` count are therefore only roughly comparable (U). Hook STAT lines: `k=2.0000` and `k=4.0000`, virt/real exactly K; clamp counts (cumulative since the game start, about 20 min) 647 and 2491.

### Which kill count is right (V logs, cause U)

Two different counters. The observer counter (`killtime` lines and the minute line's `killsGame`) counts a monster death once when the bot's snapshot sees the monster go; `killsGame` resets when the game is re-entered, so the last minute line (142 and 197) misses kills before a re-entry at k=4 (204 `killtime` lines in total). The `combat kill` event is emitted by the combat/observe path per kill report and, with its repeat-collapsed lines, gives 154 and 271. They agree to within 8 percent at k=2 and differ by 33 percent at k=4. Likely (U, not proven): at k=4 the game runs 100 frames per real second and a monster can die and vanish between two snapshots, so the observer undercounts, while the event path can report one monster more than once; the truth lies between 204 and 271 and neither is independent ground truth. The per-kill xp sum (1442 against 2174) does not settle it because xp is split evenly over the kills of a snapshot. Use the observer counter as the conservative figure and the combat-kill figure for comparison with P3.

## Reading

1. The claim holds for the chores: scaling the bot's game-clock waits removes the P3 failures. At k=4 the 16 "gold did not change" purchase failures became 0; at k=2 the 8 "npc not found" became 0. This confirms the P3 hypothesis (fixed real waits spanning k times more game time).
2. Kill rate per real minute is no longer reduced: k=2 14 to 15 and k=4 20 to 27, against 10.0 at k=1 and 12.5 unhooked (P3). Per GAME minute it is 7.1 to 7.7 (k=2) and 5.1 to 6.8 (k=4) against 10.0 (k=1): the bot recovers 70 to 77 percent of the k=1 yield per game minute at k=2 and 50 to 68 percent at k=4, not all of it. (U whether that is noise: one run each, P3 saw 100 against 125 for k=1 against unhooked.)
3. The k=4 run was not a clean Blood Moor farm: the area field of the minute lines shows the bot in the Encampment, Blood Moor, Cold Plains, the Cave, the Den, the Tristram road and others (leveler and goal switching, "leveling target not reachable" 7, "portal did not lead to the level" 5, "area skipped" 7), with 44 `nav stuck` and 29 `nav stalled`. Booti's belt was empty of healing potions at both chickens (22 to 24 percent life). The nav trouble has two candidate causes (U): k itself (the world moves k times further per bot decision and the d2input holds are real time) and the empty belt that sends the bot through town and portals. The 9 `nav stuck` at k=2 against 44 at k=4 points at k, but the character state also drifted between the runs.
4. Frame rate: steady windows are 25 k to 0.5 percent (k=2 49.96; k=4 99.5 to 100.1). The counter resets when the bot saves and exits and re-enters (a watch total over the run is therefore meaningless: the tool printed 18 and 20 fps totals); stalls: one window under 40 fps at k=2 (10 s around the re-entry), at k=4 the windows at the two re-entries.

## What remains (U, not measured here)

- The d2input helper holds, sleeps and key `hold=` strings run in real time inside a sequence (docs/timescale.md): not scaled, and a hold under one game frame (10 ms at k=4) may be dropped. Nothing in the logs shows a failed hold, but nothing proves they did not shorten effects.
- Farm duration and goal ActiveSeconds are real time by design: `--run-for` and the farm duration count real seconds, so a measured run at k yields k times more game time per real second (this is the point of the speed-up, but kills per game minute is the quality measure).
- Area loads and menu transitions are real-time work: the share of game time grows with k (the two re-entries cost 10 s of frames each at k=4).
- Chickens and an empty belt dominate the field phase on this character; a fair soak needs a character reset (P3 next step 2) and a noise band of three runs per k.
- A graceful-exit REHASH was not captured again (U).

## Verdict

k=2 and k=4 with the scaled bot clock are usable for chore and mechanics testing: shop and walking chores no longer fail and the real-time kill rate is at or above the unhooked rate. They are NOT yet cleared for a measured baseline or soak. Open (U): single runs, no noise band (three unhooked, three k=1, three per k from a reset character), a kill count that depends on which counter is used (204 against 271 at k=4), a per-game-minute yield 25 to 50 percent below k=1 at k=4, 44 nav-stuck lines at k=4 with an unexplained mix of k and empty-belt effects, unscaled d2input holds, and bands/gates that were calibrated at k=1 in real time.

## Next steps

1. Reset the character state (gold, potions, level) before each run and take a noise band: 3 unhooked, 3 k=1, 3 k=2, 3 k=4, same arguments.
2. Count kills independently of the bot (diff of the game's monster list per frame, read-only) to decide which counter is right at k=4.
3. Per-site check of the d2input holds and sleeps at k=4 (scale them or floor them at one frame), then repeat k=4 and compare `nav stuck` against k=1 from a full belt.
4. Pass the scale to the bot through a documented flag or live.sh environment forwarding, so the local harness patch is not needed.

## State left

i4: original DLL restored (byte-identical to the saved copy), no proxy, no cfg, hook logs moved out of the game folder (kept on the worker host); Booti alive in Rogue Encampment, no panels; instance stopped by the coordinator.
