# Route-guard floor measurement (Gate 2 seeds 11, 12, 13)

> status: measurement only. It feeds the SME floor design (per-zone data floors plus a raise-only death loop); it recommends no numbers.

Source: the three finished Gate 2 runs (seed 11 `g2a`, seeds 12 and 13 `g2v`), read-only on the worker host, 8 h wall each. One `minute` log line is one game minute (lockstep clock), so minutes of exposure are line counts. Tags: V = read from the logs by the scripts below, S = secondary (docs or recalled tables), U = unverified.

## Reproduce

```
G=<gate-run folder on the worker host>   # run through labrun; D2_LABRUN_HOST must be set
tools/floors/extract.sh OUT $G/gate2-s11-20261103-1954-g2a $G/gate2-s12-20261103-2210-g2v $G/gate2-s13-20261103-2211-g2v
python3 tools/floors/floors.py OUT > tables.md
```

`extract.sh` only greps `bot-0.log` (minute lines, `Entered ..` lines, `chose chicken first`, `Save and Exit`) and copies `supervisor.log`. `floors.py` is stdlib Python and prints every table below. Attribution: the delta between two minute lines is credited to the area and clvl of the earlier line; negative deltas (restart rollbacks) are not counted as gains.

## What the log has and does not have

- V per minute line: area, clvl, xp, kills, deaths, potionsUsed (cumulative), `potions` (healing potions in the belt), `packHP` (healing potions in the pack), `hpMinPct` (lowest life percent in that minute), restarts, xpLost.
- Max life (absolute): **not in the log**, in any run (grep for maxlife, max_life, hpMax: 0 hits). Only percentages exist (`hpMinPct`, `life=NN%` on chicken lines, `minlife=` on fight lines). The floor field minMaxLife has no raw input here; it needs a new log field.
- Healing potions at entry/exit: belt plus pack from the minute line of the first and last minute of a visit (a visit is a run of consecutive minute lines in one area, town trips split visits).
- Chickens: V from the decision line `potion: not drinking because chose chicken first life=.. belt=hp=..`, one per event (deduplicated within 20 s). The `Save and Exit (chicken / leave game)` line is also used by ordinary save points (4426 in seed 13), so it is not a chicken marker.
- Deaths: the minute-line `deaths` counter.

## 1. Exposure per area and clvl band

Areas 7 (Tamoe Highland), 26 (Monastery Gate) and 27 (Outer Cloister) first; the table is sorted by area id. Area 27 was never entered in any of the three runs. Names: S (ids 3, 6, 7, 17, 18, 19, 20, 26, 27 appear in docs/boss.md, docs/nav.md, docs/noareameasure.md; the others follow the level order). Rows with under 5 minutes are anecdote. Bands outside 13-16 are not tabulated.

