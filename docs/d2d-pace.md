# Time and progress on d2d (agent D2DPACE, 2026-10-30)

Tags: V = offline test named or live log, S = secondary, U = unverified.

## Time sources under lockstep

| Source | Clock | Note |
|---|---|---|
| Runner: minute ticker, no-progress detector (`bumpReal`/`rearm`), ladder timers, death log, state timers | `env.Clock` = `d2denv.LateClock`: the `Pacer` (virtual, advances only by `step`) when `clock` says lockstep, else wall | V by reading `internal/bot` (every use is `r.e.Clock`) |
| Combat, nav, town sleeps | same clock (the Rig/Town envs are copies sharing it) | V |
| Quest source cache (200 ms) | now the env clock (was wall: a cached quest dump outlived many stepped seconds) | V `d2denv.Build` sets `Now: clk.Now` |
| Goal manager (1 s tick, event stamps, `activeSeconds`) | wall, by design (`goals.Options.Clock` default) | its totals are wall seconds; the runner's are virtual |
| Heartbeat/health | wall beat under lockstep (counters only move when stepped), counters when free-running | V `TestD2DHealthLockstepNeverHangs` |
| Watchdog on `Runner.LastStep` | wall, by design: a stall of the bot loop itself | |
| Soak runner hours (`-clock bot`) | `$D2DSOAK_HOURS` now stops the bot after that many VIRTUAL hours (`stopAtVirtual`, `soakHours`) | V `TestStopAtVirtualUsesTheEnvClock`; `--run-for` stays wall |

Fixes in the `Pacer` (`internal/d2dinput/pacer.go`):
- Virtual time went BACKWARDS when the failsafe ladder restarted d2d: game_ms counts from 0 again and `cur = game_ms - start` dropped by the whole previous game. Every no-progress timer then read negative and could never fire. `follow` carries the earlier virtual time across a game_ms reset (V `TestPacerFollowSurvivesGameRestart`). A failed `step` now also feeds the carry so it cannot be undone by the next reading.
- The default `Base` was the zero `time.Time` (year 1, as the comment said Unix epoch): virtual time 0 read as `IsZero`, i.e. "unset", to every sentinel check. Now Unix epoch (V `TestPacerNowIsNotZeroTime`).

## Why the summary said 0 kills, 0 deaths

The minute line counts from the runner's own observer (`killsGame`, `deaths`), the final `summary` line from the goal manager's `Stats`, which only sees events that reach the runner's channel. `d2dcombat.Attach` and the town/loot envs copy the `env.Env` by value BEFORE `bot.New` installs `Env.Emit`, so combat's `EvKill`/`EvPotion` and loot's drops were emitted into a nil func (combat logged `combat kill` lines, the manager saw nothing: also no kills towards kill goals, no xp/h). `runD2D` now installs a `lateEmit` forwarder before the stack is built and binds it to the runner's emitter after `bot.New` (V `TestLateEmitForwardsAfterBind`; live: see below). On Game.exe the behaviors hold the env POINTER, so that path was not affected (S, read only).
Minute-line `deaths` came from the same observer and read 0 only because no death had happened yet in those logs (U: not contradicted).

## Live sessions
Own d2d: fork binary `d2d.lab2`, `--lockstep`, `--no-save`, seed 7, throwaway Sorceress "Pacey" (fresh start, no debug verbs needed), `nice 10`, lock held only while the bot runs (`~/d2d-lab/fork/pace-run/run.sh N`, 9 min `--run-for`), load average about 25. Three sessions, clvl 1 in all of them.

| run | virtual / wall | kills (combat) | deaths | what ended it |
|---|---|---|---|---|
| 1 | about 2.5 virtual min in 9 wall min (pace log not yet there) | 3, summary now 3 (was 0) | 0 | 3 minutes with no ping answered, 50 s grace passed, "exited", game restarted, 86 xp lost |
| 2 | ratio 0.5 (`pace` line: 785 ticks per wall minute, 5 steps/s, 2.5 ticks per step) | 3 | 0 | `ToTown` -> "hand: panel did not open: ui flag 0x1 never became true" x5 per rung, ladder rungs 1..3 in 3 virtual min, game restarted |
| 3 | ratio 0.5, then 76 to 91 after the d2d socket vanished | 7 (13 lifetime) | 0 | see below |
| 4 (after the Pacer wall-wait and `d2dTown`) | ratio 0.41 (223 virtual s in 540 wall s) | 13, 9 potions drunk, xp 304 | 0 | no "ui flag" errors any more, no failsafe storm; 9 wall min ended at clvl 1 (clvl 2 needs 500 xp), gold 0 |

