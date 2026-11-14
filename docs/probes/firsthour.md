# First hour: fresh hero, 60 realtime minutes at 1x

Headline metric of the project review (docs/research/project-review-2026-11-10.md, R2; TASK-245). One row per run,
appended by `tools/firsthour.sh` (cmd/d2firsthour). Tags: V read from a bot log line (docs/lifelog.md), S derived
by the rule below, U the log may not carry it.

| column | tag | derivation |
|---|---|---|
| date, start | V | log time of the fresh hero's first line (`d2bot start`) or the -start given |
| build | V | `d2bot start` version value(s) in the window; ` > ` joins a swap |
| clvl@60, gold | V | last `minute` line in the window; gold = carried + stash when logged |
| kills, deaths | S | `minute` counters summed across process restarts (a drop = a reset) |
| chickens | S | `safety decision action=chicken`; a chicken inside the fight loop is not logged (U how often) |
| entries, shuttles | V, S | `zone entry` lines; shuttle = A,B,A over three consecutive non-town entries (U the cap) |
| poverty stalls | S | minutes of each `poverty mode: on` to `off` span over 10 min (open at the hour's end counts) |
| restarts | S | the larger of the `minute` restarts counter growth and the extra `d2bot start` lines |

A row is only written when the log covers the whole hour of a fresh hero (clvl at most 2 on the first minute line).

| date | build | class | start | clvl@60 | kills | deaths | gold | chickens | entries | shuttles | poverty stalls (min) | restarts |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2026-11-12 | 0.0.261112021709 | paladin | 02:18Z | 6 | 528 | 0 | 1007 | 0 | 33 | 7 | 0 | 1 |
| 2026-11-12 | 0.0.261112171114 | paladin | 17:11Z | 5 | 445 | 0 | 281 | 0 | 78 | 49 | 0 | 1 |
| 2026-11-12 | 0.0.261112232458 | paladin | 23:25Z | 5 | 355 | 0 | 26 | 5 | 56 | 5 | 13+16 | 1 |
| 2026-11-14 | 0.0.261113174258 | paladin | 09:55Z | 5 | 494 | 0 | 726 | 0 | 17 | 3 | 0 | 1 |

## Targets (recalibrated 2026-11-13; derivation and tags: docs/first-hour.md "Targets")

The retired "clvl 13-15" figure is not a target (unreachable from Act 1 Normal kills, V xp-gold-trace). Tiers at 60 min, fresh Paladin: floor clvl 5 / 0 chickens / 0 deaths; target clvl 7, at least 14 kills per minute in the field, Cold Plains by minute 15; stretch clvl 8 to 9, Stony Field by minute 40. Gold is not a target.

| run | build | clvl@60 | tier on clvl | kills | chickens | verdict (V from the rows above) |
|---|---|---|---|---|---|---|
| 1 (02:18Z) | 0.0.261112021709 | 6 | floor+ (target is 7) | 528 | 0 | meets floor; 1 level under target; entries 33, shuttles 7 |
| 2 (17:11Z) | 0.0.261112171114 | 5 | floor | 445 | 0 | meets floor; 78 entries and 49 shuttles (routing waste) |
| 3 (23:25Z) | 0.0.261112232458 | 5 | floor, but 5 chickens | 355 | 5 | misses the floor on chickens (potion starvation, 13+16 poverty minutes) |
| 4 (2026-11-14 09:55Z) | 0.0.261113174258 | 5 | floor (292 XP short of clvl 6) | 494 | 0 | meets floor; misses target (8.2 kills/min, clvl 5); Den never entered (exit tile does not fire), 17 entries, 3 shuttles: docs/probes/hour4-analysis-2026-11-14.md |
