# noareameasure: XP per area, the picker's excluded set and the kit at the Gate 2 seed 13 block

> status: corrected 2026-11-04 per the SME review (docs/noareacards2.md): section 5 band 5.08% (not 1.95%), Cave 2 traced 1.4 XP/kill, the Catacombs 2 figure marked not a forecast, the economy wording limited to the 01:06 flood, the tome drop tagged U.

Tags: V = verified (computed from the logs, or a log line, say which), S = secondary (code read, not run), U = unverified.
Docs only. Nothing here was run live. It measures; it changes no code and takes no policy decision (it feeds `docs/noarea-policy.md`).
Evidence (read only): the minute logs of Gate 2 seeds 11, 12, 13 (committed, thinned for seed 13, in `docs/probes/soak-logs/`:
`gate2-s11-20261103-1954-g2a`, `gate2-s12-20261103-2210-g2v`, `gate2-s13-20261103-2211-g2v`) and `bot-0.log` of seed 13 on the
worker host (line numbers below are lines of that file).

Method (V): a "minute" is one `minute` line (a VIRTUAL minute, `internal/d2dsoakstat/invariants.go:55`); an hour is 60 such lines. The
`time` stamps in the lines are wall clock (EST) and are what "23:35" means below. For each pair of consecutive minute lines where both
have clvl >= 13, the XP delta and the `kills` delta of the second line are credited to its `area`. Negative XP deltas (XP lost on a
death, seen once: seed 13 area 3, -1370) are left out of the XP sum. XP/h = XP / minutes * 60; XP/kill = XP / kills of the same
lines. Transit minutes (a walk between two areas) are credited to the area the line says, so thin areas are noisy: n < 30 minutes is
marked "thin". Areas are numbered as in `levels.txt`: 1 Rogue Encampment (town), 2 Blood Moor, 3 Cold Plains, 4 Stony Field,
5 Dark Wood, 6 Black Marsh, 7 Tamoe Highland, 8 Den of Evil, 9 Cave 1, 10 Underground Passage 1 (S: id not in plan data), 13 Cave 2,
17 Burial Grounds, 18 Crypt, 19 Mausoleum, 20 Forgotten Tower, 26 Monastery Gate (internal/task/route.go comment).

## 1. XP per hour and per kill at clvl >= 13 (V, from the minute lines)

Columns: area, minutes, XP, kills, XP/h, XP/kill. Ranked by XP/h. Town (1) and areas with 0 XP are listed last.

### Seed 11 (run g2a; clvl 13 reached 21:20, max clvl 16)
| area | min | XP | kills | XP/h | XP/kill |
|---|---|---|---|---|---|
| 7 Tamoe Highland (thin) | 18 | 10740 | 132 | 35800 | 81.4 |
| 26 Monastery Gate (thin) | 4 | 862 | 9 | 12930 | 95.8 |
| 13 Cave 2 | 310 | 45754 | 3721 | 8856 | 12.3 |
| 6 Black Marsh | 99 | 11367 | 627 | 6889 | 18.1 |
| 18 Crypt | 142 | 13031 | 784 | 5506 | 16.6 |
| 4 Stony Field | 258 | 18529 | 1854 | 4309 | 10.0 |
| 5 Dark Wood | 101 | 6941 | 965 | 4123 | 7.2 |
| 1 town | 214 | 11302 | 790 | 3169 | 14.3 |
| 9 Cave 1 | 264 | 13038 | 1603 | 2963 | 8.1 |
| 3 Cold Plains | 108 | 3452 | 321 | 1918 | 10.8 |
| 10 (thin) | 13 | 397 | 28 | 1832 | 14.2 |
| 19 Mausoleum (thin) | 3 | 21 | 9 | 420 | 2.3 |
| 8 Den (thin) | 12 | 73 | 71 | 365 | 1.0 |
| 17 Burial Grounds (thin) | 29 | 174 | 14 | 360 | 12.4 |
| 2 Blood Moor | 63 | 83 | 225 | 79 | 0.4 |

