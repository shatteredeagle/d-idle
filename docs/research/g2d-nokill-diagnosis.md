# Gate 2 g2d: why NOKILL failed on seeds 11 and 17

> status: current. Read-only analysis of the committed minute logs of the g2d run (2026-11-05, seeds 11, 13, 17); no game, no Go changes. Tags: V = read from a committed log or source file (quoted), U = inference or unverified. Times are the log's wall clock (-05:00); the criterion counts virtual minutes (one `minute` line each, wall length varies from 1 s to 22 s per line, so wall time is NOT virtual time).

## 1. How NOKILL is measured (V)
`internal/d2dsoakstat/invariants.go` lines 358-372: for each `minute` line i>0, if `kills` > previous `kills` OR `xp` > previous `xp` the counter resets, else it counts +1; `NoKillXPMin` is the maximum; violated when > `NoKillMin` = 10 (lines 58, 65, 541). It uses the JSON key `kills` (not `killsGame`, which equals `kills` in all windows below) and no state filter: town minutes, chores, travel, fresh games and `waiting` all count as no progress. Window n = number of lines, so n = 11 means 11 consecutive lines without a kill or XP gain.

Extraction: JSON lines with `msg == "minute"`, 0-based index. Logs have more lines than the 1889-line cut of the multiseed record (s13 2463, s17 1980, s11 1889); every window below lies inside the first 1889 lines.

## 2. Windows >= 8 lines (V)
| seed | idx (first-last) | n | wall first line to last line | last progress line / next progress line | note |
|---|---|---|---|---|---|
| 11 | 732-742 | **11** | 10:09:29 to 10:10:06 | 731 10:09:24 / 743 10:10:10 | FAIL, matches record |
| 13 | 783-791 | **9** (peak, PASS) | 09:43:52 to 09:44:36 | 782 09:43:51 / 792 09:44:39 | |
| 17 | 1151-1167 | **17** | 11:30:46 to 11:33:06 | 1150 11:30:35 / 1168 11:33:28 | FAIL, matches record |
| 17 | 1202-1209 | 8 | 11:36:26 to 11:37:10 | 1201 11:36:25 | under the limit |
| 17 | 1585-1592 | 8 | 13:36:40 to 13:37:46 | 1584 13:36:23 | under the limit |

