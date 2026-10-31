# Death metrics computed from the real G1 soak logs (attempts 6-14)

Purpose: calibrate the death rule (plan-v2 section 9 candidates a-f). Data: bot logs `~/d2lod-live/soak24/runs/*.log` on the worker host (read-only, copied to a scratchpad), attempt map from `docs/g1-log.md`. Script: `docs/plan/death-data/` (`RUNS=<dir of logs> python3 analyze.py summary|detail|matrix`). Tags: V = computed by us from the logs, U = unverified inference. Attempt 1 (spiral baseline) is added as a reference row; attempts 2-5 were not analysed (environment failures or 0 deaths).

## 1. Definitions and data quality (read first)

| Item | Definition used | Quality |
|---|---|---|
| Death | one `"msg":"character died"` line (area id, killer, corpseValue). Cross-check: d2soak `samples.jsonl` lifetime `deaths` counter deltas match (a10 +2, a11 +5, a12 +3) V | good. The 34 `"msg":"death"` lines in the a11/a12 logs are a history replay at start (old corpse records), NOT new deaths: excluded |
| Kills | sum of positive deltas of minute-line `kills` (a drop is treated as a counter reset) | UNDERCOUNTS: d2soak lifetime kills delta says a11 552 vs 464 here, a12 102 vs 64. No verdict changes (a12 still FAIL) |
| Death-recovery time share | S1 = duration of `state` lines whose activity is in R = {`Recovering after death`, `Restocking after death`, `Retrieving the corpse` (only ever seen in a `minute` step, never as a state line), `Chores in town (poverty rebuild after failed retrieval)`}. S2 = union of windows [death line, first `combat fight` line or `area_change` to a non-town area]. S3 = S1 plus `Recovery: farming gold` (the BROKEPHASE farm) | S1/S2 good. `Waiting: <area> banned for N more min` (230 state lines) is NOT counted: in a14 it occurs with 0 deaths (carried-over bans) |
| Rolling hour | sliding 60 min window, 1 min step, per attempt. Attempts shorter than 60 min (a1, a7, a9, a10, a12, a13) use ONE window with a fixed 60 min denominator (conservative: unlogged time counts as not recovering) | a14 is still running: log snapshot to 01:11 (74 min) |
| Gold lost per death | last `minute` gold before the death minus first `minute` gold after (each up to 60 s stale). Minute-line gold is INVENTORY gold; the bot withdraws stash gold after a death (a11: 0 on the minute line, 1039 in `shop: potion plan` 30 s later), U | approximate. `corpseValue` is a gear value estimate, not gold (a12 d2: corpseValue 296, gold lost 1125) U |
| Gold earned | sum of positive minute-line gold deltas, excluding the 2 minute lines after each death (stash-withdraw jumps); includes vendor sales | rough |
| Belt potions at death | `beltHP` of the last `shop: visit` (post-shop) minus `combat potion kind=healing` lines since. The minute-line `potions` field is 0 in every line of every attempt: unusable | belt reading 105-493 s old; a12 d1 and all a1 deaths have no earlier shop line: unknown |
| clvl gain | `clvl` increase between minute lines | good |

## 2. Per attempt (whole attempt, actual span)

| Att | Start-end | Min | Deaths | Deaths/h | Kills | Deaths/1000 kills | S1 % | S2 % | S3 % | Gold earned | Gold lost | Lost/earned |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 (ref) | 12:32-12:43 | 12 | 4 | 20.8 | 15 | 267 | 5.5 | 12.6 | 5.5 | 0 | 0 | n/a |
| 6 | 14:39-16:45 | 125 | 0 | 0 | 1351 | 0 | 0 | 0 | 0 | 7698 | 0 | 0 |
| 7 | 17:10-17:46 | 35 | 0 | 0 | 393 | 0 | 0 | 0 | 0 | 1344 | 0 | 0 |
| 8 | 17:46-19:52 | 126 | 3 | 1.4 | 774 | 3.9 | 2.7 | 3.1 | 2.7 | 961 | 347 | 36 % |
| 9 | 19:53-20:28 | 36 | 0 | 0 | 62 | 0 | 0 | 0 | 0 | 28 | 0 | 0 |
| 10 | 20:29-21:09 | 41 | 2 | 2.9 | 265 | 7.5 | 6.1 | 7.7 | 53.8 | 162 | 17 | 10 % |
| 11 | 21:10-22:32 | 82 | 5 | 3.6 | 464 | 10.8 | 2.3 | 4.0 | 57.5 | 1163 | 379 | 33 % |
| 12 | 22:33-23:11 | 38 | 3 | 4.7 | 64 | 46.9 | 35.0 | 33.6 | 69.2 | 1205 | 1316 | 109 % |
| 13 | 23:11-23:42 | 31 | 0 | 0 | 201 | 0 | 0 | 0 | 58.6 | 2740 | 0 | 0 |
| 14 (running) | 23:56-01:11 | 74 | 0 | 0 | 557 | 0 | 0 | 0 | 11.5 | 2330 | 0 | 0 |