### Seed 12 (run g2v; clvl 13 reached 23:35, max clvl 16)
| area | min | XP | kills | XP/h | XP/kill |
|---|---|---|---|---|---|
| 7 Tamoe Highland (thin) | 25 | 8371 | 158 | 20090 | 53.0 |
| 6 Black Marsh | 157 | 24189 | 1417 | 9244 | 17.1 |
| 5 Dark Wood | 162 | 21520 | 1483 | 7970 | 14.5 |
| 4 Stony Field | 791 | 49195 | 3784 | 3732 | 13.0 |
| 10 (thin) | 19 | 1097 | 45 | 3464 | 24.4 |
| 13 Cave 2 | 76 | 4336 | 885 | 3423 | 4.9 |
| 1 town | 180 | 8011 | 594 | 2670 | 13.5 |
| 19 Mausoleum (thin) | 37 | 1462 | 228 | 2371 | 6.4 |
| 18 Crypt | 252 | 9796 | 901 | 2332 | 10.9 |
| 9 Cave 1 | 148 | 5332 | 799 | 2162 | 6.7 |
| 17 Burial Grounds | 44 | 1386 | 198 | 1890 | 7.0 |
| 26 Monastery Gate (thin) | 5 | 132 | 4 | 1584 | 33.0 |
| 8 Den (thin) | 11 | 272 | 95 | 1484 | 2.9 |
| 3 Cold Plains | 122 | 1852 | 679 | 911 | 2.7 |
| 2 Blood Moor (thin) | 19 | 151 | 81 | 477 | 1.9 |

### Seed 13 (run g2v; clvl 13 reached 22:53, max clvl 16)
| area | min | XP | kills | XP/h | XP/kill |
|---|---|---|---|---|---|
| 6 Black Marsh | 59 | 30616 | 560 | 31135 | 54.7 |
| 20 Forgotten Tower (thin) | 1 | 335 | 14 | 20100 | 23.9 |
| 13 Cave 2 | 119 | 27437 | 1760 | 13834 | 15.6 |
| 5 Dark Wood | 46 | 9326 | 361 | 12164 | 25.8 |
| 7 Tamoe Highland (thin) | 7 | 516 | 17 | 4423 | 30.4 |
| 4 Stony Field | 218 | 13969 | 1225 | 3845 | 11.4 |
| 9 Cave 1 | 168 | 10000 | 1172 | 3571 | 8.5 |
| 10 (thin) | 3 | 156 | 19 | 3120 | 8.2 |
| 1 town | 242 | 11967 | 474 | 2967 | 25.2 |
| 3 Cold Plains | 121 | 4179 | 728 | 2072 | 5.7 |
| 17 Burial Grounds | 78 | 2470 | 273 | 1900 | 9.0 |
| 18 Crypt | 161 | 4821 | 765 | 1797 | 6.3 |
| 19 Mausoleum | 21 | 299 | 97 | 854 | 3.1 |
| 8 Den (thin) | 16 | 156 | 115 | 585 | 1.4 |
| 2 Blood Moor | 558 | 18 | 34 | 1.9 | 0.5 |
| 26 Monastery Gate (thin) | 1 | 0 | 0 | 0 | - |

Reading it (V for the numbers, U for the reasons):
- Black Marsh and Dark Wood are at or near the top of every run among the areas with a real sample (n >= 46): 6.9k-31k XP/h and
  4.1k-12k XP/h. Cave 2 is the other leader (3.4k-13.8k). Crypt, Mausoleum, Burial Grounds, Cold Plains, Blood Moor, the Den
  and Cave 1 are at the bottom (0.1k-3.6k) in all three runs.
- Seed 13 spent 558 minutes at clvl >= 13 in Blood Moor for 18 XP (1.9 XP/h). That block is the dead stretch after the block of
  section 2 (U: it was not traced here).
- XP/kill here is 5-25 for the open fields and caves at clvl 13+. The repo's traced plain-monster mean for clvl 15 is below 1
  XP/kill for the same areas (section 5), so the bot's kills are mostly not "plain" ones: packs, champions and uniques carry the
  XP (U: not separated in the minute lines). Seed 13's Black Marsh 54.7 XP/kill shows how large that share can be.
- XP by clvl (V; minutes, XP, XP/h), seed 13: clvl 13 71 min 27196 22983/h; 14 248 min 35678 8632/h; 15 632 min 45327 4303/h;
  16 867 min 8064 558/h. Seed 11: 15: 747 min 3518/h; 16: 378 min 4839/h. Seed 12: 15: 1069 min 2454/h; 16: 553 min 3283/h.
  Seed 13 at clvl 16 earned a tenth of what seeds 11 and 12 did, in more minutes.
- Seed 13 at clvl 15 only (V, 632 min): Cave 2 13066 XP/h (109 min), Stony Field 3996, town 2978, Cave 1 2435, Cold Plains 1566,
  Crypt 1340, Burial Grounds 1233, Den 677, Mausoleum 540, Blood Moor 0. Black Marsh and Dark Wood do not appear at clvl 15 at all
  (see section 4).

## 2. Seed 13: the picker's excluded set at the `no eligible reachable` lines