Context (V, same script): runs of >= 5 no-progress lines over the full logs: seed 11: 6 runs; seed 13: 19; seed 17: 18. So the 10-line limit sits only a few lines above the normal background (5 to 8 lines is routine), and the tail decides the verdict. Earlier g2c logs (same seeds, 2026-11-04 23:3x): seed 11 none >= 8, seed 13 12/8/13 lines then the known cursor wedge (8021 lines), seed 17 one run of 10. So a 10-line stretch is not new in g2d (V for the counts; whether #186 changed anything is U).

## 3. Per-window diagnosis

### 3.1 Seed 11, idx 732-742 (11 lines): unplanned new-game cycling (failsafe ladder)
Evidence (V, minute fields):
- `rung` (the bot's failsafe ladder, see 4.1) goes 1 -> 3 at idx 737 (10:09:50), 3 -> 4 at 738, 4 -> 1 at 739, 1 -> 3 at 740, stays 3 through 745, 746 3 -> 1, 747 1 -> 3, 748 3 -> 4 (rc listing). Over 737-757 `rung` changes 20 times.
- `restarts` (counts unplanned new games, observe.go newGameRun) goes 16 (idx 729) -> 17 (731) -> 18 (737) -> 19 (740) -> 20 (741) -> 21 (747) -> 23 (751) -> 24 (756): five to eight fresh games in 20 lines, three of them inside the window (17 -> 20).
- `area` inside the window: 1 (town) in 6 of 11 lines (idx 732, 736-740), others 3, 18, 6, 17 on single lines; all `state=farming`, `step=Quest: fighting (Level up to 17)`, `rstate=clearing`.
- `life` 187-190/190 on every line, `hpMinPct` 94-100, `potions` 4 constant, `gold` 550 -> 350 (two drops of 100 at idx 733 and 738), mana full (63/63), `deaths` 0. So not a fight, not a belt problem, not a death.
- Before the window: idx 722-727 normal fighting (kills 4957 -> 4966), 728 town chores `Chores in town (heal below the resume line)` (a=1), 729-731 farming with kills (4983 -> 4992), and `merc: died` at idx 817 and 832 only (nothing near the window; merc deaths at 178, 179, 186, 501, 504 are unrelated).
- No stall/kill/restart lines in this window (the only non-minute lines of the seed are `killtime` 274, a `d2d process exited` at 441, plus summary lines).
Diagnosis: the character kept leaving and re-entering the game (rung 3 text is `Failsafe: leaving and re-entering the game`, V at idx 485 step field and loop.go lines 574-593). Each cycle puts it in town and costs 1-3 lines; five cycles in a row produce 11 lines without a kill. Class: failsafe/new-game loop, not a stuck fight. What action failed to start the ladder (U): the `failsafe ladder` WARN carries `what` and `err` but is not in the committed record; candidates are a walk/travel failure (ErrStuck) or an input failure on the way back to the field. Why rung 3 ran four times in 12 lines is also U (rung 3 resets only after `realResetMin` = 3 real-progress units).

### 3.2 Seed 13, idx 783-791 (9 lines): same signature, shorter
V: `rung` 1 -> 3 at idx 784 (RS 14 -> 15), 3 -> 4 at 785, `rung 4` through 789, 790 4 -> 1 (RS 15 -> 16). `area` 1,1,7,7,9,1(chores `[heal]`, hp 159/190),19,18; `state` farming except idx 788; `hpMinPct` 83-100; `potions` 5 stable; deaths 1 (old, idx 67). Right after the window kills resume in area 18 (idx 792: +17 kills). Diagnosis: one failsafe cycle with two fresh games, resolved by the next area; the 9-line peak (limit 10) is the same mechanism as seed 11 but one cycle shorter, i.e. luck decides pass or fail (U).

### 3.3 Seed 17, idx 1151-1167 (17 lines): empty field after a resupply trip, no failure visible
V:
- `rung` = 4 on every line from idx 1057 to 1189 (rc listing: last change 1057, next 1190), `restarts` = 12 constant through the window. Rung 4 is therefore a stale value (the ladder resets only inside the next escalate call, loop.go 537-541), not an active failure; no fresh game happens in the window.
- idx 1143 (11:28:51) `Chores in town ([low on mana potions])`, back in the field at 1144. At idx 1152-1155 (11:30:51-57) `Returning to the field`, areas 1,1,1,2 then 4 at idx 1156 (11:31:04): the second town trip begins right after the last kill (idx 1150, kills 7073), gold 442 -> 222.
- idx 1156-1166: `area` 4 (Stony Field, V: drill/d3.go and saves/progress.go name area 4) on 11 consecutive lines, `walkingRaw` 1, `state=farming`, `step=Quest: fighting (Level up to 17)`, life 190/190, `hpMinPct` 100 (no fight in any of these minutes: takeHPMin returns 100 when no fight, hpfields.go), kills 7073 and xp 191188 unchanged, `stam` varying 2912-10022 (changes line to line; so the character moves, U), `invPct` 30 -> 35 at idx 1161 (an item was picked up, U), `goldPickup` 0.
- Wall pace in the window is 10-12 s per line, the normal fighting pace (idx 1141-1150: 11-18 s), not the 2 s per line of town or loading.
- idx 1167: area 7, 1168: area 6, kills resume (+4, xp +297, hp 155/190).
- The same pattern is recorded in an earlier soak: docs/probes/postfix-soak2.md lines 58 and 94, "26 virtual lines ... `Recovery: farming gold` then `Quest: fighting` in area 4".
Diagnosis: a level-up fight step in area 4 with nothing to kill for about 12 lines at full health, after a chore trip. Class: waiting for / searching for spawn or exploring an already-cleared field (U); a pathing loop that does not move is argued against by the stamina changes (U). Not a stuck fight (no hostile fight, life 100 %), not belt/potions (potions 6, 6), not merc (last `merc: died` before is idx 968, none in window), not a portal loop (area 1 only 3 lines, all at the trip start). The step is a `Clear` in `taskActs.Clear` (internal/bot/task.go 515ff): it sweeps until exploration is exhausted (`sweepPass`), where one explore may run `nav.exploreTimeoutSec` = 300 s (tuning_nav.go default), and the field may be sweepable for minutes with zero monsters if the earlier game cleared it (idx 1141-1150 had kills right up to 1150, so this game still had monsters near; after them the rest was empty, U).

### 3.4 Seed 17, idx 1202-1209 and 1585-1592 (8 lines each), for completeness
1202-1209 (V): `rung` 1, `restarts` 14, town chores 5 lines, farming 3, `potions` 4 -> 6: a resupply trip plus walk back; legitimate chore time (U that nothing is wrong). 1585-1592 (V): `rung` 1, `restarts` 20 constant, areas 1,6,17,18,7 with `state=farming`: hopping between areas without kills (U: the planner moved on from empty areas); no failure signature.

## 4. Comparison with the stuck-fight fix (#186, `internal/nav/stuckfight.go`)

### 4.1 What the `rung` field is (V)
The minute line's `rung` is `r.ladder` (observe.go line 960), the bot FAILSAFE ladder (loop.go `escalate`: 1 skip target/close panels, 2 reset to town, 3 leave and re-enter game, 4 restart game process, else self-pause). It is NOT the nav stuck-ladder rung that `stuckFight` reads (`w.rung` inside one walk, `nav.maxRecoveries` 6). So `rung >= 2` in the minute log says nothing about the #186 trigger.

### 4.2 Trigger conditions vs the windows
`stuckFight` runs only from `walk.go` line 298, inside a GoTo that has not moved 2.5 subtiles for `nav.stuckMs` (1200 ms) and only when `w.rung >= 2`, life < `nav.stuckFightLifePct` (60) and hostiles within the fight radius (stuckfight.go lines 18-28). Windows:
- Life: every minute-line life sample in the three windows is >= 84 % (s11 >= 98 %, s13 >= 83 % by hpMinPct 83, s17 100 %), V. Life only on minute samples, so a short dip between lines is U. With life >= 60 the function returns at line 23 before any log: it did NOT trigger and could not (thresholds not reached: life, plus no hostile in the s17 window).
- Hostiles: none are fought in the s17 window (hpMinPct 100 = no fight), V.
- Rung: the nav rung is not visible in the minute log.
Conclusion: #186 is not the cause and not the cure here; the failure modes are different (3.1/3.2 failsafe cycling, 3.3 empty field). Its log lines, if it had fired, would be `nav: stuck ladder -> fight` (rung, life, hostiles, pos) or `nav: stuck ladder -> portal`; plus `nav stuck`, `nav: stuck recovered`, `nav stalled by monsters, yielding`. None of these lines is in the committed record (the minute-log files hold `minute` lines and a few others only, V by the message census: seed 11 `merc: died` x17, `killtime`, `combat kill`, `d2d process exited`, `game lost, restarting it`, `death report`; seed 17 `task step` once), so the absence of a nav line proves nothing (U).

## 5. Proposals, ranked (smallest change per cause; proposals only)
1. **Capture first (no fix before it).** Seed 11 and 13 windows cannot be fixed blind: the `failsafe ladder` WARN (loop.go line 550: step, what, err) and the nav lines are the missing evidence. Either keep them in the soak collect (ops side, next run: all WARN/ERROR and every line containing `failsafe`, `nav stuck`, `stuck ladder`, `campaign stalled`, `planner`, `clear killed nothing`, `hunting leftovers`) or put the last failsafe `what` into the minute line (`internal/bot/observe.go` minuteLine, one new key, ParseMinuteLines ignores unknown keys).
2. **Seed 17 class (empty field, 12 lines): shorten the empty sweep.** Smallest knob: lower `nav.exploreTimeoutSec` (default 300 s, range 30-1800, tuning_nav.go) so a monsterless explore gives up and the planner moves on; or in `sweepPass`/`taskActs.Clear` (task.go 515, goal.go 1154) end a sweep that killed nothing for N virtual minutes (new tuning knob, `bot.*`). Risk: leaving fields with spawns not yet seen. The second form acts on exactly what the criterion measures.
3. **Seed 11/13 class (new-game cycling): bound the loop.** Hypotheses only until item 1 is read: (a) `FailsafeRung3Max` style cap per window already exists (loop.go 576) but counts only rung-3 cycles per `ladderProgress`; a lower cap or a longer wait after a rung-3 game that did not reach a first kill would pause instead of recycling (`internal/bot/loop.go` escalate). (b) find why the first action after a fresh game fails (walk back to the field).
4. **Watchdog mismatch.** `bot.stallWarnMin` = 30 and `stallExitMin` = 30 (tuning_bot.go line 78) measure `campaignAt` progress, which counts area entries, purchases and quest steps (stall.go header), so they can neither see a 10-minute kill drought nor fire in an area-hopping window (V text, U effect). A kill/XP-only drought watchdog at 8 virtual minutes that triggers a town portal + fresh area (not a restart) would align the bot with the gate; it needs its own knob and is the largest change here.
5. Chores: seed 17 shows `[low on mana potions]` chores at idx 1143 and again 1170 with `packMP` 0 before and after (V fields); a repeat of the same chore reason 27 lines later may be a resupply that does not buy mana potions (U). Smallest check: `internal/town` chore for mana potions (not investigated).

## 6. Criterion note (for the USER; proposal only, no change made)
- Implementation vs wording (V): the criterion text says "without progress" and "state"; d2dsoak counts only kills or XP and applies no state filter (invariants.go 358-372). Bot-side progress (stall.go) includes area entries and purchases, and the d2dsoak PROGRESS invariant (limit 30) counts those too (docs/probes/postfix-soak2.md line 58). So there are three different "progress" definitions.
- An exemption for `town chores` / `resupplying` (suggested earlier in docs/research/gate2-analysis-20261102.md line 60) would NOT excuse these failures: the seed 11 window is 11 of 11 `farming` lines, the seed 17 window 13 farming + 4 town chores (V, state field). Town trips are a minority contribution here.
- A margin issue (V): 5 to 8 line runs are routine (seed 17 has 18 runs >= 5), so a limit of 10 lines passes or fails on luck of one extra cycle (seed 13 peak 9). If the user wants a robust gate, options: judge the limit over a rolling statistic (e.g. no more than 2 windows over 10 per 30 virtual hours) or exclude windows that contain a counted fresh game (`restarts` change) but still cap them, e.g. at 20. Not proposed as a recommendation: criteria and number tables change only with user approval.

## 7. What is U, and what to capture next run
- U: why the failsafe ladder started in seed 11/13 (what/err), what the character did during the seed 17 field minutes (sweep, loop, or wait), whether #186 fired anywhere in the run (nav lines absent), whether stamina changes prove movement, what each 100-gold drop in seed 11 was.
- Capture next run (ops side, no bot change needed): the full `bot.log` WARN/ERROR lines, any `failsafe`, `nav stuck`, `stuck ladder`, `stuck recovered`, `campaign stalled`, `planner` (area choice), `clear killed nothing`, `campaign clear: hunting leftovers`, `task step`, `shop`, `Chores` lines for windows >= 6 lines; ideally positions (`pos` appears in nav lines) so a pathing loop shows. A 60 s sampler of `x,y,area` into the minute line would settle moving vs not moving.
- Method note (V): analysis scripts read the three `*-g2d-minutes.jsonl.gz` files; counts reproduce the record (11 / 9 / 17).
