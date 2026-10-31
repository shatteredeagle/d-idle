# Per-kill time split, original vs d2d (agent GAPTIME, 2026-10-30)

> status: current. Offline analysis of existing logs plus the d2d real-cast probe (docs/d2d-casts-real.md). Tags: V = computed by me this run from the named files, S = read from a doc, U = unverified or inferred. Nothing was run on Game.exe.

## Inputs (V)

- Original bot (Game.exe, Sorceress, real time): minute lines of `~/d2lod-live/soak24/runs/*` (Soaky, i4) and `soak24-i5/runs/*` (Fresha, i5) on the worker host, 2026-10-29/30; only lines with an `rstate` field (672 older lines have none). A minute line gives `area`, `clvl`, cumulative `kills`, `rstate` (clearing, resupplying, recovering, in_town, traveling). Kill gaps from the `combat kill` lines (incl. "last line repeated").
- d2d bot (virtual time, 1 minute line = 1 virtual minute): `~/d2d-lab/loop/bot-{m11-14,g11,g12,g14,h11-14,r24,r25}.log` (ECONOMY and D2DLOOP runs, fresh Sorceress, clvl 1 to 6, mostly Blood Moor).
- Probes: docs/probes/orig-casts.csv (S, GAPCASTS), docs/probes/d2d-casts-real.csv (V), spawn counts docs/orig-spawns.md vs d2d (S, ORIGPROBES).
- Method for the budget: each minute is assigned to one component by `rstate` and by whether kills happened in it; per-kill seconds = component minutes x 60 / kills of the set. Kill deltas above 60 per minute or negative (restarts) dropped.

## What the soak states can and cannot tell (V)