| area | name | band | minutes | kills | XP/h | deaths | potions drunk | visits | healing (belt+pack) at entry mean/min | at exit mean/min | seeds with exposure |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Rogue Encampment | 13 | 46 | 388 | 16808 | 0 | 35 | 39 | 4.0/0 | 4.0/0 | 11,12,13 |
| 1 | Rogue Encampment | 14 | 100 | 373 | 7988 | 0 | 35 | 70 | 4.7/0 | 4.7/0 | 11,12,13 |
| 1 | Rogue Encampment | 15 | 267 | 805 | 1887 | 0 | 52 | 185 | 5.8/0 | 5.8/0 | 11,12,13 |
| 1 | Rogue Encampment | 16 | 2596 | 741 | 141 | 0 | 68 | 2161 | 0.2/0 | 0.2/0 | 11,12,13 |
| 2 | Blood Moor | 13 | 9 | 38 | 167 | 0 | 2 | 6 | 5.5/4 | 5.5/4 | 11,12 |
| 2 | Blood Moor | 14 | 10 | 15 | 48 | 0 | 0 | 10 | 6.0/4 | 6.0/4 | 11,12 |
| 2 | Blood Moor | 15 | 43 | 133 | 92 | 0 | 2 | 22 | 5.6/2 | 5.6/2 | 11,12,13 |
| 2 | Blood Moor | 16 | 11435 | 119 | 0 | 0 | 2 | 2087 | 0.0/0 | 0.0/0 | 11,12,13 |
| 3 | Cold Plains | 13 | 77 | 587 | 2811 | 0 | 33 | 23 | 5.9/4 | 5.8/2 | 11,12,13 |
| 3 | Cold Plains | 14 | 61 | 460 | 5255 | 0 | 32 | 42 | 6.0/2 | 5.9/2 | 11,12,13 |
| 3 | Cold Plains | 15 | 144 | 694 | 2705 | 0 | 26 | 131 | 6.3/0 | 6.3/0 | 11,12,13 |
| 3 | Cold Plains | 16 | 69 | 474 | 1701 | 0 | 28 | 47 | 5.0/0 | 5.0/0 | 11,12,13 |
| 4 | Stony Field | 13 | 95 | 510 | 7112 | 0 | 44 | 11 | 5.3/4 | 4.9/2 | 11,12 |
| 4 | Stony Field | 14 | 76 | 421 | 6077 | 0 | 37 | 12 | 5.2/3 | 5.5/3 | 11,12,13 |
| 4 | Stony Field | 15 | 829 | 4076 | 3135 | 1 | 336 | 92 | 6.3/1 | 6.2/3 | 11,12,13 |
| 4 | Stony Field | 16 | 267 | 1375 | 2417 | 0 | 101 | 37 | 4.6/0 | 4.9/0 | 11,12,13 |
| 5 | Dark Wood | 13 | 23 | 159 | 15702 | 0 | 21 | 3 | 7.0/4 | 6.0/4 | 12,13 |
| 5 | Dark Wood | 14 | 78 | 695 | 14455 | 0 | 151 | 17 | 4.1/3 | 4.1/2 | 12,13 |
| 5 | Dark Wood | 15 | 1 | 12 | 1680 | 0 | 1 | 0 | n/a | n/a | 12 |
| 5 | Dark Wood | 16 | 207 | 1593 | 3178 | 0 | 270 | 39 | 4.6/1 | 4.9/1 | 11,12,13 |
| 6 | Black Marsh | 13 | 34 | 318 | 36986 | 0 | 99 | 10 | 3.7/1 | 4.0/1 | 12,13 |
| 6 | Black Marsh | 14 | 70 | 581 | 17018 | 0 | 94 | 17 | 4.5/3 | 5.1/3 | 11,12,13 |
| 6 | Black Marsh | 15 | 27 | 178 | 13358 | 0 | 46 | 6 | 5.7/4 | 5.2/3 | 11,12 |
| 6 | Black Marsh | 16 | 184 | 1284 | 5027 | 0 | 182 | 38 | 5.1/0 | 4.8/0 | 11,12,13 |
| 7 | Tamoe Highland | 14 | 9 | 55 | 36547 | 0 | 23 | 4 | 5.8/4 | 5.0/4 | 11,12 |
| 7 | Tamoe Highland | 15 | 2 | 0 | 0 | 0 | 0 | 1 | 5.0/5 | 5.0/5 | 12 |
| 7 | Tamoe Highland | 16 | 38 | 275 | 17965 | 0 | 71 | 21 | 5.3/3 | 5.2/1 | 11,12,13 |
| 8 | Den of Evil | 13 | 11 | 65 | 1429 | 0 | 10 | 3 | 4.0/4 | 4.0/2 | 12 |
| 8 | Den of Evil | 14 | 21 | 111 | 426 | 0 | 3 | 4 | 5.8/5 | 6.2/6 | 11,13 |
| 8 | Den of Evil | 15 | 7 | 69 | 711 | 0 | 4 | 2 | 8.0/8 | 8.0/8 | 13 |
| 9 | Cave Level 1 | 13 | 86 | 729 | 6269 | 0 | 80 | 26 | 6.0/3 | 5.8/2 | 11,12 |
| 9 | Cave Level 1 | 14 | 107 | 1017 | 6476 | 0 | 89 | 51 | 6.3/1 | 6.3/3 | 11,12,13 |
| 9 | Cave Level 1 | 15 | 324 | 2871 | 6963 | 0 | 208 | 178 | 6.4/4 | 6.3/2 | 11,12,13 |
| 9 | Cave Level 1 | 16 | 63 | 495 | 2155 | 0 | 54 | 23 | 4.9/2 | 5.0/2 | 11,12,13 |
| 10 | Underground Passage 1 | 13 | 3 | 18 | 5520 | 0 | 0 | 3 | 5.0/4 | 5.0/4 | 12 |
| 10 | Underground Passage 1 | 14 | 10 | 75 | 9258 | 0 | 6 | 10 | 3.8/1 | 3.8/1 | 11,12 |
| 10 | Underground Passage 1 | 15 | 1 | 5 | 8940 | 0 | 1 | 1 | 3.0/3 | 3.0/3 | 12 |
| 10 | Underground Passage 1 | 16 | 21 | 174 | 6363 | 0 | 18 | 18 | 4.3/1 | 4.2/1 | 11,12,13 |
| 13 | Cave Level 2 | 13 | 54 | 418 | 9742 | 0 | 45 | 16 | 5.9/4 | 5.7/3 | 11,12 |
| 13 | Cave Level 2 | 14 | 106 | 1029 | 7382 | 0 | 112 | 43 | 6.4/4 | 6.2/2 | 11,12,13 |
| 13 | Cave Level 2 | 15 | 330 | 3427 | 3734 | 0 | 375 | 147 | 6.4/3 | 6.3/2 | 11,12,13 |
| 13 | Cave Level 2 | 16 | 15 | 151 | 2800 | 0 | 17 | 8 | 5.1/4 | 5.0/2 | 11,12,13 |
| 17 | Burial Grounds | 13 | 33 | 162 | 3945 | 0 | 18 | 14 | 6.1/4 | 6.0/4 | 11,12,13 |
| 17 | Burial Grounds | 14 | 39 | 176 | 2286 | 0 | 21 | 22 | 6.1/4 | 5.9/0 | 11,12,13 |
| 17 | Burial Grounds | 15 | 60 | 287 | 1290 | 0 | 17 | 43 | 6.6/4 | 6.6/4 | 11,12,13 |
| 17 | Burial Grounds | 16 | 19 | 71 | 1273 | 0 | 7 | 14 | 5.8/4 | 5.8/4 | 11,12,13 |
| 18 | Crypt | 13 | 44 | 239 | 12289 | 0 | 38 | 12 | 6.2/4 | 6.5/3 | 11,12,13 |
| 18 | Crypt | 14 | 58 | 266 | 6833 | 0 | 36 | 21 | 6.1/4 | 6.8/5 | 11,12,13 |
| 18 | Crypt | 15 | 384 | 1357 | 1218 | 0 | 163 | 82 | 6.6/2 | 6.7/1 | 11,12,13 |
| 18 | Crypt | 16 | 71 | 226 | 921 | 0 | 48 | 15 | 5.5/4 | 5.3/3 | 11,12,13 |
| 19 | Mausoleum | 14 | 4 | 24 | 3375 | 0 | 2 | 1 | 5.0/5 | 5.0/5 | 12 |
| 19 | Mausoleum | 15 | 26 | 96 | 612 | 0 | 12 | 4 | 6.0/5 | 5.8/4 | 12,13 |
| 19 | Mausoleum | 16 | 31 | 126 | 848 | 0 | 18 | 10 | 5.6/4 | 5.4/4 | 11,12,13 |
| 20 | Forgotten Tower | 14 | 1 | 0 | 0 | 0 | 0 | 1 | 6.0/6 | 6.0/6 | 13 |
| 20 | Forgotten Tower | 15 | 3 | 0 | 0 | 0 | 0 | 1 | 6.0/6 | 6.0/6 | 12 |
| 20 | Forgotten Tower | 16 | 1 | 0 | 0 | 0 | 1 | 1 | 1.0/1 | 1.0/1 | 11 |
| 26 | Monastery Gate | 14 | 1 | 8 | 77760 | 0 | 2 | 1 | 4.0/4 | 4.0/4 | 11 |
| 26 | Monastery Gate | 16 | 9 | 67 | 18427 | 0 | 10 | 9 | 6.2/4 | 6.2/4 | 11,12,13 |

