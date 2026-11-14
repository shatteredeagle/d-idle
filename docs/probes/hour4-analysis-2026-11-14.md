# Fresh Paladin first hour 4: where the 60 minutes went (2026-11-14)

> status: analysis, no code changed. Source: the sandbox bot log, build 0.0.261113174258 on d2d devp6 (rebased fork), window 09:55:24Z to 10:55:25Z (14,585 JSON lines), streamed with throwaway scripts (no log is copied into the repo). Tags: V = read from a log line or counted (the line is named), S = stated by a doc, U = unverified or inferred. Predecessor: [hour1-analysis-2026-11-12.md](hour1-analysis-2026-11-12.md) (same method, run 1). Task: TASK-407.

## 0. Result

| measure | run 4 | run 1 (02:18Z 11-12) | tag |
|---|---|---|---|
| clvl@60 / xp | 5 / 13,883 (292 XP short of clvl 6 at 14,175) | 6 / 17,032 | V `minute` |
| kills (`minute` counter) / per minute | 494 / 8.2 | 528 / 8.8 | V |
| xp per kill / per minute | 28.1 / 231 | 32.3 / 284 | V (derived) |
| deaths / chickens | 0 / 0 | 0 / 0 | V |
| zone entries / shuttles / bounces | 17 zone entries (19 `arrival` lines: 7 town, 7 Blood Moor, 5 Cold Plains) / 3 shuttles / 0 bounces | 33 / 7 / 6 | V `arrival`, `zone entry` |
| Den of Evil entered | never (4 attempts, section 3) | yes (minute 13, 28 % of the hour) | V |
| Cold Plains first entry | minute 10.5 (a 17 s visit to take the waypoint); real farming from minute 27.5 | minute 12 | V `arrival` |