- Can: time in state (clearing vs town chores vs recovery vs travel), kills per minute per area and clvl, share of clearing minutes with zero kills (the bot is searching or stalled), counts of log events per kill.
- Cannot: walk vs cast inside a clearing minute (one state), exact kill attribution (a kill is counted in the minute it is noticed; a town minute can carry the previous minute's kills), the original's per-area times at low clvl (only 20 Blood Moor minutes at clvl 1 to 6, 153 kills; most original Blood Moor and Cold Plains time is clvl 7 to 15), and anything about speed of movement.
- Both bots use the same Go code, so differences are the backend (monsters, timing, map) plus the character state (the soak characters were not god-moded; the probes were).

## Table 1: seconds per kill by area, same area class, whole minutes spent in the area (V)

| Area | original s/kill (kills) | d2d s/kill (kills) | ratio |
|---|---|---|---|
| Blood Moor, clvl 1 to 6 | 7.8 (153) | 20.9 (3709) | 2.7 |
| Blood Moor, all clvl | 5.2 and 8.2 per soak (172, 249) | 19 to 22 (m, g, h: 975, 781, 1197; r24 25.6) | 2.5 to 4 |
| Cold Plains, clvl 1 to 6 | 6.3 (239) | 29.5 (309) | 4.7 |
| Cold Plains, all clvl | 7.5 and 7.1 (1209, 594) | 23 to 41 (r24 168 kills 23.2; m, g, h under 35 kills each) | 3 to 5 |
| Stony Field | 4.7 and 5.8 (2345, 466) | no data (22 kills in h) | |

Blood Moor and Cold Plains are the areas where d2d has data; Stony Field is not reached in 2 virtual hours. Original rate is nearly independent of clvl (Blood Moor 7.8 s at clvl 1 to 6, 8.8 at 7 to 10, 5.2 at 11+), so the clvl mix does not explain the ratio.

## Table 2: per-kill budget by component, early character (V, minutes by state)

Original: soak minutes at clvl 1 to 6 in Blood Moor and Cold Plains (113 minutes, 742 kills, 9.1 s per kill, small sample). d2d: the 4+3+4 runs m, g, h, clvl 1 to 6 (1297 virtual minutes, 3292 kills, 23.6 s per kill).

| Component | how measured | original s/kill | d2d s/kill | gap (d2d - orig) |
|---|---|---|---|---|
| cast | real casts per kill x 0.56 s (d2d V: 1.16 at CB1; original: 0.96 commands, U) | 0.5 | 0.7 | +0.2 |
| fight and approach (clearing minutes with kills, minus cast) | `rstate` clearing, kills > 0 | 5.3 | 12.2 | +6.9 |
| search or stall (clearing minutes with zero kills) | `rstate` clearing, kills = 0 | 0.3 | 5.5 | +5.2 |
| town, chores, recovery, travel | `rstate` not clearing | 3.0 | 5.2 | +2.2 |
| total | | 9.1 | 23.6 | +14.5 |

Reading: **the biggest component gap is "fight and approach" (+6.9 s, 48 %), then search/stall (+5.2 s, 36 %), then town (+2.2 s, 15 %); cast is 1 %.** Walk (nav and travel) cannot be separated from the fight minutes by the soak states: "fight and approach" is walking between packs, dodging, potions, waiting for mana and the casts. The d2d minutes with zero kills are 31 % of the clearing minutes at Blood Moor (63 % at Cold Plains, 113 minutes) against 13 % (9 %) on the original (V).

Event rates per kill from the logs (V; d2d m11-14 and h11, r24 vs original i5 and soak24): `clear killed nothing` 0.30 / 0.23 (d2d, r24) vs 0.09 / 0.08; `nav stuck` 0.26 to 0.6 vs 0.22 / 0.19; `out of mana` 0.14 (d2d m) vs 0.24 (original), so mana waits are not why d2d is slower; `no town portal` (a walk to town) 0.07 to 0.22 per kill on d2d vs 0.01 to 0.03.

## Table 3: gaps between consecutive kills, Blood Moor, seconds per kill (V, kill timestamps)

| Gap class (time since previous kill) | original (2817 kills, all clvl) | d2d (3620 kills, virtual time interpolated between minute lines, U) |
|---|---|---|
| 4 s or less (same fight) | 1.0 | 0.6 |
| 4 to 30 s (next pack, walk) | 2.8 | 4.5 |
| 30 to 120 s (stall, mana, chore) | 1.0 | 7.0 |
| over 120 s (town trip, recovery) | 0.4 | 10.1 |
| total | 5.2 | 22.2 |

Same story as Table 2 from another angle: the time inside a fight and between neighbouring packs is close (3.8 vs 5.1 s); the gap is long waits between kills (15.7 s of the 17.0 s).

## Table 4: the Fighter alone (probes, god mana and life) (V)

| | original (7 min real, 1 run per level) | d2d Fighter (docs/d2d-casts-real.md) |
|---|---|---|
| seconds per kill row, CB 1 / 4 / 8 | 5.1 / 6.8 / 4.0 (S) | 6.2 / 5.5 / 5.5 at 450 ms (V) |

With cast, targeting, wander and movement the Fighter on d2d is as fast as on the original. The d2d bot loses the factor 3 above it: in bot code paths (search, stalls, trips), not in the combat engine. GAPCASTS' 10.9 s for d2d was based on an assumed 3600 s per run; real runs end at 260 to 650 s.

## Density (S spawn counts, V probe by seed)

Monsters per area visit, original vs d2d (ORIGPROBES): Blood Moor 90 vs 69 (1.30), Cold Plains 173 vs 127 (1.36), Stony Field 131 vs 90 (1.45); the original count is a lower bound (U). In the d2d probe the Fighter's time per kill falls with the seed's Blood Moor spawn total: 48 monsters 6.5 s, 54 6.2 s, 81 4.8 s, 82 4.0 s (V, 6 runs per seed, 4 seeds; elasticity about -0.9, 4 points, U) so about 1/density.

## Ranked explanation of the +14.5 s per kill (Blood Moor and Cold Plains, clvl 1 to 6)

| Rank | Cause | est. seconds | share | basis |
|---|---|---|---|---|
| 1 | idle between kills: search or stall minutes with zero kills, long waits (30 s+) | 5 to 8 | 35 to 55 % | Tables 2, 3 (V). Includes roam over a cleared area ("clear killed nothing" 3x per kill), nav stuck, d2d-specific stalls. Walking vs waiting inside it: U |
| 2 | town, no-portal walks, recovery | 2 to 4 | 15 to 25 % | Table 2 (V), docs/d2d-economy.md (poverty, no scroll) |
| 3 | density (1.3 to 1.45x fewer monsters) | 1.3 to 5 | 10 to 35 % | lower bound 1/1.3 on search minutes only, upper bound 1/1.3 on everything except cast and town; S spawn counts, probe elasticity V |
| 4 | walking speed and route | not measured | U | the Fighter-only probe (walks included) shows no gap, so any effect is inside bot-level routing (roam coverage, exits), not speed |
| 5 | casts (real) | 0.2 | 1 % | docs/d2d-casts-real.md |

The shares overlap (a longer search is partly because of lower density); they sum to about 100 % only as ranges. Density cannot explain more than about a third. Casting and dropped commands are ruled out.

## Recommended next experiment

1. (0.3 agent-h, d2d only) Run `cmd/d2dcasts2` without the life and mana top-up (`-notopup`, to add) in the Blood Moor at clvl 1 to 5 and compare seconds per kill with the god runs (5.5 s): this isolates mana, potions and chicken from the bot's search and trips.
2. (0.5 agent-h, offline then d2d) Add a per-kill budget log line to the bot (seconds in walk, cast, wait, town per 10 kills) from `nav` and `combat` timers, so both backends report the same split; today the soak minute line has one state for all of clearing.
3. (0.3 agent-h) Seed sweep: Blood Moor seeds with 80+ monsters (seeds 2, 4) vs under 55 (1, 3) with the full bot loop, to quantify density without touching the fork.
4. Only then spend fork time (user decides): monster count per room in `d2d` (1.3 to 1.45x) if the spawn deficit survives more original visits (4 per area now).