Reading notes (V from the logs unless tagged). Area 1 clvl 16 (2596 min) and Blood Moor clvl 16 (11435 min, 0 XP/h) are the paused or waiting state after the run stalled at clvl 16 (mostly seed 13), not farming; ignore their XP/h. The XP/h of a one-minute row is a single minute. A minute line credits the delta to the earlier line's area, so a transit minute can carry the kills of the area just left (U, not separated). Area 26 shows as 9 visits of one minute each at clvl 16, 1 at clvl 14. Max life per row: not in the log.

## 2. Raw input at area entry

Visits to areas 7, 26 and 27 (every visit; 27 has none). Healing is the belt count and the pack count at the first and last minute line of the visit.

| seed | area | entry (EST) | clvl | minutes | belt healing in/out | pack healing in/out | kills | XP gain |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 11 | 7 | 11-03T21:57:43 | 14 | 1 | 6/6 | 1/1 | 0 | 0 |
| 11 | 26 | 11-03T21:58:04 | 14 | 1 | 2/2 | 2/2 | 0 | 0 |
| 11 | 7 | 11-04T00:14:57 | 16 | 2 | 4/4 | 0/0 | 15 | 1950 |
| 11 | 7 | 11-04T00:15:53 | 16 | 1 | 4/4 | 0/0 | 0 | 0 |
| 11 | 7 | 11-04T00:24:56 | 16 | 1 | 4/4 | 0/0 | 0 | 0 |
| 11 | 7 | 11-04T00:25:26 | 16 | 6 | 4/1 | 0/0 | 39 | 1996 |
| 11 | 7 | 11-04T00:51:29 | 16 | 1 | 5/5 | 0/0 | 0 | 0 |
| 11 | 26 | 11-04T00:52:23 | 16 | 1 | 5/5 | 0/0 | 0 | 0 |
| 11 | 26 | 11-04T00:59:05 | 16 | 1 | 4/4 | 0/0 | 0 | 0 |
| 11 | 7 | 11-04T01:27:33 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 11 | 26 | 11-04T02:27:04 | 16 | 1 | 6/6 | 1/1 | 0 | 0 |
| 11 | 7 | 11-04T02:52:48 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 11 | 7 | 11-04T03:44:04 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 11 | 7 | 11-04T03:53:05 | 16 | 3 | 3/4 | 0/0 | 9 | 1383 |
| 12 | 7 | 11-04T00:40:08 | 14 | 4 | 6/3 | 2/2 | 18 | 1978 |
| 12 | 7 | 11-04T00:41:07 | 14 | 1 | 4/4 | 0/0 | 0 | 0 |
| 12 | 7 | 11-04T00:41:25 | 14 | 3 | 4/4 | 0/0 | 18 | 1242 |
| 12 | 7 | 11-04T03:12:43 | 15 | 2 | 5/5 | 0/0 | 0 | 0 |
| 12 | 7 | 11-04T03:55:04 | 16 | 2 | 6/6 | 0/0 | 6 | 194 |
| 12 | 26 | 11-04T04:08:22 | 16 | 1 | 6/6 | 2/2 | 0 | 0 |
| 12 | 7 | 11-04T04:14:52 | 16 | 2 | 6/6 | 1/1 | 9 | 302 |
| 12 | 26 | 11-04T04:15:59 | 16 | 1 | 6/6 | 1/1 | 0 | 0 |
| 12 | 26 | 11-04T04:17:45 | 16 | 1 | 6/6 | 1/1 | 0 | 0 |
| 12 | 7 | 11-04T04:18:01 | 16 | 1 | 6/6 | 1/1 | 0 | 0 |
| 12 | 7 | 11-04T04:33:52 | 16 | 2 | 5/5 | 1/1 | 9 | 302 |
| 12 | 7 | 11-04T05:00:50 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 12 | 26 | 11-04T05:26:16 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 12 | 7 | 11-04T05:26:21 | 16 | 4 | 6/6 | 0/0 | 37 | 613 |
| 12 | 7 | 11-04T05:27:05 | 16 | 1 | 4/4 | 0/0 | 0 | 0 |
| 12 | 26 | 11-04T05:59:27 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 12 | 7 | 11-04T06:10:49 | 16 | 2 | 6/6 | 0/0 | 5 | 156 |
| 13 | 7 | 11-04T00:44:47 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 13 | 26 | 11-04T00:46:10 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 13 | 7 | 11-04T00:46:37 | 16 | 1 | 6/6 | 0/0 | 0 | 0 |
| 13 | 7 | 11-04T00:54:49 | 16 | 2 | 5/5 | 0/0 | 2 | 48 |
| 13 | 7 | 11-04T00:56:16 | 16 | 3 | 4/4 | 0/0 | 15 | 468 |

