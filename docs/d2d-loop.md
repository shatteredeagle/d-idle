# Full bot loop on d2d: hours of a fresh Sorceress (agent D2DLOOP, 2026-10-30)

Tags: V = verified live on d2d (how stated), U = unverified. Card: `openspec/changes/d2d-loop-hours`.
Setup (all V): fork binary `d2d.lab5` (docs/d2d-lab.md), lockstep, managed by the bot, fresh Sorceress "Loopy" (`--d2d-class 4`, no boost, no cheat verb,
no `internal/cheat`), `--goals campaign --toggle autoloot=off,autoparty=off`, own socket and home under `~/d2d-lab/loop/` on the worker host, `flock /tmp/d2d-lab.lock`,
`nice -n 10`, stop by `D2DSOAK_HOURS` (virtual clock). Runner script: `~/d2d-lab/loop/m.sh TAG BOTBIN SEED HOURS` (copy in the report; not in the repo).
Deviation from the brief: **no `--no-save`**. A chicken exit quits the d2d process; with `--no-save` the character is gone on every chicken (r3: xp back to 0 after each
one). Without it the managed d2d saves in the bot's own home dir and the next start loads the character (`Cycler`, V r4 onwards). Saves are private to the run dir.

## Determinism of lab5 (phase 0, V)