Where the lines are (V, `grep -n "no eligible reachable"` of bot-0.log): the first block is three lines at 23:36:00.76-.84
(41596, 41597, 41601; clvl 15, step `level.17`), nothing between them and the flood, which starts at 01:06:54 (86940, clvl 16) as
`area gate refused the walk` lines carrying the same text.

### 2.1 First block, 41596 at 23:36:00.76 (clvl 15, in town after a failsafe restart)
What happened just before (V): the character stood in Burial Grounds with an empty tome (see 3), the walk home was refused
(`the click would cross the trigger area of an unwanted exit`, 41473-41585), the failsafe skipped Burial Grounds `nav-stuck`
(41504) and then did Save and Exit (41590), re-entering in Rogue Encampment at 23:36:00 (41592-41593). The fresh game empties the
cleared set (S: `syncGames`, internal/bot/task.go:875), so nothing was excluded as "cleared" at 41596 (Burial Grounds had been
reported cleared at 41464 in the old game).

Excluded areas, with reason, in the order of the log (V = the line; the set itself is S: assembled from goal.go:862-878, 126-141):
| area | reason | length / text | log line (time) |
|---|---|---|---|
| 13 Cave 2 | ban `unreachable` | 150 s, then 300 s | 40953 (23:35:21), 41188 (23:35:36) |
| 4 Stony Field | ban `unreachable` | 150 s, then 300 s | 40962 (23:35:21), 41196 (23:35:36) |
| 18 Crypt | ban `nav-stuck` | 150 s | 41324 (23:35:48) |
| 9 Cave 1 | ban `unreachable` | 150 s | 41348 (23:35:48) |
| 8 Den of Evil | ban `unreachable` | 150 s | 41354 (23:35:48) |
| 3 Cold Plains | ban `unreachable` | 150 s | 41360 (23:35:48) |
| 2 Blood Moor | ban `unreachable` | 150 s | 41366 (23:35:48) |
| 17 Burial Grounds | ban `nav-stuck` | 150 s (dashboard: "banned for 3 more min", 41598) | 41504 (23:35:58) |
| 7 Tamoe, 26-37 (Monastery Gate to Catacombs 4) | route guard | `blocked monastery (no town portal)` | 41434 (23:35:56) |
| 20-25 Forgotten Tower and Tower Cellars | tower guard | `blocked tower (no merc, clvl 15 < 16)` | 41434 |
| every area outside {2,3,4,8,9,13,17,18} (this includes 5 Dark Wood, 6 Black Marsh, 19 Mausoleum) | merc gold farm | `merc: gold farm on need=2145 gold=2022 clvl=15`, on since 23:35:06, still on until 00:31:44 | 40713, 75501 |
| 19 Mausoleum, 17, 18 | thin kit | steps marked `xp_fallback_full_kit` are skipped in the XP fallback without a full kit (town portal + belt) | S: plan.go:407, 454 |

The last row's data are S (code read); the gold farm rule is S for its effect (goal.go:126-132: `MercGoldFarm` set, Normal, act 1)
and V for being on (the two log lines). Cleared at 41596: none (fresh game, S). Everything the farm rule allows (2, 3, 4, 8, 9, 13, 17,
18) was banned, so the result was empty: 8 of 8 allowed areas were under a ban of 150-300 s at that moment. The old doc
`docs/noarea-policy.md` section 2 said "Burial Grounds was the only other entered area not yet banned"; the gold-farm rule is the
piece it did not have: Dark Wood and Black Marsh were not banned, they were not allowed.
Relief (V): at 23:36:00 the idle-break lifted the bans of Cold Plains and Cave 1 (41606, 41607) and the plan picked Cave 1 (41605,
`past every window, last eligible (leaves at 10)`). Crypt and Burial Grounds were lifted at 23:36:57 (42235, 42236).

### 2.2 First lines of the flood, 86940 at 01:06:54 (clvl 16)
What the log shows (V): gold farm was OFF since 00:31:44 (75501: clvl 16 over the farm cap), so it was not the farm rule. Bans live or
just issued: Blood Moor `nav-stuck` 150 s (86931, 01:06:54; again 87043, 01:06:59), Black Marsh `nav-stuck` 150 s (86702, 01:05:19).
Route guard `blocked monastery (0 healing potions < 4); blocked tower (0 healing potions < 4)` (86849, 01:06:17). The state at that
minute (V, minute line 86943): town, clvl 16, gold 26, stash gold 0, healing potions 0 in the belt, 3 mana potions, `Recovery`
state, 131 restarts, XP lost to deaths 1375. Dark Wood: the plan had picked it at 01:05:51 (86790) and no ban or cleared line for it
follows in the log. Why Pick returned nothing at 86940 is not in the log (no `level plan` or `area skipped` for Dark Wood, Mausoleum,
Cave 2, Cave 1, Cold Plains or Stony Field in the seconds before; the kit gate and recovery path were also active: U).
The tome state at the flood was not examined.