Summary of the table (V): clvl 14 visits to 7 and 26 had 4-6 belt healing potions at entry; the clvl 16 visits had 3-6 at entry (one visit exited with 1). No visit to 7/26/27 happened at clvl 13 or lower: the first entry in any seed was clvl 14.

## 3. Deaths and chickens

Deaths: one death in 3 seeds x 8 h (V). Seed 12, 04 Nov 02:48 EST, Stony Field (area 4), clvl 15, belt healing 7 on the line before, pack healing 0, `hpMinPct` 57 on the previous line and 0 on the death line. That is a non-route-guard zone with a full belt, so it does not speak for the 7/26/27 floors.

Chickens by area and band (87 events, V). The belt-healing column is the strongest signal in the data: 76 of 87 chicken events happened with zero healing potions in the belt.

| area | name | band | chickens | with belt healing 0 | median life% at the decision |
| --- | --- | --- | --- | --- | --- |
| 1 | Rogue Encampment | - | 2 | 1 | 24 |
| 1 | Rogue Encampment | 13 | 1 | 1 | 42 |
| 1 | Rogue Encampment | 14 | 1 | 1 | 19 |
| 1 | Rogue Encampment | 16 | 1 | 1 | 19 |
| 2 | Blood Moor | - | 3 | 3 | 52 |
| 3 | Cold Plains | - | 1 | 1 | 45 |
| 3 | Cold Plains | 14 | 1 | 1 | 24 |
| 3 | Cold Plains | 16 | 1 | 0 | 25 |
| 4 | Stony Field | - | 1 | 1 | 25 |
| 4 | Stony Field | 15 | 2 | 2 | 47 |
| 5 | Dark Wood | - | 5 | 3 | 25 |
| 5 | Dark Wood | 14 | 6 | 5 | 47 |
| 5 | Dark Wood | 16 | 4 | 4 | 53 |
| 6 | Black Marsh | - | 7 | 6 | 46 |
| 6 | Black Marsh | 13 | 8 | 8 | 52 |
| 6 | Black Marsh | 14 | 3 | 3 | 50 |
| 6 | Black Marsh | 15 | 2 | 2 | 48 |
| 6 | Black Marsh | 16 | 6 | 6 | 25 |
| 7 | Tamoe Highland | 14 | 1 | 1 | 54 |
| 7 | Tamoe Highland | 16 | 2 | 2 | 41 |
| 8 | Den of Evil | 13 | 1 | 1 | 25 |
| 9 | Cave Level 1 | - | 11 | 8 | 42 |
| 9 | Cave Level 1 | 13 | 1 | 1 | 22 |
| 9 | Cave Level 1 | 14 | 1 | 1 | 25 |
| 10 | Underground Passage 1 | - | 2 | 2 | 25 |
| 13 | Cave Level 2 | - | 3 | 2 | 25 |
| 13 | Cave Level 2 | 14 | 2 | 1 | 25 |
| 13 | Cave Level 2 | 15 | 4 | 4 | 25 |
| 17 | Burial Grounds | - | 2 | 2 | 54 |
| 17 | Burial Grounds | 16 | 1 | 1 | 25 |
| 18 | Crypt | 13 | 1 | 1 | 50 |