`det.py` of the fork (fresh process per run, `--lockstep --seed 3`, `debug stat 12 8`, `debug goto 2`, 8 blocks of `step` with `monsters` after each, md5 of the monster dumps):
`D2D_BIN=bin/d2d.lab5`, 3 runs of 8 x 500 steps (passive zombies): hash `b6db2744...` x3 (equal); 3 runs of the fight scenario (player warped next to the monsters, clvl 30, 8 x 250 steps):
hash `5f906303...` x3 (equal), and the same value as the lab2-era run in docs/d2d-fork.md section 5. So lab5's frame-loop change did not alter the simulation for these scripted walks and fights.
Not covered: the whole bot loop twice with the same seed (the bot's decisions depend on wall-clock timing in places, e.g. the health monitor and the lockstep sleeps), see below: runs with the same seed diverge.

## Run log

| run | seed | virtual h | clvl at end | what broke, fix |
|---|---|---|---|---|
| r1 | 1 (attached via d2dsoak) | 1.6 | 1 | post-death chores looped on "panels still open" (stash and inventory flags stay set, `d2dinput.CloseAll` only escaped a vendor window). Fix: Escape for stash/cube/inventory/skill tree/waypoint too (`UIFlags.Inventory` from `ui.rects.inv_grid`). |
| r2 | 1 | 2 | 1 | `roam` coverage points next to the Den mouth: walk enters the Den, the guard walks back, the roam restarts, for 1.5 virtual hours at 10 Hz. First fix: drop coverage points within 7 subtiles of a tile exit (`dropNearExits`); not enough alone (see r12 to r17). |
| r3 | 1 | - | 1 | chicken with `--no-save` wipes the character; dropped `--no-save`. |
| r4 | 1 | 1.9 | 4 | leveler "panel did not open (flag 0x2)": `dump ui` has no character-panel flag, stat points never spent. Fix: `Client.CharacterHint` (hotkey tracked like the inventory hint on lab2, U whether it can desync), `d2dsrc` sets `UICharacter`. Also `ExitTo` "no path: empty path" next to a tile exit now counts as arrived. |
| r5 to r8 | 2 | - | - | bot halted: "game lost (exited) and the restart failed twice" after a chicken. Not a double leave (serialized `LeaveGame` anyway); did not reproduce with extra logging, so the cause is U (timing between the new d2d start and the monitor's 46 s ping rule). 9 chickens in r8 ran clean. |
| r9 | 3 | 0.3 | - | **no pickup ever ran**: `seams.Loot` was never copied into the d2d backend (`cmd/d2bot/d2dbackend.go` copied Fighter and Town only), gold stayed 0 for all earlier runs, every death-less run was a poverty run. One line. |
| r11 | 3 | 1.9 | 4 | gear: every equip failed "cursor never held item N": `dump items` does not list the item on the cursor, `dump ui` gives only its base code (`held`). Fix: `d2dsrc/cursor.go` (the unit missing since the previous read with that code). Equip of cap, gloves, boots, armour works afterwards (V log lines "equip: confirmed"). |
| r12 | 4 | 4 | 4 | belt fill: shift-click unsupported on d2d: fall back to the drag, then `cmd belt put <id> <box>` (lab3 verb) through the town input; stall: Den ping-pong for the last 1.5 virtual hours. |
| r13 to r17 | 4 | 3 | 5 | Den ping-pong root cause (V, `cmd move` probes from the arrival point in the Blood Moor): arriving from the Den puts the character inside the warp's trigger area; the area reaches about 2 tiles south and west of `debug warps` position (67,28), east, north and the diagonals are safe. The walk to town started south-west through it. nav's exit penalty was a plateau at the 250 cap (no gradient). Fix: radius 14 subtiles, peak 31 (below the cap) so the penalty has a gradient away from the mouth. 157 Den entries in r17 are mostly the legitimate "Clear the Den" quest at clvl 5. |
| r18 to r20 | 5 | 6 | 6 | Den mouth again on another map seed: d2d's own pathing leaves the arrival point in the direction of the warp whatever our waypoint says. Probes (`mouth.py`, V): seed 5 arrival (28.1,28.1) warp (27,28): moving south or west fires, north/east/diagonals do not; seed 4 arrival (21.1,69.1) warp (21,68): north and east fire, south/west do not. Moving straight away from the warp's position was safe on both. Fix: `Navigator.mouthStep`: a walk that starts within 9 subtiles of an exit it does not want first goes 20 subtiles directly away (V log "nav: stepping away from a cave mouth", Den entries fell from hundreds to the legitimate ones). |
| r20 | 5 | 6 | 6 | 17 % of the time "Returning to the field" with no kills: `BackToField(3)` from the town: portal, waypoint and `ExitTo` all fail for a level that is not a town neighbour. Fix: fall back to `ToArea` (level chain). |
| r21 | 5 | - | 3 | **71 chicken restarts in a row**: after a restart the d2d world cache (`d2dworld.World`) was never reset (`Cycler.OnCycle` was not set), so the navigator's roam coverage of the previous game said "everything visited", the planner declared the area cleared and started a fresh game, again and again. Fix: `Cycler.OnCycle = World.ResetNext` (swapped by the next update on the reader's goroutine). |
| r22, r23 | 5 | 6 | 6/5 | poverty sell trips every 6 virtual minutes for 1 to 2 junk items, each a 5 minute round trip on foot (about 40 % of the farming time). Fix: without a town portal scroll a poor character needs 4 sellable items (`povertyWalkMinItems`, shared `internal/bot`: affects Game.exe runs too, flagged). r23 then halted ("restart failed twice"): the restart after a chicken ran under combat's fight context, the runner cancelled it after the d2d process was launched, the next leave failed on the missing socket. Fix: `LeaveGame` runs the restart under its own context and continues to `EnterGame` when `SaveExit` failed (the reap already ran). |

## Pace (V)

Bot attached to d2d under lockstep (clvl 1 to 6 character, fights, restarts included): 46x to 62x real time over 2 to 5 wall minutes (`pace` lines: 17.3e3 virtual s in 300 wall s = 58x, 8.3e3 in 180 s = 46x). The 27x of D2DSPEED was a boosted clvl 25 character in the Den (denser fights); a fresh low-level character is cheaper per virtual second.

## Replicated runs (4 seeds x 2 virtual hours, bot binary of r23's fixes, V)

Seeds 11 to 14, `m.sh`, 2 virtual hours each (about 2.5 wall minutes each), fresh Sorceress, no cheats. Metrics by `d2dsoak extract -hours 2` over the bot's minute lines, compared with
`tools/d2dsoak/baselines-early.json` (the only baseline of a fresh clvl 1 to 7 character: 1 run, old builds, S quality) and `baselines.json` (clvl 11 to 14 Game.exe runs: not the same character, shown for the record).

| metric | d2d (mean of 4, sd) | early baseline | gap | late baseline (not comparable) |
|---|---|---|---|---|
| XP/h | 3013 (580) | 6600 | -54 % (outside the 20 % gate) | 10901 |
| kills/h | 132 (23) | no value | | 500 |
| deaths/h | 0 (0 of 4 runs had a death) | 0.29 | -100 % | 0 |
| clvl after 1 h / 2 h | 3 / 4 (all four seeds) | no value | | 13 / 14 |
| state shares | recovery 63 %, town 19 %, farming 18 % | | | farming 70 %, recovery 6 % |

The tool's verdict is INCONCLUSIVE (4 seeds, 8 virtual hours in total, below its minimums of 5 seeds, 3 h each and 15 h for deaths). Read as indicative: **XP/h is about half of the early Game.exe baseline and kills/h about a quarter of the late one; deaths are
below the baseline (no death in 8 hours), because the bot chickens out (9 to 23 restarts in 3 to 8 hours) instead of dying.** The fidelity gate (within 20 % on XP/h, kills/h, deaths/h, deaths per 1000 kills) is NOT met.

Single long runs (seed 5, same fixes): 6 virtual hours clvl 6 / 18.2k xp (r22), 8 virtual hours clvl 6 / 19.7k xp, 762 kills, 1 death (r24, 8 h = 8 wall minutes at 43x to 58x). Per virtual hour of r24: clvl 4 at 1 h (3.9k xp), 4 at 2 h, 5 at 3 h, 5 at 4 h, 5 at 5 h, 6 at 6 h, 6 at 7 h and 8 h (the hourly marks are by minute-line index).
**clvl 10+ was not reached**: clvl 6 after 8 virtual hours (the XP needed per level grows faster than the rate of this loop: 2.4k to 3.3k XP/h).

## Where the time goes (V, r24 `idle.py` over the minute lines, before the border-loop breaker)

Farming gold in the recovery mode 37 %, "Returning to the field" with zero kills 30 % (the Cold Plains / Cave Level 1 shuttle, below), town trips and chores 14 %, quest fights 11 %, quest travel 4 %, death-spot sweep 1.5 %.
Kill rate while fighting: 2.5 to 3.4 per minute (about 150 to 200 per hour), against about 8 per minute of the Game.exe baseline. Out-of-mana waits (a clvl 1 to 6 Sorceress has 35 to 60 mana, Fire Bolt / Charged Bolt, "combat out of mana: waiting"),
and a fresh character with 4 potions and no gold: **the poverty mode never ends** (`gold 0 below kit cost 190/500`), so the bot stays in "recovery: farming gold" in the Blood Moor and leaves for a sell trip as soon as junk piles up.

## Gates of the card

- "at least 4 virtual hours without a bot stall longer than 10 minutes, or the causes are listed": met for the single runs only after the fixes above (r17: 3 h, r22: 6 h, r24: 8 h had no bot-level stall except the Cave Level 1 shuttle, 143 virtual minutes, cause listed, breaker added, check below); before the fixes every run stalled (Den shuttle 1.5 h, stale roam coverage storm, pickup never wired, equip never worked).
- 3 seeds x 4 virtual hours: NOT done as specified (4 seeds x 2 h, plus 2 long single-seed runs).

## What still blocks (U where not measured)

1. Economy of a fresh character: gold income is about 20 gold/h net, the kit is never complete (no town portal scroll most of the time: every town trip is a walk), so the sell trips and the potion-less fights chicken out (23 restarts in 3 h at the worst, 9 in 8 h at the best).
2. Cave mouths: d2d's trigger area depends on the map seed (Blood Moor/Den seeds 4 and 5, Cold Plains/Cave 1 seed 5 probed): walking straight away from the warp's position is safe in the probes, but the walk to the "away" point can still be routed by d2d's own pathing through the trigger (Cold Plains). Proper fix needs the warp's trigger rectangle from the fork (`debug warps` has only a point): ask the fork for the tile size, then block those cells in nav (lab6 verb).
3. Belt fill: `cmd belt put` is wired (BeltPut through `HoverFirstBelt`) but only the last runs exercised it, see the run log.
4. Repair "not confirmed (durability 0% unchanged)" and "1 item no vendor would buy" chores fail every visit; sell prices and the repair button geometry are Game.exe models (docs/d2d-town.md).
5. A restart after a chicken: seed-dependent halts seen in r5 to r8 ("restart failed twice") were not reproduced after the serialization and the context fix; U whether another cause remains.
6. Determinism of the whole loop is not given: the bot's timing (health monitor, wall-clock waits) makes two runs with one seed diverge, so the replicated runs are different games per seed AND per rerun.

## Border-loop breaker, first live look (r25, V)

r25 (seed 5, breaker with 8 crossings per minute and a 10 minute skip): it fired 3 times in the first 2.6 virtual hours, all in the Blood Moor against the Den mouth (nav stuck beside the mouth after the step-away blacklisted cells), and the planner then chose Burial Grounds (Blood Raven) at clvl 1 because the other areas were banned (no death followed in that run). Therefore the shipped values are 10 crossings per minute and a 3 minute skip; this setting is NOT validated live (U). Cold Plains / Cave Level 1, the case it was written for, did not recur in r25.

## Changes outside the d2d directories (announce)

`internal/bot` (`goal.go` poverty trip threshold without a town portal scroll: shared with Game.exe; `loop.go`/`runner.go` border-loop breaker: shared), `internal/nav` (`nav.go` exit penalty, `walk.go` mouth step, `roam.go`, `exit.go`: shared with Game.exe), `internal/town` (`trip.go` BackToField route, `beltfill.go`: shared), `internal/loot` (one debug line).
Game.exe behaviour of these is untested here: the poverty threshold, the exit penalty radius (6 to 14 subtiles, gradient) and the mouth step apply to Game.exe runs too and need a G1/G2 check before they are merged into a Game.exe soak.
`cmd/d2bot/d2dbackend.go` is owned by the card d2d-default-backend in docs/OWNERS (one added `seams.Loot` copy).