(V.) The a13 log ends at 23:42 (the user stopped it at 23:50). S3 is high in a10-a13 and a14 hour 2 purely from `Recovery: farming gold` (a13 and a14 have 0 deaths): S3 is NOT a death measure.

## 3. Per attempt hour (non-overlapping hours from the attempt start)

| Att | Hour | Min | Deaths | Kills | Deaths/1000k | S1 % | S2 % | Gold earned | Gold lost | clvl+ |
|---|---|---|---|---|---|---|---|---|---|---|
| 6 | 1 | 60 | 0 | 599 | 0 | 0 | 0 | 5531 | 0 | yes |
| 6 | 2 | 60 | 0 | 690 | 0 | 0 | 0 | 2167 | 0 | no |
| 8 | 1 | 60 | 2 | 346 | 5.8 | 2.5 | 2.9 | 560 | 324 | yes |
| 8 | 2 | 60 | 1 | 385 | 2.6 | 3.2 | 3.7 | 399 | 23 | no |
| 10 | 1 | 40 | 2 | 265 | 7.5 | 6.1 | 7.7 | 162 | 17 | no |
| 11 | 1 | 60 | 5 | 343 | 14.6 | 3.2 | 5.4 | 1085 | 379 | yes |
| 11 | 2 | 22 | 0 | 121 | 0 | 0 | 0 | 78 | 0 | no |
| 12 | 1 | 38 | 3 | 64 | 46.9 | 35.0 | 33.6 | 1205 | 1316 | no |
| 14 | 1 | 60 | 0 | 433 | 0 | 0 | 0 | 1745 | 0 | yes |
| 14 | 2 | 14 | 0 | 124 | 0 | 0 | 0 | 585 | 0 | no |

Attempts 1, 7, 9, 13 are single partial hours with the section 2 numbers. Short hours use their own length as denominator.

Worst sliding 60 min window per attempt (the values the rules are evaluated on):

| Att | Max deaths/60 min | Max S1 % | Max S2 % | Max deaths/1000k (kills >= 100) | Max lost/earned (earned >= 100) |
|---|---|---|---|---|---|
| 1 | 4 | 1.1 | 2.4 | inf (15 kills) | 0 |
| 6, 7, 9, 13, 14 | 0 | 0 | 0 | 0 | 0 |
| 8 | 3 (from 17:55) | 5.7 | 6.6 | 8.6 | 84 % |
| 10 | 2 | 4.2 | 5.2 | 7.5 | 10 % |
| 11 | 5 (from 21:10) | 3.2 | 5.4 | 18.7 | 61 % |
| 12 | 3 | 22.3 | 21.4 | inf (64 kills) | 109 % |

(a1 and a12 shares use the fixed 60 min denominator, so a1 shows 2.4 % S2 although it was 12.6 % of its 12 real minutes.)

## 4. Every death