Every chicken event, as raw input (seed, time EST, area, clvl, life percent at the decision, belt healing, belt rejuvenation, pack healing):

| seed | time (EST) | area | clvl | life % | belt healing | belt rejuv | pack healing |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 11 | 11-03T19:55:08 | 2 | 2 | 23 | 0 | 0 | 0 |
| 11 | 11-03T19:58:39 | 9 | 4 | 28 | 0 | 0 | 0 |
| 11 | 11-03T19:58:50 | 9 | 4 | 52 | 0 | 0 | 0 |
| 11 | 11-03T19:59:16 | 2 | 4 | 52 | 0 | 0 | 0 |
| 11 | 11-03T20:02:50 | 9 | 6 | 25 | 3 | 0 | 0 |
| 11 | 11-03T20:06:20 | 9 | 7 | 25 | 3 | 0 | 0 |
| 11 | 11-03T20:16:20 | 10 | 9 | 22 | 0 | 0 | 0 |
| 11 | 11-03T20:49:47 | 9 | 12 | 37 | 0 | 0 | 0 |
| 11 | 11-03T20:55:04 | 4 | 12 | 25 | 0 | 0 | 0 |
| 11 | 11-03T20:59:00 | 9 | 12 | 45 | 0 | 0 | 0 |
| 11 | 11-03T21:10:43 | 13 | 12 | 52 | 0 | 0 | 0 |
| 11 | 11-03T21:38:01 | 13 | 14 | 24 | 6 | 0 | 2 |
| 11 | 11-03T21:55:17 | 9 | 14 | 25 | 0 | 0 | 0 |
| 11 | 11-03T21:58:49 | 6 | 15 | 48 | 0 | 0 | 0 |
| 11 | 11-03T21:59:01 | 6 | 15 | 42 | 0 | 0 | 0 |
| 11 | 11-04T00:07:28 | 4 | 15 | 25 | 0 | 0 | 0 |
| 11 | 11-04T00:08:32 | 4 | 15 | 47 | 0 | 0 | 0 |
| 11 | 11-04T01:33:21 | 6 | 16 | 52 | 0 | 0 | 0 |
| 11 | 11-04T01:47:59 | 5 | 16 | 53 | 0 | 0 | 1 |
| 11 | 11-04T02:30:35 | 6 | 16 | 12 | 0 | 0 | 1 |
| 11 | 11-04T02:31:45 | 6 | 16 | 25 | 0 | 0 | 0 |
| 11 | 11-04T03:01:22 | 5 | 16 | 53 | 0 | 0 | 0 |
| 11 | 11-04T03:53:06 | 7 | 16 | 41 | 0 | 0 | 0 |
| 12 | 11-03T22:15:15 | 17 | 4 | 20 | 0 | 0 | 0 |
| 12 | 11-03T22:21:04 | 9 | 6 | 52 | 0 | 0 | 0 |
| 12 | 11-03T22:23:46 | 9 | 7 | 23 | 2 | 0 | 0 |
| 12 | 11-03T22:32:28 | 1 | 9 | 23 | 3 | 0 | 0 |
| 12 | 11-03T22:32:54 | 5 | 9 | 32 | 0 | 0 | 1 |
| 12 | 11-03T22:33:42 | 5 | 9 | 25 | 0 | 0 | 0 |
| 12 | 11-03T22:42:44 | 6 | 9 | 20 | 4 | 0 | 0 |
| 12 | 11-03T22:43:09 | 10 | 9 | 25 | 0 | 0 | 0 |
| 12 | 11-03T22:44:06 | 5 | 10 | 22 | 4 | 0 | 0 |
| 12 | 11-03T22:50:17 | 6 | 10 | 46 | 0 | 0 | 1 |
| 12 | 11-03T22:50:58 | 6 | 10 | 39 | 0 | 0 | 0 |
| 12 | 11-03T23:07:26 | 17 | 12 | 54 | 0 | 0 | 0 |
| 12 | 11-03T23:58:32 | 9 | 13 | 22 | 0 | 0 | 0 |
| 12 | 11-04T00:03:00 | 6 | 13 | 54 | 0 | 0 | 0 |
| 12 | 11-04T00:03:51 | 6 | 13 | 19 | 0 | 0 | 0 |
| 12 | 11-04T00:05:55 | 6 | 13 | 54 | 0 | 0 | 0 |
| 12 | 11-04T00:14:01 | 8 | 13 | 25 | 0 | 0 | 0 |
| 12 | 11-04T00:15:22 | 18 | 13 | 50 | 0 | 0 | 0 |
| 12 | 11-04T00:25:35 | 13 | 14 | 25 | 0 | 0 | 1 |
| 12 | 11-04T00:29:30 | 6 | 14 | 53 | 0 | 0 | 0 |
| 12 | 11-04T00:30:04 | 6 | 14 | 50 | 0 | 0 | 0 |
| 12 | 11-04T00:40:36 | 7 | 14 | 54 | 0 | 0 | 2 |
| 12 | 11-04T00:43:26 | 5 | 14 | 47 | 0 | 0 | 0 |
| 12 | 11-04T00:49:15 | 5 | 14 | 52 | 0 | 0 | 0 |
| 12 | 11-04T00:49:48 | 5 | 14 | 45 | 0 | 0 | 0 |
| 12 | 11-04T00:56:23 | 5 | 14 | 25 | 1 | 0 | 0 |
| 12 | 11-04T01:02:06 | 5 | 14 | 22 | 0 | 0 | 0 |
| 12 | 11-04T01:17:47 | 13 | 15 | 24 | 0 | 0 | 0 |
| 12 | 11-04T01:22:24 | 13 | 15 | 24 | 0 | 0 | 0 |
| 12 | 11-04T01:58:12 | 13 | 15 | 25 | 0 | 0 | 0 |
| 12 | 11-04T02:05:57 | 13 | 15 | 25 | 0 | 0 | 0 |
| 12 | 11-04T04:24:46 | 6 | 16 | 25 | 0 | 0 | 0 |
| 12 | 11-04T04:40:59 | 5 | 16 | 54 | 0 | 0 | 0 |
| 12 | 11-04T05:26:54 | 7 | 16 | 33 | 0 | 0 | 0 |
| 12 | 11-04T05:44:23 | 5 | 16 | 45 | 0 | 0 | 0 |
| 12 | 11-04T05:48:53 | 6 | 16 | 23 | 0 | 0 | 0 |
| 12 | 11-04T06:00:39 | 3 | 16 | 25 | 1 | 0 | 0 |
| 12 | 11-04T06:00:44 | 1 | 16 | 19 | 0 | 0 | 0 |
| 13 | 11-03T22:11:57 | 2 | 1 | 52 | 0 | 0 | 0 |
| 13 | 11-03T22:16:00 | 9 | 4 | 52 | 0 | 0 | 0 |
| 13 | 11-03T22:16:15 | 1 | 4 | 24 | 0 | 0 | 0 |
| 13 | 11-03T22:19:21 | 3 | 5 | 45 | 0 | 0 | 0 |
| 13 | 11-03T22:20:47 | 9 | 6 | 42 | 0 | 0 | 0 |
| 13 | 11-03T22:20:59 | 9 | 7 | 53 | 0 | 0 | 0 |
| 13 | 11-03T22:21:44 | 13 | 7 | 25 | 2 | 0 | 1 |
| 13 | 11-03T22:22:00 | 13 | 7 | 23 | 0 | 0 | 1 |
| 13 | 11-03T22:29:23 | 5 | 9 | 23 | 2 | 0 | 0 |
| 13 | 11-03T22:30:08 | 5 | 9 | 45 | 0 | 0 | 0 |
| 13 | 11-03T22:40:21 | 6 | 11 | 36 | 0 | 0 | 1 |
| 13 | 11-03T22:40:55 | 6 | 11 | 50 | 0 | 0 | 0 |
| 13 | 11-03T22:41:07 | 6 | 11 | 50 | 0 | 0 | 0 |
| 13 | 11-03T22:42:27 | 6 | 12 | 54 | 0 | 0 | 0 |
| 13 | 11-03T22:58:49 | 6 | 13 | 45 | 0 | 0 | 0 |
| 13 | 11-03T22:59:01 | 6 | 13 | 52 | 0 | 0 | 0 |
| 13 | 11-03T22:59:11 | 6 | 13 | 52 | 0 | 0 | 0 |
| 13 | 11-03T23:02:59 | 6 | 13 | 48 | 0 | 0 | 0 |
| 13 | 11-03T23:03:13 | 1 | 13 | 42 | 0 | 0 | 0 |
| 13 | 11-03T23:03:24 | 6 | 13 | 52 | 0 | 0 | 0 |
| 13 | 11-03T23:13:46 | 3 | 14 | 24 | 0 | 0 | 1 |
| 13 | 11-03T23:17:07 | 6 | 14 | 43 | 0 | 0 | 0 |
| 13 | 11-03T23:17:31 | 5 | 14 | 50 | 0 | 0 | 0 |
| 13 | 11-03T23:17:35 | 1 | 14 | 19 | 0 | 0 | 0 |
| 13 | 11-04T00:57:54 | 6 | 16 | 34 | 0 | 0 | 0 |
| 13 | 11-04T01:02:20 | 17 | 16 | 25 | 0 | 0 | 0 |