### Blockers to leveling (in order of how often they hit)
1. **Town portal opening needs the inventory flag.** `town.Town.openPortal` -> `hand.OpenInventory` waits for `UIInventory`, which `dump ui` never reports (`d2dsrc/player.go` lists it unknown). Every `ToTown` with a scroll failed 5 times, the ladder restarted the game and the character lost its xp (`xpLost 86` and `154`). Fixed in the wiring: `d2dTown.ToTown` opens the portal with combat's `d2dPortal` first (it uses `i`, a fixed wait and `debug itemat`) and then lets the adapter walk into the open portal. V live only up to "the ToTown path no longer hits the panel flag"; whether the portal then opens in the Den with 17 hostiles around is U (run 3 ended in the next item). The same missing flag also blocks every other `OpenInventory` user (equip, belt buy, stash, leveler item moves): **D2DTOWN2 should either get d2d to report an inventory flag in `dump ui` or track it in the Controller (toggle on `i`, clear on Escape/CloseAll) and set `UI[UIInventory]` in `d2dsrc`**.
2. **A clvl 1 Sorceress in the Den of Evil.** The planner sends clvl 1 to the Den ("den: floor waived, nothing else to level in"; Blood Moor is "cleared" in about 3 s after a restart because `area cleared` memory survives). 17 to 19 hostiles are around the entry, 4 minor healing potions are drunk in 12 s (life 40, potions at 17 to 22 life), the "emergency town portal" then starts at life 15. 6 to 7 kills before that, 0 xp per kill is shown for some Fallen (xp split), clvl 2 needs 500 xp. Poverty mode (gold 0, no pickup because autoloot is off and D2DLOOT's pickup is not wired) means no potion refill, so every trip ends at the same point. D2DLOOT/D2DTOWN2/planner: gold and potions are the real gate; the Den should wait for clvl 2 to 3 or for a belt of 4 potions plus mana.
3. **Mana**: at clvl 1 the Sorceress has 26 to 35 mana; Fire Bolt (1 pt) used about 3 mana per cast and `castsPerSec` is 1.8 to 2.1, so the mana is gone after 10 to 15 casts (`mana 32 -> 9` inside 4 s in run 3). The combat profile `sorceress/lightning` has Fire Bolt first at clvl 1 and the staff melee fallback of `manabackoff` (merged by another agent, 4d79b0d) only helps once mana is empty. The Game.exe bot has the same table (docs/earlygame.md section 1, "Fire Bolt / CB" at clvl 1): the difference on d2d is not the policy but the absence of gold, so no mana potions and no belt refill (mana potion count in the belt 0 in every minute line). Not changed.
4. **Virtual clock races when the channel dies** (fixed, V `TestPacer*` and live evidence): in run 3 the d2d socket vanished at 12:02:39 (`dial ... no such file`, d2d printed no crash line and the managed process was still listed; cause U, the old and the restarted d2d log into one d2d.out). Every failed `step` then advanced virtual time without waiting: 27400 virtual seconds (7.6 h) in 5 wall minutes, 339000 failed steps, rungs 1 to 5, "Paused" and "recovery: end (time cap 45 min)" inside one second. The Pacer now also waits the same time in wall clock after a failed step. A failed restart of a managed d2d whose socket is gone but whose process lives (the bot cannot `quit` it) is not handled: `Proc.Stop` kills by pid, but `--clean-start`-style recovery was not exercised (U).
5. **Pace** (see "Speed" below, D2DSPEED): lockstep under load 25 runs at 0.5 virtual seconds per wall second while the bot is busy (5 steps per wall second at 2.5 ticks: each bot tick reads a full snapshot). The d2d itself does 100 to 130 frames/s idle and 2700 frames/s when stepped in big chunks (d2d.out `13714 frames/5s`). So virtual-hour targets (soak) cost twice their length in wall time and the minute line appears every 2 wall minutes. A cheaper snapshot per tick (or a larger step per sleep in the walking loops) would raise it; not done here.

## Open
- Whether `d2dPortal` can open a portal at all in the Den under attack (it was the next thing to be exercised).
- Socket vanishing / d2d wedge (runs 1 and 3 both lost the channel in the Den within 10 minutes): needs the d2d.out of the first instance (the managed restart appends to the same file; rotate it per start) to be kept per instance.

## Speed (agent D2DSPEED, 2026-10-30)

Tags: V = measured live (trace table of `D2DCTL_TRACE`) or offline test, U = unverified. Same session setup as D2DPACE (own d2d, `--lockstep`, `--no-save`, seed 7, throwaway Sorceress, nice 10, load 6 to 17), runs in `~/d2d-lab/fork/speed-run` on the worker host (`run.sh TAG BIN RUNFOR`, `D2DBIN` picks the d2d).

### Profile (V)
- Every devctl verb costs ONE d2d frame, about 16.6 ms, whatever it does (`state`, `monsters`, `cmd move`: all 16.5 to 17.7 ms, none under 15). `Channel::pump` runs once per frame and idle lockstep frames were paced at the 60 fps budget. A snapshot is 8 verbs (`state`, `debug players`, `dump` head, `dump items`, `monsters`, `npcs`, `ground`, `debug portals`) = about 133 ms, plus one `step` (36 ms for 2.5 ticks). Baseline run: 995 steps and 1972 ticks in 180 wall s, ratio 0.6; the verbs took 100 % of the wall time (step 29 s + 7 reads x about 14 s per 820 snapshots).
- After pipelining the read the verbs were still 17 ms each (`state` is the first line of the batch and pays the frame): the frame itself was slow because every idle frame was DRAWN (d2d.out `render avg=9 present avg=4`, software renderer, 10 to 15 ms).

### Fixes
1. `d2dctl` trace (always on, `Client.Trace`/`TraceReport`; `D2DCTL_TRACE=60s` logs a table per verb with count, mean, max, histogram, bytes) and `D2DCTL_PPROF=<file>` (CPU profile plus a wall-clock goroutine sampler). V.
2. `Client.Prefetch`: the 8 reads of a snapshot go in ONE write; d2d dispatches every complete line of its buffer in one pump (S reading `devctl.cpp`, V live: 8 reads cost one round trip). The typed calls find their reply in a one-shot store (`Source.Read` prefetches, `DropPrefetch` at the end; `Options.NoPrefetch` is the old behaviour). Same snapshot as before: `TestPrefetchSameSnapshotOneWrite`. On lab2 (16 ms frames) this alone took the ratio from 0.5 to 0.9 (3 runs).
3. Fork `lab4` (docs/d2d-fork.md, commits on the forge): idle and last-step lockstep frames are drawn at 10 fps and wait 1 ms instead of 16. Verbs drop from 16.6 ms to 0.1 to 1.3 ms, `step` from 36 ms to 3 ms.
4. Not done: bigger `step` chunks and a lower read cadence (c): after 2 and 3 the d2d accounts for about 25 % of the wall time, the rest is in the bot (not yet located, see below), so cadence would not help yet.

### Result (V, trace and `pace` lines)
| run | d2d | virtual / wall |
|---|---|---|
| base (before) | lab2 | 0.6 (108 s in 180 s) |
| go (prefetch) | lab2 | 0.9 (166 s in 180 s) |
| l42 | lab4.2 | 2.7 in the first wall minute, 1.8 over two, falling with deaths |
| l44 | lab4.2 | 3.6 first minute, 2.0 over 2 minutes |
The ratio falls because the Sorceress at clvl 1 chickens out of the Den within 2 to 3 minutes (Save and Exit quits d2d, the runner then restarts it: wall-time bound restarts of 5 to 10 s). **The 20x gate is NOT met**: peak 3.6x. In the live fight minutes d2d answers in a few ms per step and the Go side is idle (40 s CPU profile: 50 ms of samples), so the missing time is WAITING on something that is not d2d: candidates (U, unlocated): the `time.Sleep(120+350 ms)` hover waits of `d2dtown.hoverFirst.Click` (town clicks), the 50 ms wall sleep per world link poll (`LateClock.Poll`, `d2dcombat/rig.go:72`), the Prefetch/Do mutex shared with the health `Ping`, restarts. A wall-clock goroutine sampler is in (`D2DCTL_PPROF=file` writes `file.stacks`); the windows I got fell on restart phases. Next step: sample inside a fight without a chicken (a character that does not die, e.g. after D2DLEVEL), then remove the largest wait.

### Inventory flag (f)
`dump ui` of lab3 and later has `rects` (only open panels): `d2dsrc` sets `UIInventory` from `rects.inv_grid` (V by `TestInventoryFlag`, rects format V live in D2DVERBS). On lab2 (no `rects`) the `Client` tracks the hotkey (`key i` toggles, `key Escape` clears; `Client.InventoryHint`), reported as derived/U. V offline only; the live effect on `hand.OpenInventory` (equip, belt buy, stash) was not exercised. Use `D2D_BIN=d2d.lab3` or newer for the true flag.

### Socket "vanishing" (g) and d2d.out (h)
The socket did not vanish by itself (V live, three runs, `Proc.Diagnosis` in the log): combat's `Chicken` action calls `Leave` = `Cycler.LeaveGame` = Save and Exit, which sends `quit` to the managed d2d; the process exits and `Proc.clear` removes the socket file. The runner then notices "exited" only about 90 s later (the log shows `d2d channel not answering: ... socket file is gone` and, now, `d2d: Save and Exit requested (chicken / leave game)`). The D2DPACE guess of a d2d crash is not supported: d2d.out ends with normal `alive:` lines. Consequence for the bot (not changed, not my directories): a chicken exit should re-enter the game at once instead of waiting for the monitor.
`d2d.out` is rotated per start: the previous file becomes `d2d.out.<mtime>` (24 kept, files under 2 KB are not kept); `Proc.Logf` reports the exit state and the log tail whenever the channel stops answering.
## level (agent D2DLEVEL, 2026-10-30)

Question: why does the wired bot enter the Den at clvl 1 on d2d?

- Inputs are fine (V live, `d2d.lab2` session "Levely", log: `recovery: farm area Blood Moor (xp/kill est 10 at clvl 1)`, `level:1`): class is known (Sorceress, class start picks the Sorceress table), level reads 1, belt 4 potions, mana 0 shown by the minute line.
- Root cause 1 (shared logic, fixed, V offline `TestDenFloorDoesNotYieldFarBelowFloor`): `target0` waived the Den floor ("den: floor waived, nothing else to level in") whenever Blood Moor was marked cleared and the belt held 2 potions, at any level. Now the yield only applies within one level of the floor (`denYieldGap`), so a clvl 1 to 3 caster waits for a fresh game instead.
- Root cause 2 (d2d adapter, NOT fixed, owner D2DSPEED / the d2d nav layer): Blood Moor is marked "cleared" 3 s after arrival with 0 kills. d2d hands out the whole level map at once (docs/d2d-backend.md "Map growth"), so `Navigator.Explore` returns `ErrExplored` immediately and `sweepPass` treats `!hostile()` after that as "cleared". A real sweep on d2d needs to roam to points of the level (a walk target list) instead of frontier exploration; until then no kill ever happens in the first field and the character stays clvl 1 with 0 gold.
- Live evidence after the fix (V, one 9 min session): Den not entered; Blood Moor "area cleared" in 3 s, then `leveling: everything worth farming is cleared, starting a fresh game` repeated about every 2.5 s with the character in town (no new game visible in the log; whether `Cycler.NewGame` runs on d2d from this path is U).

## roam (agent D2DROAM, 2026-10-30)

Problem: d2d returns whole level maps, so `Navigator.Explore` ended with `ErrExplored` at once and `sweepPass` marked areas cleared with 0 kills (section 'level').

Design (V offline, `internal/nav/roam_test.go`):
- `d2dworld.World.FullyKnown()` is the capability; `Navigator.explore` checks `n.w.(interface{ FullyKnown() bool })` and calls `roam` (internal/nav/roam.go). Game.exe (`d2map.Tracker`) has no such method and keeps the unknown-cell exploration unchanged.
- Coverage points: one reachable walkable cell per 7x7 block (about the monster wake radius, S, not measured on d2d), unreachable ground is not sampled. Tour: nearest by walking distance from the player, re-picked every leg. A point counts as visited when the character is within 5 cells; a failed leg (stuck, no path, timeout) gives the point up.
- `done()` (hostile near) ends the walk with nil, the bot fights and calls Explore again; the state (`Navigator.cov`) persists per level and grid, so visited points are not repeated. `ErrExplored` only when every point is visited or given up, or after 3 times the point count in legs (cap). Explore timeout still applies per call.
- Bot: `sweepPass` counts kills per area and logs `sweep: coverage N/M points, kills K, cleared=true` when it marks an area cleared (coverage 0/0 on Game.exe).
- Live (V, d2d.lab2, lockstep, throwaway Sorceress, two runs of 7 to 8 min): Blood Moor sampled to 1764 coverage points (the map is about 86k walkable cells, so a full tour is long: 3x cap is 5300 legs); 21 to 23 kills and xp 480 to 489 per run (clvl 2 needs 500), gold 0 (autoloot off, no pickup wired), no death, no clear within the run. First run rebuilt coverage on every town trip (d2dworld swaps the Grid on refresh; fixed by keying on the LevelMap, second run built it once). Still blocking leveling: no gold or potion refill (loot not wired), mana 0 at clvl 1, a full Blood Moor tour is too long for the sweep timeout, so cap or sparser points (spacing 10+) is the next knob (U).

### Round 2: 20x reached (D2DSPEED, V live)
Changes: fork `lab5` (= lab4 + one commit, `bin/d2d.lab5`, docs/d2d-fork.md): lockstep never draws except `step n render` or a `screenshot` (its reply is deferred until a frame is drawn), and the idle loop sleeps in `poll()` on the control socket so a command runs on arrival. Go: wall sleeps under lockstep now step the clock (`hoverFirst` 120+350 ms via `Controller.Wait`, link poll 50 ms removed in `LateClock.Poll` and `d2dcombat/rig.go`), `Proc.Alive` skips the ping while a request is in flight (`Client.Busy`), and `Cycler.LeaveGame` re-enters a managed d2d at once (`Reenter`, so a chicken no longer leaves the bot without a game for about 90 s). Lab hook `D2D_BOOST="12=25,7=153600,6=153600,14=5000"` (debug stats on the fresh throwaway character) gives a surviving clvl 25 Sorceress.

Session: `D2D_BOOST=... D2DBIN=d2d.lab5 run.sh r5 d2bot.speed5 10m` (seed 7, campaign, `--run-for 10m`):
- 600 wall s = 16445 virtual s: **27.4x** (60 s: 59.8x, 300 s: 32.2x, 600 s: 27.4x; the cumulative ratio falls as the fight gets denser, marginal rate in the last minutes about 22x). 142278 steps, 411121 ticks: 2.9 ticks per step, 1.46 ms wall per tick.
- 0 deaths, clvl 25, 52 kills in the final game, area 8 (Den) with the combat/clear loop running (same behaviour classes as before; no area progression beyond the Den in 10 minutes: the planner, not the speed).
- Verbs per step (trace): step 470 us, `dump` 50 us (2.3 per step: head and items), state 60 us, monsters 50 us. d2d time is about 110 s of 600 s (18 %). The Go side runs at about 107 % CPU: 31 % of the samples are GC (`gcBgMarkWorker`), the rest bot logic (`taskActs.Clear` sweep 20 %, nav, combat). Remaining ceiling is bot CPU plus allocation (U for exact split; CPU profile `cpu5.prof`).

Second session, seed 11 (`SEED=11 ... run.sh r6`, 10 wall min): 33.5x at 540 s (18109 virtual s), 0 deaths, clvl 25, 149 kills in the game: the gate holds on a second seed.