| Att | Time | Area (id) | Killer | Gold before -> after | corpseValue | Sec to back in field | Belt healing potions (est.) | Since previous death |
|---|---|---|---|---|---|---|---|---|
| 1 | 12:35:07 | 10 | Vile Hunter | 0 | 0 | 44 | unknown | |
| 1 | 12:37:47 | 10 | Vile Hunter | 0 | 0 | 32 | unknown | 160 s |
| 1 | 12:40:17 | 10 | Vile Hunter | 0 | 0 | 6 | unknown | 150 s |
| 1 | 12:42:18 | 10 | Carver | 0 | 0 | 6 | unknown | 121 s |
| 8 | 18:17:58 | 10 | Vile Hunter | 324 -> 0 | 728 | 98 | 4, none used | |
| 8 | 18:22:24 | 10 | Vile Hunter | 0 | 0 | 6 | 2, none used | 266 s |
| 8 | 18:54:49 | 9 | Fallen | 23 -> 0 | 0 | 133 | 1, none used | 32 min |
| 10 | 20:54:50 | 9 | Dark Ranger | 3 -> 0 | 0 | 89 | 0 (1 used 6 s before) | |
| 10 | 21:06:37 | 9 | Soul Cloud the Sharp | 14 -> 0 | 80 | 100 | 0 | 12 min |
| 11 | 22:02:23 | 7 | Dark Stalker | 86 -> 0 | 778 | 98 | 2, none used | |
| 11 | 22:05:02 | 26 | Steel Grumble (boss) | 293 -> 0 | 352 | 49 | 2, none used | 159 s |
| 11 | 22:07:02 | 26 | Steel Grumble | 0 | 0 | 12 | 2, none used | 120 s |
| 11 | 22:08:24 | 7 | Dark Stalker | 0 | 0 | 6 | 2, none used | 82 s |
| 11 | 22:09:38 | 7 | Dark Stalker | 0 | 0 | 31 | 2, none used | 74 s |
| 12 | 22:37:39 | 26 | Dark Lancer | 191 -> 0 | 456 | 47 | unknown | |
| 12 | 22:56:47 | 26 | Dark Stalker | 1125 -> 0 | 296 | 36 | 2, none used | 19 min |
| 12 | 22:59:44 | 7 | Dark Stalker | 0 | 232 | never (stuck in town 11+ min, ended by user) | 2, none used | 177 s |

(V. Area names from the death lines: 7 Tamoe Highland, 26 Monastery Gate (on the Barracks quest walk), 10 Underground Passage Level 1; area 9 is Cave Level 1 by inference from the area changes, U. "Sec to back in field" is often 6-12 s because the bot takes the waypoint straight back to Black Marsh and walks the same route again.)

Findings from this table:
- Median seconds from death to back in the field is about 40 s over 16 values; max 133 s except the a12 stuck case (V). Recovery itself is fast; the cost is that the bot immediately re-walks the killing route.
- 10 of 13 deaths (a8-a12) had 1-4 healing potions in the belt and no `combat potion kind=healing` line since the last shop: potions went unused (the POTIONPOLICY and DEATHAUTOPSY fixes). Only a10's two deaths had an empty belt; a12 d1 is unknown. A missing log line is not proof that no potion was drunk (U).
- Every death with inventory gold took all of it (8 of 8: 324, 23, 3, 14, 86, 293, 191, 1125 went to 0). Corpse retrieval never recovered anything (5 `corpse attempt` lines, all `ok:false`) V.

## 5. XP lost per death in Normal

On the 13 deaths of a8-a12 plus a1's 4, the `xp` on the minute line after the death is never lower than before (deltas 0 to +404, which is XP earned in that minute). The only xp drop in all attempts is a6 15:45 (46436 to 46091) during the injected `Game lost` restart. The `xpLost` field counts progress since the last save at a restart or failsafe (a8 68 after the kill injection), not deaths: it did not change across any death. Conclusion: Normal death costs no XP in our logs; it costs the carried inventory gold plus the corpse items if not retrieved (V from logs). That classic D2 charges an XP penalty only in Nightmare/Hell (5 %/10 %) and gold in all difficulties is U from memory.

## 6. Repeat-site, spirals, streaks

| Att | Repeat-site pairs (same area, <= 30 min) | 3 deaths in 20 min | 3 in 30 min, no clvl gain | Max deaths/60 min |
|---|---|---|---|---|
| 1 | area 10 x4 (3 pairs) | yes | yes | 4 |
| 8 | area 10 at 18:17 and 18:22 | no | no (3rd death 37 min after the 1st) | 3 |
| 10 | area 9 at 20:54 and 21:06 | no | no (2 deaths) | 2 |
| 11 | area 26 x2, area 7 x3 (3 pairs) | yes (22:02-22:07, 4.6 min) | yes | 5 |
| 12 | area 26 at 22:37 and 22:56 | NO (22:37:39 to 22:59:44 = 22.1 min) | yes (22.1 min, clvl 11 throughout) | 3 |
| 6, 7, 9, 13, 14 | none | no | no | 0 |

Death-spiral windows (3 deaths in 30 min without clvl gain): a1 (12:35-12:42), a11 (22:02-22:07), a12 (22:37-22:59). None in a8, a10, a14.

## 7. What each candidate rule says (V; worst sliding 60 min window)

"Truth" is what we know happened (g1-log.md).