Chickens in areas 7/26/27: 3 (seeds 11 and 12, one at clvl 14, two at clvl 16; all with an empty healing belt) (V).

## 4. The seed 13 "-1370 XP on a death in Cold Plains"

Verdict: **not a death; a process restart from the injected fault.** (V)

- Minute lines (seed 13): 23:12:04 EST, area 3 (Cold Plains) both before and after, clvl 14, xp 155363 to 153993 (-1370), `deaths` 0 to 0, `restarts` 18 to 19, `xpLost` 0 to 1375.
- The supervisor log: `23:11:56 fault injected: kill -9 ..`, `23:12:01 recovery ok in 2.5s`. The drop is 8 s after the kill. The fault is injected at minute 60 of every Gate 2 run.
- The same pattern in the other two seeds: seed 11 at 20:54:31 (xp -364, xpLost 476, restarts 11 to 12, kill at 20:54:25) and seed 12 at 23:11:21 (xp -2015, xpLost 2016, restarts 5 to 6, kill at 23:11:16). All three drops are fault kills; the bot's own `xpLost` counter (progress since the last save point) matches the drop (1375 vs 1370: 5 XP earned in the minute).
- Mechanism (V from the counters): the character is reloaded from the last save after the kill, so XP earned since the last Save and Exit is gone. No death occurred; the one real death (seed 12) shows no xp drop (207793 before and at the line).
- game.exe rule: Normal has death exp penalty 0 (S: docs/data.md from `difficultylevels.txt`, 0/5/10 for Normal/Nightmare/Hell). Our own logs agree (V): docs/plan/death-data.md section 5 found no xp drop across 17 earlier Normal deaths. The seed 13 number is therefore a harness artifact (the fault injection), not a d2d deviation. No evidence in these logs that d2d applies a death XP penalty in Normal.
- Side effect for the floors (V): the `xp` series is not monotonic around restarts, so XP/h from gains only (as here) is the safe estimate; a floor derived from raw first-to-last xp would be cut by the fault rollback (up to 2016 XP in these runs).

