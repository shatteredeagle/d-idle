# Area thrash, cave-mouth holds and the Dark Wood exit (fun killer 5)

> status: analysis V (sandbox log), fix OFFLINE ONLY (unit tests, not run live). Source: `docs/research/beta-readiness-review.md` section 3.5. Task: TASK-242. Tags: V verified in the log, S secondary, U unverified.

## What the sandbox day shows (V, `d2play-20261108`, 2026-11-09 13:40 to 2026-11-10 01:24 UTC, read-only, nav.trace lines included)

Pairs of consecutive `zone entry` lines (1324 entries):

| Pair | Count | Typical gap | Reading |
|---|---|---|---|
| 3 -> 9 and 9 -> 3 (Cold Plains / Cave Level 1) | 324 + 323 | 1.9 s in, 1.1 s out | the cave shuttle: 49 percent of all entries |
| 4 -> 10 -> 4 (Stony Field / Underground Passage 1) | 51 + 48 | 6 s, 1.2 s | arrival bounce, 7 of 55 entries got through to Dark Wood |
| 17 -> 18 -> 17 and 17 -> 19 -> 17 (Burial Grounds / Crypt / Mausoleum) | 36 + 12 | under 1.5 s | same pattern |
| real travel (1-2, 2-1, 4-1, 1-4 ...) | the rest | 30 s to 2 min | town trips and field walks |

The 3/9 shuttle came in 19 bursts of 9 to 71 entries each (about 5 s per round trip). Kills in Cave Level 1 over the day: 0 (V, `combat kill` by area). `area skipped ... reason=border-loop` fired 18 times for area 9: the breaker worked, but with a flat 3 minute skip the planner came back every time.

## Causes

1. Planner keeps choosing an area the hero cannot stay in (V). At clvl 10 the level plan puts Cave Level 1 in its window. The hero arrives 5.8 subtiles from the return exit (arrival (143,270), exit (140,265)); the walk's step-away click is gated and rewritten (`nav mouth gate: click rewritten`), and 0.6 s later the hero is back in Cold Plains. 320 of 320 arrivals bounced (V). Why the game fires the exit although the click points away is U (known from `docs/d2d-loop.md`: d2d puts the hero inside the trigger area on return; the trigger shape is seed dependent). A run-versus-walk difference is ruled out (V: 7 of 8 Underground Passage successes ran too).
2. The flat breaker (10 crossings per minute, 3 minute skip) cost about 50 s per burst and returned 18 times.
3. `combat swing held: the click would walk into a cave mouth` (658): 587 of them (89 percent, V) fall within 60 s of a Cave Level 1 entry, at (255..261,140..143) in Cold Plains, the cave mouth at (255,140). The hold is the stay guard doing its job (`docs/fixmouth4.md`: a swing walks the hero into the mouth); it fires because the shuttle keeps the hero at the mouth. It is a symptom, not a second defect. No change to `internal/combat`.
4. `nav stuck` (1016): the largest cells are (250,140) in Cold Plains (94, the same mouth), (190,220) in Blood Moor (67) and (110,50) in area 19 (42); 32 of the 70 recoveries were side steps in Blood Moor. Not changed (U whether the Blood Moor spot is a map problem).
5. "Dark Wood exit lands in Stony Field" is not an exit-mapping error (V). The logged gates are right: from Underground Passage 1 the exit to Dark Wood is at (135,15) and the exit to Stony Field at (20,505) (`nav exit` lines, from the game's own map). The hero arrives at (25,515), 11 subtiles from the Stony Field exit, steps away to (30,522) or (8,528), and the exit fires anyway (48 of 55 entries); then `ErrLevelChanged` reads "left level 10 for 5 but ended in 4". `internal/senses/adjacency.json` is only the pair list for drawing neighbour maps and plays no part. Same arrival bounce as cause 1, 13 percent success.

## Why the hero sat in Blood Moor (coordinator question, V)

The planner never chose Blood Moor in the day: 0 `level plan` lines name it; its picks were Stony Field 33, Black Marsh 31, Dark Wood 30, Cave 2 8, Cave Level 1 7. The Blood Moor time (282 min at 932 XP/h, per the killloop finding) comes from `poverty mode: farming` (39 lines) and `gear gate: farming the first field` (2): `starveArea` and `safeFarmArea` send a poor hero to its first field, whatever the XP rate. The quest-chain order and the clvl-17 gate do not route there. Two interacting facts: the picks that need Underground Passage 1 (Dark Wood, Black Marsh) bounce 87 percent (cause 5), are skipped, and the hero falls back to Stony Field or, when poor, Blood Moor. I did not change the poverty first-field rule: it is a safety rule (`resource`, geargate) owned by the gold-loop work; proposal for that card: pick the poverty farm area as the safe area with the best XP per hour (`stepXP`) among those the gear verdict allows, and leave it once gold passes the threshold (it already does). Note: the sandbox binary of that window lacked the Paladin out-of-mana swing fix (dd208fa9), so back-off counts overstate.

## Fix (this branch)

`internal/bot/loop.go` border-loop breaker: 5 unwanted crossings of the same target within a minute (was 10, any target); the skip doubles for each repeat of the same area within 2 hours: 3, 6, 12, 24, 48 min (cap 4 doublings). Test: `TestBorderLoopSkipEscalates` in `internal/bot/areaskip_test.go`. Expected effect on the sandbox day (S, arithmetic only): about 40 round trips instead of 162 for Cave Level 1 and no 3-minute returns. The breaker is area-keyed and class independent; the test uses the default rig only, the logic reads no class data.

## Not fixed, next steps

- The arrival bounce itself (U). Needs a live look with `nav.trace` at 50 ms around the first click after a warp, one hero on the cave arrival tile: does the game fire the exit on the first move order whatever its direction? If yes, the right fix is a first order that is not a move (or a short wait), not a longer step-away.
- Planner side: do not put a level window area into the plan after N bounces (the breaker only skips the target; the unreachable skip for Dark Wood and Black Marsh is already escalating, `targetFailSkip`).
- Acceptance run: one hour realtime with entries per area per hour under a stated number is NOT done (no live game here).
