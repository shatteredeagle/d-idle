# Den of Evil <-> Blood Moor shuttle (TASK-295)

> status: cause V (sandbox log), fix OFFLINE ONLY (unit tests, not run live). Follows docs/areathrash.md (same d2d arrival bounce, other pair). Tags: V verified in the log, S secondary or arithmetic, U unverified.

Source (V): read-only `docker logs --since 2h` of the sandbox container `d2play-20261108`, fresh Paladin, window 01:28:08 to 02:28:08 UTC 2026-11-11 (clock times in the log are UTC). Area ids from the plan table: 1 Rogue Encampment, 2 Blood Moor, 8 Den of Evil (`areaDenOfEvil`).

## What the hour is (V)

| | value |
|---|---|
| zone entries | 106: 49 Blood Moor, 45 Den, 12 town |
| entries inside shuttle bursts (gap under 10 s, pair 2/8) | 81, in 4 bursts of 3, 16, 20 and 42 entries (3 s, 18 s, 45 s, 97 s: 163 s in all), 0 kills in all of them |
| time by area (zone entry to next) | Blood Moor 46.2 min, Den 6.3 min, town 7.5 min |
| kills by area | Blood Moor 267 (5.8 per min there), Den 60 (9.5 per min there) |
| whole hour | 309 kills (5.1 per min), XP +6725, clvl 4 to 5 |
| time of the cascade below | 02:03:16 to 02:12:21 = 9.1 min, 15 percent of the hour |

So "a shuttle every 40 s" is the average of bursts: the hero is not in the Den all hour. The 2 to 3 kills per minute in `combat health` is a 60 s window that includes walking and waiting; over the hour it is 5.1.

## Why the hero leaves the Den (V, five cycles)

The Den is left by the planner, not by the danger gate, a chicken or the quest step: every leave is `goldfarm: farming gold on user request` (the user's order is on, `step = Farming gold on your order`) changing its target between the rung zones Blood Moor (2) and Den (8).

| # | time | what happened (log lines) |
|---|---|---|
| 1 | 01:58:16 | clvl 5 reached, `goldfarm` target 2 -> 8 (FarmRung is one rung below the plan pick, and the pick moved with the level). 01:58:40 arrival in the Den, `nav: stepping away from a cave mouth`, back in Blood Moor 1.1 s later (`exploring crossed an area border (target area 8)`), in again 1.4 s later: 3 entries, then 5 minutes in the Den with 21+ kills. |
| 2 | 02:03:15 | `guard: drank healing at life 45%`, belt healing 2 -> 1. 02:03:16.8 `goldfarm` target 8 -> 2 and `nav exit to 2` at 02:03:16.9, in the middle of a fight. Cause (code, V): `goldFarmTarget` excluded the Den when belt healing < 2 (`ex[areaDenOfEvil]`), a rule meant for entering, applied to a hero already inside. 02:03:39 in Blood Moor, 16 entries in 18 s (each side's `replanCause` fires `crossed an area border` for the other target), 02:03:58 breaker `area skipped Blood Moor 180 s border-loop`. |
| 3 | 02:04:00 to 02:05:03 | no rung left (Blood Moor banned, Den kit-excluded): `area gate refused the walk`, `Waiting: Blood Moor banned`, 24 state flips between clearing and in_town (one per 5 s, harmless but noisy). 02:05:03 `area wait: all areas banned, farming Blood Moor anyway`; the town to Blood Moor walk ends in the Den (`left level 1 for 2 but ended in 8`, the entry for area 2 is never logged: the bounce is faster than the poll), `area skipped unreachable` 150 s then 300 s, 20 entries in 45 s, no town portal scroll so the walk home bounces too. |
| 4 | 02:07:22 | restocked, belt healing back to 2: target 8 again, `nav: no path: start {142 351} is not on known walkable ground` (the arrival spot at the Blood Moor mouth), `area skipped Den 150 s unreachable`. |
| 5 | 02:09:13 | `farming Den of Evil anyway (remaining 65 s)`: 42 entries in 97 s, `walk to town failed ... left level 2 for 1 but ended in 8` three times; calm only when the Blood Moor ban lifted (02:10:19) and a walk from town stuck (02:12:11). |

The d2d arrival bounce (leaving a cave mouth puts the hero inside the trigger and the next move fires it; U why, known from docs/d2d-loop.md and docs/areathrash.md) makes every crossing between the two zones cost 2 to 40 entries. The border-loop breaker is NOT the cause and is not missing for this pair: it counts per target, so an alternating pair trips it after 5 crossings of one side (it tripped after 16 entries and 18 s). What multiplied the cost was the target flapping (cycle 2) followed by the forced "farm anyway" waits and the no-portal walks (cycles 3 to 5).

## Fix (this branch)

`internal/bot/goal.go` `denKitShort`: the potion buffer rule for the Den has a gap, entering needs `denPotionEntry` (2) belt healing potions, staying needs `denPotionStay` (1). Used at the three places that had the bare `healing < 2` (`goldFarmTarget`, the campaign leveling pick, `starveArea`). One drink inside the Den no longer turns the target around; an empty belt still leaves. Data stays the plan table (the Den is the existing `areaDenOfEvil` buffer rule; nothing class specific). Tests: `TestDenKitHasAGapBetweenEntryAndStay`, `TestGoldFarmStaysInTheDenAfterOneDrink` (Sorceress and Paladin; the second one fails with `denPotionStay = 2`, i.e. it reproduces the log).

Expected effect (S, arithmetic only): cycle 2 does not happen, which removes the ban and cycles 3 to 5 (9.1 of 60 min and 78 of the 81 shuttle entries) in this hour. It does not remove the first bounce of a normal entry (cycle 1, 3 entries).

## Not fixed, next steps

- The arrival bounce itself (U, d2d side; same open question as docs/areathrash.md). The proper fix removes all bursts.
- Breaker per pair (not done on purpose): per-target counting already trips; a pair counter would trip a little earlier (about 10 s) and would have to choose which side to ban.
- "all areas banned: farming X anyway" forces a walk that bounces again when the bounce is the reason for the ban; a forced walk after a border-loop ban could wait for the ban instead (U, needs a decision: it exists so the hero never idles).
- Other mid-zone evictions by exclusion rules (cleared set, guarded areas) were not audited here.
- Live acceptance: one hour on a fresh Paladin and a Sorceress with the farm order on, count entries per area pair and Den to Blood Moor flips with a potion drink as the trigger (none expected). NOT run.