## 5. What the data supports and what it does not

Seeds: 3. Run length: 8 h wall each, but only seed 13 has a long game-time record (15483 game minutes); seeds 11 and 12 have 2313 and 2740 minute lines. All three reached clvl 16 and then mostly stalled (the Gate 2 failures are listed in each `supervisor.log`).

- Road front 13 (SME start number): there is **no exposure at clvl 13** in 7/26/27 (first entry was clvl 14) and no death or chicken there at clvl 13. The data neither supports nor contradicts 13. Exposure at 14-16: 7 has 49 minutes in total, 26 has 10, 27 has 0.
- Deep 15 (areas such as the Cave 2, Crypt, Burial Grounds range): exposure at clvl 15 is the largest in the table (Cave Level 2 330 min, Crypt 384 min, Cave Level 1 324 min, Stony Field 829 min) with 0 deaths there; the one death was Stony Field at clvl 15. Chickens at clvl 15 in Cave Level 2: 4, all with an empty belt. The data does not show a failure at 15; it also has no exposure below 15 in a comparable amount for a floor comparison.
- Tower 16: Forgotten Tower (20) has 5 minutes in total (clvl 14 to 16, 0 kills, 1 potion): no information.
- Andariel 18: no run reached clvl 17 or 18, so there is no data.
- Max life floors: no raw value exists; a log field must be added first.
- Healing potions: in the 7/26/27 visits the belt was 3-6 at entry; the chicken table says that a chicken at an empty belt is the common case (76 of 87 events, V). A minHealPots floor can be calibrated against that column; the sample is small.
- A raise-only death loop needs deaths as its signal: 1 death in about 3 x 8 h means the loop would almost never fire on this data; chickens are 87 events and are the more frequent signal (V).

Limits: 3 seeds, one character build, one fault per run, exposure of tens of minutes in the guarded zones, XP/h of short rows is noise, area attribution lags by up to one minute, Blood Moor and town rows at clvl 16 are stall time.