## 3. Seed 13 kit at 23:35 EST on 2026-11-03 (V)
Sources: the minute lines (fields: `gold` carried, `stashGold`, `potions` = healing potions in the belt, `mana` = mana potions in the
belt, `packHP`/`packMP` = potions in the inventory, observe.go:961) and the `inv-tp-dump` lines of bot-0.log (`counted=` is the tome charge count).

| time | area | carried gold | stash gold | belt healing | belt mana | pack HP/MP | tome charges (inv-tp-dump) |
|---|---|---|---|---|---|---|---|
| 23:35:01 | 13 Cave 2 | 1074 | 1176 | 3 | 0 | 0 / 1 | 6 at 23:34:54, 5 at 23:35:02 |
| 23:35:03 | 13 Cave 2 | 810 | 1186 | 4 | 4 | 0 / 0 | 5 |
| 23:35:09 | 1 town | 430 | 1186 | 4 | 4 | 1 / 0 | 4 (23:35:06, 23:35:09) |
| 23:35:29 | 1 town | 370 | 1186 | 6 | 1 | 0 / 0 | 1-2 (23:35:29) |
| 23:35:52 | 1 town | 266 | 1186 | 6 | 2 | 0 / 0 | 0 from 23:35:43 (`hastp=false`) |

So at 23:35: gold 430-1074 carried plus 1186 in the stash (about 1600-2250 in all; the farm check read 2022 at 23:35:06),
3-6 healing and 0-4 mana potions in the belt, and a tome that went from 6 charges at 23:34:54 to 0 at 23:35:43: it was used up
in 50 s (U: the dumps do not say what consumed each charge; the town trips of the same minutes are the likely users). After it hit 0 the bot could not portal home (41468: `no portal source (empty)`).
The tome was refilled to 3 charges at 23:36:07 (inv-tp-dump, area 1). Intermediate dumps at 23:35:08, :10, :23, :38 read `counted=0`
with address `0xffffffff` (a transient read of the item, verdict dumps; not a real zero).
Cross-check: `route guard: blocked monastery (3 healing potions < 4)` at 23:35:00 and `(2 healing potions < 4)` at
23:35:02 agree with the minute lines (belt healing 3-2 plus pack).
Gold around the trips (S, SME, not re-read here): 2022 at 23:35:06 (V, line 122), then 1616 within 3 s after the chores. The revive
order is V in code: `mercTown.Chores` runs `Townsman.Chores` and then `ReviveStep` (cmd/d2bot/mercwire.go:664-673), so the revive comes
after the town chores of its visit; whether shopping sits inside `Townsman.Chores` is U.
> NOTE: correction unverified for the 1616 figure: its log line is not cited in the doc's own sources and was not re-read (the worker-host
> bot-0.log path is not in this job's scope). The U tag on the tome drop (line 157) was already present.

## 4. Dark Wood and Black Marsh at the block (the SME's question)
- Banned? No. The last Black Marsh ban lifted at 23:33:58 (39812, `area ban lifted`; `level plan: clvl 14 -> area 6 Black Marsh (in
  window 10-14)`, 39811) and no Black Marsh ban appears again until 01:05:19 (86702). The last Dark Wood ban lifted at 22:5x (20589); the
  earlier `area skipped` for it was 300 s `unreachable` (19138). No new ban for either before 41596 (V: full-log grep of `area skipped|area ban lifted`).
- Eligible? No, by the picker's own rules (S): from 23:35:06 the merc gold farm was on, and it excludes every Normal act 1 area not in
  `merc.FarmAreas` = {2,3,4,8,9,13,17,18} (internal/merc/farm.go:10, internal/bot/routeguard.go:126-132). Dark Wood (window 9-13, leaves
  13) and Black Marsh (10-14) are not in that set. They were also past their window at clvl 15 (leave 13 and 14, a gap of 2 and 1 levels, inside the floor of 4).
  At clvl 14 and 23:33:58 the farm was off (39809) and the picker chose Black Marsh. So "banned or eligible" has a third answer: not allowed.