| Att | Truth | (a) 3/20 min or >= 5/h | (b) share S2 (S1): WARN > 5, FAIL > 10 | (c) deaths/1000k: WARN > 5, FAIL > 15 | (d) lost > 25 % of earned | (e) repeat site in 30 min | (f) 3/30 min no clvl gain or > 4/h |
|---|---|---|---|---|---|---|---|
| 1 | spiral, FAIL | FAIL | ok 2.4 % (fixed denominator; 12.6 % real would FAIL) | FAIL | n/a (0 gold) | trigger | FAIL |
| 6, 7, 9, 13, 14 | clean (a13: 0 deaths, not attacking) | ok | ok | ok | ok | ok | ok |
| 8 | 3 deaths in 2 h, should NOT fail | ok | WARN 6.6 (S1 5.7) | WARN 8.6 | FAIL (324 lost of 386 earned in that window; 36 % whole run) | trigger | ok |
| 10 | 2 deaths, empty belt | ok | WARN 5.2 | WARN 7.5 | ok 10 % | trigger | ok |
| 11 | death chain, should FAIL | FAIL | WARN 5.4 only (S1 3.2) | FAIL 18.7 (whole run 10.8 = WARN) | FAIL 61 % | trigger | FAIL |
| 12 | death loop, should FAIL | ok (misses by 2 min; 3 deaths < 5/h) | FAIL 21.4 (S1 22.3) | FAIL (3 deaths, 64 kills) | FAIL 109 % | trigger | FAIL |
| 14 hour 1 | 0 deaths, should NOT fail | ok | ok | ok | ok | ok | ok |

By rule:
- (a) streak 3/20 or 5/h: FAILs a1 and a11, MISSES a12 (the third death came 2.1 min after the 20 min window). Correct for a8 and a14.
- (b) time share: FAILs only a12. MISSES a11 (S2 5.4 %, WARN) because each a11 death costs 6-98 s: the bot re-walks the route at once, so time in recovery stays small while deaths pile up. Time share measures stuck-in-recovery, not chains. WARNs a8 and a10 (acceptable, not FAIL).
- (c) deaths per 1000 kills with a kills floor of 100: FAILs a1, a11, a12, WARNs a8 and a10: matches truth. a11 FAILs only in the worst 60 min window (18.7; whole run 10.8 is WARN). Needs the floor (0 kills gives infinity), and the kills counter undercounts (section 1).
- (d) gold lost share: FAILs a8 (false positive: tiny earnings), a11, a12. Unusable as FAIL without an absolute floor (lost >= 500 gold would fail only a12). Report/WARN only.
- (e) repeat site: fires in a1, a8, a10, a11, a12, i.e. every attempt that had two or more deaths. Fine as an action (area change plus planner dump), a false positive as a FAIL. In a8 it fires on the same Underground Passage walk that spiralled a1.
- (f) tightened streak: FAILs exactly a1, a11, a12 and passes a8, a10, a14: the best match to what we know.
- S3 (adding `Recovery: farming gold`) would show 30-69 % in a10, a11, a13 and 60 % in a14 hour 2 (0 deaths): never include that state in the death share.

## 8. Recommendations for the death rule (draft)

1. Primary FAIL: (f), 3 deaths in 30 min with no clvl gain or more than 4 in 60 min. It separates all attempts here correctly. The plan-v2 streak (a) misses a12 by two minutes; widen 20 to 30 min.
2. Secondary FAIL: (c) deaths per 1000 kills > 15 with kills >= 100 in the window (agrees with (f), adds a rate view for long clean runs); WARN > 5.
3. Keep time share (b) FAIL > 10 % as an independent "stuck in recovery" check (it alone catches a12's 11 minute stall), defined with S1/S2 only.
4. Make (e) an action, not a FAIL. Candidate earlier cut: a death within 150 s of the previous one (a1 160/150/121 s, a11 159/120/82/74 s, a12 177 s, a8 266 s, a10 707 s) should stop the bot re-walking the same route; 150 s is borderline on this data, 180 s separates a1/a11/a12 from a8/a10.
5. (d) report only. Log per death: area, killer, belt healing potions, gold at death, seconds to field (deaths.jsonl from DEATHAUTOPSY should carry them). Fix before these become gates: minute-line `potions` is always 0, and the `kills` counter undercounts and resets.

## 9. Open points

- U: belt potions are estimated from shop lines; no per-death belt snapshot exists in the logs.
- U: whether the gold goes to the corpse or is destroyed (retrievals all failed, gold reads 0 afterwards).
- a14 is a snapshot to 01:11; recompute when it ends. Attempts 2-5 and the per-run d2soak `report.json` were not used (deaths counter cross-check only).