Tiers (docs/first-hour.md): floor MET (clvl 5, 0 chickens, 0 deaths); target MISSED (clvl 5 not 7, 8.2 not 14 kills/min); Cold Plains by minute 15 met only as the waypoint visit. The `combat kill` lines (563 in the window, 14 % more than the counter 494, U why: the same effect as run 1's 634 vs 528) are used for splits by area and skill; the counter is used for totals.

## 1. Time budget

### 1.1 By area (wall time from `arrival` lines)

| area | seconds | share | kills (counter deltas of `minute`) | kills/min | xp per minute (kill lines) | xp per kill (kill lines) |
|---|---|---|---|---|---|---|
| Rogue Encampment (town, 7 arrivals) | 195 | 5.4 % | 0 | | | |
| Blood Moor (30.6 min) | 1836 | 51.0 % | 229 | 7.5 | 160 | 22.0 |
| Cold Plains (26.1 min) | 1565 | 43.5 % | 254 | 9.7 | 376 | 28.9 |
| Den of Evil | 0 | 0 % | 0 | | | |

V. The 11 kills credited to the town minute lines are the minute-boundary lag. Blood Moor is 22.0 XP per kill, exactly the xp-gold-trace table value (S), so the area mix, not the XP formula, explains the XP gap to run 1 (section 5).

### 1.2 By activity (V `combat fight`, `nav.trace leg` by caller, `arrival`)

| bucket | seconds | share | how |
|---|---|---|---|
| fights, including the approach walk (202 `combat fight` lines, `sec=`) | 1451 | 40.3 % | sum of `sec`; 24 ended `timeout`, 17 `error` (all Cold Plains, 9 to 17 monsters remaining, 0 to 1 kills) |
| roam / explore legs between packs (564 legs) | 1203 | 33.4 % | callers `nav.roam < explore` |
| exit and waypoint travel (15 legs) | 289 | 8.0 % | caller `ExitTo < travelTo` |
| town (7 arrivals incl. 2 save-point re-entries) | 195 | 5.4 % | |
| loot and gold pickup walks (55 legs) | 166 | 4.6 % | caller `loot.pickOne` |
| other legs (waypoint, unit) | 65 | 1.8 % | |
| unattributed (observe waits, target search, panels, state changes) | 231 | 6.4 % | 3600 minus the rows above, U |

The fight bucket overlaps the 369 combat-approach legs (`Fighter.move`, 993 s, 27.6 % of the hour, 301 stalls): most of a "fight" is the walk to the next monster. Nav stall pauses (348 `resume` lines) add up to 156 s (4.3 %); they sit inside the legs, not on top (V sum `pauseMs`).

### 1.3 Five-minute windows (V; town = seconds in area 1, kills = `combat kill` lines)

| from (min) | town s | Blood Moor s | Cold Plains s | kills | xp (kill lines) | what happened |
|---|---|---|---|---|---|---|
| 0 | 11 | 285 | 0 | 42 | 915 | gold recovery ("gold 0 < kit 500"), Attack only, clvl 2 at 2.4 min |
| 5 | 0 | 300 | 0 | 55 | 1140 | Sacrifice learned at 8.2 min; Den hunt A starts at 10.7 min |
| 10 | 43 | 240 | 17 | 8 | 195 | waypoint visit at 10.5 min, Den hunt A fails at 13.9 min, heal trip, save point |
| 15 | 63 | 237 | 0 | 33 | 804 | pack trip, BM re-cleared (3 dead minutes) |
| 20 | 0 | 300 | 0 | 48 | 1074 | clvl 4 at 23.2 min, Den hunt B (249 s, 23 kills) |
| 25 | 8 | 147 | 145 | 39 | 2514 | first real Cold Plains (2,370 XP from 32 kills in 145 s, unique packs among them, U) |
| 30 | 38 | 8 | 254 | 81 | 1787 | pack trip, clvl 5 at 33.0 min, Cold Plains 19.1 kills/min |
| 35 | 0 | 57 | 243 | 36 | 718 | Den hunt C (BM 65 s, 0 kills), back to Cold Plains |
| 40 | 0 | 0 | 300 | 84 | 1699 | Cold Plains, 16.8 kills/min |
| 45 | 0 | 41 | 259 | 17 | 445 | Cold Plains cleared at 48.6 min (`sweep ... cleared=true`, 163 kills), Den hunt D |
| 50 | 32 | 221 | 47 | 50 | 1136 | heal trip, save point, walk town to BM to Cold Plains (3.1 min) |
| 55 | 0 | 0 | 300 | 70 | 2305 | fresh Cold Plains of the third game, 14 kills/min |

(The clock minutes in the text are minutes after 09:55:24Z.) Cold Plains runs at 14 to 19 kills/min while it has monsters (windows 30, 40, 55: V); Blood Moor never exceeds 11/min after clvl 3. The whole-hour 8.2/min is the weighted result of 4 Den-hunt visits, three Blood Moor re-sweeps and a depleted Cold Plains.

### 1.4 `killtime` (495 lines, 3575 s)

| area | kills | total s | walk | cast | wait | per kill (total / walk / cast / wait) | median total |
|---|---|---|---|---|---|---|---|
| Blood Moor | 223 | 1981 | 536 | 179 | 1268 | 8.9 / 2.4 / 0.8 / 5.7 | 2.6 |
| Cold Plains | 272 | 1595 | 420 | 262 | 912 | 5.9 / 1.5 / 1.0 / 3.4 | 2.2 |
| whole hour | 495 | 3575 | 956 (26.7 %) | 441 (12.3 %) | 2180 (61.0 %) | 7.2 / 1.9 / 0.9 / 4.4 | |

V. Run 1 for comparison (hour1 analysis section 2): walk 22.5 %, cast 14.9 %, wait 62.6 %. `wait` is the residual (total minus walk minus cast; definition in `observeKills`/`killtime`), so it holds roaming between packs, loot and gold stops, the Den hunts (24 Blood Moor `killtime` lines are in state `traveling`), town and link waits; it cannot be split further from the log (U). The medians (2.2 to 2.6 s) show a typical kill is quick; the mean is dragged by the long gaps: 9 kills have a gap above 60 s and 26 above 30 s (V `gap_since_last_kill`).

## 2. Why 8.2 kills/min against the decomp14 model (14.6 in a perfect route)

What the bot does between kills, in minutes of the hour (V unless noted; the rows overlap a little, so they are not summed):

| activity | minutes | evidence |
|---|---|---|
| roaming between packs | 20.1 | 564 roam legs, 1203 s; the Blood Moor sweep reached 375 of 376 coverage points three times (09:55 to 10:09, 10:10 to 10:22, 10:31 and 10:46 to 10:49), Cold Plains 440 of 440 once |
| walking up to monsters | 16.5 | 369 approach legs, 993 s; 34 `combat target fallback ... out of melee reach` lines, 17 fights `end=error` with 9 to 17 monsters left (U cause: the d2d devp5/6 monster behaviour, section 6 rank 3) |
| Den hunts | 9.8 | 4 visits, section 3 (these also contain roam legs and 27 kills) |
| nav stall pauses | 2.6 | 348 pauses, 156 s |
| town | 3.3 | 7 arrivals; 3 chore trips, 2 of them `[heal]` that bought nothing useful (section 2.2) |
| loot and gold walks | 2.8 | 55 legs; 156 `loot:` summary lines, 52 of them one filtered item ("Bolts", no walk), 9 `loot_memory: going back`, 4 `gold grab`, 3 gold ledgers (Cold Plains 41 piles seen, 16 picked, V: the priority rule skips small piles, 2x "small gold pile" lines) |
| exit travel | 4.8 | 289 s of `ExitTo` legs (the 3.1 min walk town to Blood Moor to Cold Plains at 10:46 is mostly counted under fights and roam) |

### 2.1 Dead minutes
12 of 59 minutes have at most 1 kill, 21 have at most 4 (V `minute` deltas): 10:00, 10:06, 10:08, 10:09, 10:14 to 10:16, 10:23, 10:31 to 10:32, 10:44, 10:45. Of these about 6 fall in Den hunts (10:06, 10:08, 10:31, 10:32, 10:44, 10:45), 2 in town trips (10:09, 10:14), 2 are Blood Moor roaming after the first re-clear (10:15, 10:16), 2 other (10:00, 10:23).

`clear killed nothing; moving on` appears 33 times (15 between 09:55 and 09:59 with `remaining 0`, 16 between 10:10 and 10:19, 2 at 10:44): the idle clear of run 1 is still there but smaller. `explored` hit 99 to 100 % at 10:09, 10:44 and 10:46: the bot sweeps an area to the last coverage point even when the last 20 points hold nothing (U, a rule choice).

### 2.2 The seven town arrivals (V `town trip`, `chores done`, `trip budget`)

| time | reason | seconds | result |
|---|---|---|---|
| 09:55:28 | start (hotkeys, recovery "gold 0 < kit 500") | 11 | |
| 10:09:19 | `[heal]` (life 71 of 77 on exit from Blood Moor) | 28 chores + 3 Save and Exit + 12 re-entry | chore `shop` only: "wantMP 2" with 42 gold, bought nothing; then the save point |
| 10:13:48 | inventory 85 % | 62 | sell + bank + shop, +140 gold net |
| 10:25:17 | pack 80 % | 45 | sell + shop, +87 gold net |
| 10:46:09 | `[heal]` (life 70 of 99) | 20 + 4 + 12 | bought 1 Town Portal (100 gold); `heal` chore skipped; save point (xp at risk 8830) |
| 10:09:50, 10:46:30 | re-entries after the two Save and Exit save points | 12 each | game restarts (new game, full life, monsters regenerate) |

Both `[heal]` trips were followed by a save point; neither delivered a heal (the `heal` chore was in the skipped list both times): U whether the trigger is the "life below resume line" rule at the end of a sweep. The save points are not waste on their own: the new game regenerated Blood Moor (38 kills in 226 s after 10:10:02) and Cold Plains (see 2.3).

### 2.3 Supply is the limit
Cold Plains of game 2 gave kills per 2 minutes of 34, 14, 16, 40, 28, 5 (10:32 to 10:44, V), then cleared at 10:44:00; Cold Plains of game 3 (new game after the 10:46 save point) gave 24, 12, 20, 18, 11, 5 per minute from 10:49:38. The first minutes of a fresh area are at the model rate (15 to 20/min); the tail of every area is under 8/min. The rate the decomp14 model assumes (14.6/min for 45 minutes in Cold Plains) needs a supply the bot does not have: the `sweep` lines report 163 kills at Cold Plains (10:44) and 98 at the first Blood Moor sweep (V), so a Normal Cold Plains may hold only about 160 to 170 monsters per game (U), which would limit a perfect route to about 11 minutes of Cold Plains per game unless the game is re-rolled or the next area is entered. Run 4 never entered Stony Field or Burial Grounds (level plan: below their enter level, U which; see rank 2).

## 3. Why the Den of Evil was never entered

Short answer: the planner and the campaign wanted the Den every time; navigation found the exit but the exit tile never fired. The Den is not excluded by Den-claim memory, by the level-plan exit changes or by Free mode (all V below).

Decision chain (V lines, in order):
- 10:06:04 `planner area 8 Den of Evil ... quest => 1 skill pt`, `level plan: clvl 3 -> area 8 Den of Evil (in window 2-6)`; the campaign activity was `Quest: fighting (Level up to 4 for Den of Evil)`. At 10:18:39 the quest step `quest.den_of_evil/to-den` (activity "Go to the Den of Evil"); at 10:29:24 and 10:44:00 the level plan again (10 `planner ... Den of Evil` lines in all).
- 10:06:12 `nav exit to 8 gate {335,140} border false` (the map lists the Den exit in Blood Moor), the hero walks there in 16 s and stands ON the tile: status `x 335 y 140`.
- `nav exit tile did not fire, stepping off and back` four times (10:06:34, 10:18:55, 10:30:40, 10:45:08, all tile (335,140) off (341,140)); 7 s of `nav.trace move` clicks on the same tile each time, the level never changes.
- After the step-off the gate is gone from `findGate`: `nav no exit known yet, exploring {from 2, to 8}`; the sweep explores all of Blood Moor (`roam: done visited 375 of 376`), `ErrNoExit: level 8 from 2`, the edge is marked (second message "nothing new since the last search").
- `leveling area unreachable` / `area skipped {8, reason unreachable}` four times with the skip time doubling: 150, 300, 600, 1200 s (these are the 8 `unreachable` lines of the row). The quest step `to-den` bypasses the skip (it ran at 10:18 and 10:22, ended `blocked` at 10:22:46 and at 10:46:30 reported a misleading ban text "not ready: area banned after repeated deaths": 0 deaths; the same wrong text is item 2 of [hero14-analysis-2026-11-14.md](hero14-analysis-2026-11-14.md)).

Other facts: `d2dworld: arrived in level 2: 2 adjacent entries, 0 levels not linked` (Blood Moor lists the camp and Cold Plains; the Den is a separate warp). The earlier heroes in the same container (11-13 18:34Z and 11-14 00:59Z, `fh4.log`, hero 14) also never entered the Den (0 `Entered Den of Evil` in 16.5 hours of log; two "did not fire" lines at tile (295,220), whose target level is not in the log, U). Run 1 of 2026-11-12 did enter it (V hour1 analysis, build 0.0.261112021709 on the older sandbox d2d image), so this is a change between that image and devp5/6.

Cause (U, not found in the log): the tile that d2d lists for level 8 is reached and does not fire. Candidates, in order of likelihood: (a) the rebased d2d (devp5/6: warp-arrival rule #490, new monster AI, docs/fork-rebase-2026-11-13.md) changed how an exit warp triggers or what `warps` reports (the listed point may be the return spot from the Den, not the trigger rectangle); (b) the trigger needs a click on the cave mouth object rather than a walk; (c) the bot's `exitUnstick` step-off direction (+6 subtiles east) leaves the rectangle. The nav layer cannot tell (a) from (b).

Cost (V timestamps): four Blood Moor visits of 189 s (2 kills), 249 s (23 kills), 65 s (0), 86 s (2) = 589 s, 9.8 minutes, plus the Cold Plains round trips each attempt forced (10:29 and 10:44: Cold Plains to Blood Moor to Cold Plains, 20 to 45 s each way). Missed: the Akara skill point (a 5th point; the hero spent 4: Might 2, Sacrifice 1, Prayer 1; the Den quest pays a skill point and no XP, S xp-gold-trace Q5) and the Den's kills (22.3 XP/kill plain, the same as Blood Moor, S xp-gold-trace: the Den is not an XP source, only a point). The XP loss is therefore the 9.8 minutes (about 3,700 XP at the Cold Plains rate of 376/min, U: Cold Plains had little supply left, so the real gain is a part of that), not the quest.

Ready-to-run checklist (needs the game or the lab d2d; not run here):
1. On the lab d2d (devp6) in a Blood Moor game: `warps` (or `debug warps`) verb at the listed Den point (335,140 in this seed) and the neighbouring tiles; note rectangle x/y/w/h.
2. Same seed, `iv1` image (a42dba25) and devp6: walk a hero onto the point, record `level` before and after; expect iv1 fires, devp6 not (confirms (a)).
3. If the rectangle differs from the listed point, the fix is in nav `findGate` (use the rectangle), not in the planner.

## 4. Sacrifice

V from 352 `combat kill ... last_skill=Sacrifice` lines (62.5 % of 563) against 211 Attack. Skill timeline: Might at clvl 2 (09:59:19), Sacrifice at clvl 3 (10:03:38, one point), Prayer at clvl 4, Might at clvl 5. So Attack alone to minute 8.2, Sacrifice or Attack afterwards by the `combat.sacrifice*` rules (life at or above 70 %, at least 1 belt potion; `internal/combat/lifeprice.go`).

| measure | Attack only (until 10:03:38) | Sacrifice available | tag |
|---|---|---|---|
| Blood Moor fights | 41 fights, 89 kills | 71 fights, 135 kills | V `combat fight` |
| seconds per kill (fight) | 2.53 | 2.30 (Blood Moor), 2.60 (Cold Plains) | V |
| casts per kill | 1.53 | 1.16 (Blood Moor), 1.29 (Cold Plains) | V |
| `killtime` median total, Blood Moor | 2.5 s | 2.7 s | V |
| `killtime` mean total, Blood Moor | 5.4 s | 11.2 s | V; the mean doubles because the Den hunts and the re-sweeps are in the second half, not because of the skill |
| fight damage taken | 20 in 41 fights | 492 in 161 fights (427 in Cold Plains) | V `damage=`; Cold Plains hits harder, so not comparable |

Verdict: Sacrifice is used (62 % of last hits; on Fallen 194 of 300, Dark Hunter 51 of 95) and it reduces the casts per kill by about a quarter, but the fight seconds per kill do not move (2.5 vs 2.3 to 2.6, V): a fight is dominated by the approach walk and monster arrival, not by the swings, so the bot's kill rate is not damage-limited and Sacrifice cannot raise it. It did not hurt: 0 deaths, 0 chickens, 9 potions drunk (counter) in Cold Plains between 10:20 and 10:44 (the monster damage `dmgTaken` grows 69 to 439 over the same time), 8 `guard: drank healing` lines (for example at life 62 %, below the 70 % floor, so the Attack fallback was already on). The life price itself (8 % of damage dealt, about 2 to 3 life per swing at clvl 3 to 5, S knobs doc) is not logged per swing, so the share of the potion use caused by Sacrifice is U; no life-floor event is logged, and 37 % of last hits were still Attack (the floor was active in about a third of the fights, U). Neutral to mildly positive; keep it. A test that would settle the life cost: a 15 minute run with `combat.sacrifice` 0 on the same seed (ready-to-run, not run).

## 5. XP against runs 1 to 3

| run | kills | xp | xp per kill | area mix (kill lines) |
|---|---|---|---|---|
| 1 (11-12 02:18Z) | 528 | 17,032 | 32.3 | Blood Moor 211, Cold Plains 198, Den 225 (kill lines, 634 total) |
| 2 (11-12 17:11Z) | 445 | clvl 5 | | 78 entries, 49 shuttles |
| 3 (11-12 23:25Z) | 355 | clvl 5 | | potion starvation, 5 chickens |
| 4 (11-14 09:55Z) | 494 | 13,883 | 28.1 | Blood Moor 223, Cold Plains 340 (kill lines, 563 total), Den 0 |

Why fewer XP at similar kills (V arithmetic): run 4's mix is 40 % Blood Moor (22.0 XP per kill) and 60 % Cold Plains (28.9), run 1's was 33 % Blood Moor, 31 % Cold Plains and 36 % Den (Den: 22.3 XP per plain kill in the xp-gold-trace table, S; minions and uniques of a pack pay 130 to 310 each, S); run 1's Den pass 1 earned 5,607 XP in 14 minutes (400 per minute, V hour1 analysis), which run 4 replaced with 9.8 minutes of Den hunting (27 kills). The Den quest pays no XP (S), so it is not the quest bonus: it is the uniques and champions of the Den (Corpsefire and the boss packs) plus the kill supply of a third area. Run 4's log shows 2 uniques killed by name (Bishibosh, Dread Tongue the Hungry). Gap to run 1: 3,150 XP, of which the Den-hunt time (9.8 min at 231 XP/min = 2,300) is about three quarters (U).

## 6. Ranked sinks and missed gains (minutes per hour, fix, owner)

| rank | sink | minutes | proposed fix | kind | agent |
|---|---|---|---|---|---|
| 1 | The Den exit never fires: 4 hunts plus Cold Plains round trips; 1 skill point missed | 9.8 (V) | Live diagnosis first (checklist in section 3): compare `warps` and the trigger rectangle on iv1 and devp6. Meanwhile, cheap mitigation (code, bot): after the first "tile did not fire" on a KNOWN gate, mark the edge bad for the whole game (not 150 s doubling), and make the quest step `to-den` obey the area skip; a second knob-free guard: never walk Cold Plains to Blood Moor to Cold Plains for a Den plan entry that failed once this game | code + live check | d2-live for the checklist (needs i-slot and lab d2d), then d2-worker for nav; d2-decomp for the d2d trigger rule |
| 2 | Area supply: the tail of every area runs below 8 kills/min; Blood Moor swept three times (22 % of roam time), Cold Plains cleared at minute 49, no next area | about 12 (12 dead minutes, V; part of the 20.1 roam minutes) | (a) Re-roll the game (Save and Exit, 4 s, 15 s with re-entry) when the last 2 minutes yield under 4 kills and the area is more than 70 % explored: new knob `field.rerollBelowKillsPer2Min`, default off, measure on the sandbox. (b) Data: add Burial Grounds (Blood Raven, 181 XP) and Stony Field to the melee fallback when Cold Plains is `cleared` and the hero is clvl 5 with a full belt (level plan melee enter level in `internal/plandata`, U current value) | knob + data | d2-worker (planner/levelplan data), d2-live to measure |
| 3 | Walking to monsters: 993 s of approach legs, 156 s of stall pauses, 17 `end=error` fights, 34 out-of-reach fallbacks; devp5/6 halves the kill rate against the old d2d with the same bot (docs/fork-rebase-2026-11-13.md, V there) | 16.5 (overlaps ranks 1 and 2 by a few minutes) | Diagnose `combat ignoring target ... unreachable` (open card from the rebase doc): record monster positions vs the bot's grid in Cold Plains; if the monsters idle at range, let the Paladin wait for the pack at a choke instead of chasing (a combat-profile data field, name U) | diagnosis, then code/data | d2-decomp (d2d monster AI diff, docs only), d2-worker |
| 4 | Detours around the hero's own routes: town to Cold Plains on foot after the 10:46 save point (3.1 min, 28 Blood Moor kills at 9.5/min vs 15 in Cold Plains) although the waypoint was known since 10:06; two `[heal]` trips that healed nothing and bought one scroll | about 2.3 (V timestamps, U rate) | Prefer the known waypoint for town-to-field legs (`travelTo`, hero14 item 2: the wrong ban text can block waypoint walks); skip a `[heal]` trip when life is at or above the resume line and the potion plan wants nothing the gold can pay (rule in `internal/town/trip.go`, U exact condition) | code | d2-worker |
| 5 | Chores and junk: 2 sell trips (+227 gold net) at 80 to 85 % pack, 9 loot walk-backs | about 4.6 (town 195 s, loot walks 166 s) | Raise the pack-full trip trigger from 80 or 85 % to 90 % while no valuable item is in the pack (trip knob, value U); skip walk-backs (`loot_memory: going back`, 9) for plain items when the pack is above 70 %; the budget line already caps the trips at 3 per 10 minutes | knob | d2-worker |

Not a sink: the gold priority (4 `gold grab` mid-fight, 3 ledgers, 55 loot legs for 166 s; gold is not a lever, S first-hour doc), the belt (3 `belt fill` lines, one "drag did not land" at clvl 3 and two fills), Sacrifice (section 4), resting (no Prayer rest lines at all: 0 poverty stalls, 0 `rest` states; the hero never waited for life), Save and Exit points (2, each 3 s, V).

What the fixes could add (U, a model, not a promise): rank 1 and 4 give back about 12 minutes; at Cold Plains rates of the first 10 minutes of an area (15 to 19 kills/min, 28.9 XP) that is 12 x 400 = 4,800 XP, enough for clvl 6 (14,175, the run missed it by 292) but not clvl 7 (about 22,000 on the model, docs/first-hour.md 26,020 XP for the ideal route). clvl 7 needs rank 2 as well, because it is a supply question.

## Open points
- U: the exact cause of the Den tile not firing (section 3); needs the lab.
- U: the 14 % surplus of `combat kill` lines over the counter.
- U: whether Sacrifice's life drain ever pushed a fight below 70 % (no per-swing log).
- Proposed doc text for the coordinator (guard files not touched): a line for docs/decisions.md only if the user decides to change the Den rule; none proposed.