> NOTE: correction unverified for the "earlier economy cause" wording: the phrase is not in this doc (no match for "economy"). What the doc does say is consistent with the correction: section 2.1 gives the farm rule as the 23:35 cause (V), and the economy numbers (gold 26, 0 healing potions) belong to the 01:06 flood of section 2.2 only (V).
- Cost (V): they are the two best areas by XP/h in seed 13 at clvl >= 13 (section 1: Black Marsh 31135, Dark Wood 12164 vs Cave 2 13834).
  At clvl 15 the bot made 0 minutes in either (section 1). The pick of Black Marsh at 23:33:58 lasted until the clvl-15 level-up at
  23:34:11 sent it to Mausoleum, which paid 2.3 XP/kill.

## 5. What a Catacombs 2 stay pays, against the stay the bot made
Catacombs 2 = area 35 (plan: enter 13, leave 17; internal/plandata/progression-normal.json). It is on the guarded Monastery road
(route guard 26-37), so at 23:35 it was excluded by the guard (no town portal, then potion shortfalls) and by the gold farm. The run
has no minute in area 35 in any of the three seeds (V: not in the minute lines), so there is no measured pace for it: "not in the log".

| quantity | the stay the bot made | Catacombs 2 | tag |
|---|---|---|---|
| window | seed 13, 23:34:15-23:37:53, clvl 15, 61 minutes: Cold Plains 4 min, Mausoleum 9, Cave 1 11, Cave 2 5, town 9, Crypt 19, Burial Grounds 5 | clvl 15 inside the window 13-17 | V (minutes), plan data |
| XP gained | 1446 (176166 to 177612) | not in the log | V |
| XP per hour | 1422 | not in the log | V |
| kills per hour | 262 | not in the log | V |
| XP per kill | 5.4 | traced plain mean 97.2 at monster level 11 (internal/bot/recoveryxp.go:68 row 35, gap share 100% at clvl 15, d=4) | V for 5.4; trace value S (plain monsters only, bosses and packs excluded by the trace, 12 seeds of rooms) |
| XP per hour at the same 262 kills/h | 1422 | 97.2 x 262 = about 25500 (arithmetic only, not a forecast) | U (assumes the kill pace of the stay, and the traced mean, both unmeasured for Catacombs) |
| XP per hour at 100 kills/h | n/a | about 9700 | U (same assumption, a lower pace) |
| comparator, whole of seed 13 at clvl 15 (632 min) | 4303 XP/h | | V |
| comparator, Cave 2 at clvl 15 (seed 13, 109 min) | 13066 XP/h, 14.6 XP/kill | traced 27.2 x 5.08% = 1.4 XP/kill (d = 2 - 15 = -13, band 5.08, recoveryxp.go:50) | V for 13066 and 14.6; trace V arithmetic |

Caveat (S): the traced table underestimates what the bot actually earns per kill in every area where both exist at clvl 15 (Cave 2
1.4 traced vs 14.6 measured, Stony Field 1.9 vs 12.4, Burial Grounds 2.0 vs 7.8, Cave 1 1.8 vs 7.1, Cold Plains 1.3 vs 5.0; the traced
values are V arithmetic at the 5.08% band, clvl 15, the table rows of recoveryxp.go:68-71; the earlier 0.5/0.7/0.8 used 1.95%): the
kills the bot makes include uniques, champions and packs. Catacombs 2 is not in the measured set, so its 97.2 is a plain-monster
figure too and its real per-kill value is higher by an unknown factor (U). What the table does support: the gap to the stay
(5.4 XP/kill) is about 18 times at the plain mean; a clvl 15 character is in the full-XP band for Catacombs 2 (monster level 11,
d=4) and in the 5.08% band for Cave 2, Stony Field and the Burial Grounds (monsters 10 or more levels below the player, recoveryxp.go:50).

> NOTE: sign. In this section "d" is mlvl minus clvl (recoveryxp.go:36). Catacombs 2 is mlvl 11 at clvl 15, so d = -4 (the 100% band, recoveryxp.go:45-46), not d=4 as written above. The 97.2 and the 5.4 XP/kill are unchanged (V).

## 6. Not in the log (do not guess)
- Any minute in Catacombs 2, 3 or 4, or in Outer Cloister, Jail 1 (areas 27-37) for seeds 11-13 at clvl >= 13 except 4 minutes
  (s11) and 5 minutes (s12) in Monastery Gate: none measured.
- Gold per area (carried-gold pickup per virtual hour) and XP per kill by monster: neither is measured. The minute lines carry totals per minute, not per kill or per area split (SME gap).
- Why Pick was empty at 86940 (section 2.2) and the tome state there.
- The split of XP between plain kills and uniques/packs in the minute lines.
- The virtual-clock length of each ban at 41596 (the `seconds` of the lines are the ban length; the log shows no clock reading in them).

## Shared-file lines for the merger
See openspec/changes/noareameasure/merge-lines.txt.
