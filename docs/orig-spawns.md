# Original game (Game.exe) monster spawns per area versus d2d (agent ORIGPROBES, 2026-10-30)

> status: current. Original-game side of docs/d2d-spawns.md (same columns). Tags: V-orig = measured on Game.exe 1.14d (instance i0, Testy, Normal, single player, Wine) on 2026-10-30; V-d2d = docs/d2d-spawns.md (20 seeds); U = not verified.

Probe: `cmd/origcasts spawns` (driver `cmd/origcasts/run-spawns.sh`, recomputation of the flag classes `cmd/origcasts/reclass.py`, table `cmd/origcasts/compare.py`). Raw: `docs/probes/orig-spawns.csv` (columns of `docs/probes/d2d-spawns.csv` plus `coverage_pct,known_cells,explore_s,end`), the flagged units per visit in `docs/probes/orig-spawns.log`.

## Method and what the reader can see

- One fresh game per visit (`oog leave`, `oog start Testy`: new map seed each time, V-orig: the area layouts differ between visits), the character walks from Rogue Encampment through Blood Moor, Den of Evil (and back), Cold Plains, Stony Field, Underground Passage 1 (the only way on foot to Dark Wood, docs/nav.md) to Dark Wood with the bot's `nav` (`ToArea`, then `Explore` until it reports nothing left), in the same game, like the d2d sweep did in one game. Logged god cheat for survival (reason "spawn count probe", throwaway Testy, level 1), no fighting.
- Census: `Snapshot().Monsters` (the client monster table `mem` reads) polled every 200 ms while exploring, de-duplicated by unit id, counted once when first seen alive in that area; town NPCs and non-killables dropped by Monsters.txt. Dead monsters never occurred (no fighting).
- **Not the same as the d2d sweep.** d2d kept every room loaded, so its count is the whole level. Game.exe only has monsters for rooms the client has activated near the character, so the count is "what exploring the walkable level revealed". The explorer walks to every frontier it can see, so the coverage is high but not proven: known walkable cells vs the level rectangle (`coverage_pct`): Blood Moor 98.8 %, Cold Plains 95 %, Stony Field 91 %, Dark Wood 95 % (V-orig, the rectangle includes unwalkable cells, so 100 % is not reachable). For the Den the figure (4.3 %) is meaningless (U: the cave's bounding rectangle is mostly void). A monster in a room the character never came close to is missed: the original numbers are a lower bound (U how much).
- Flag byte (MonsterData+0x16, V-orig, 60+ flagged units): `0x00` ordinary (so `d2.MonsterInfo.Kind()` returns Unknown, not Normal, for almost every monster: its "normal" bit 0x02 is NOT set), `0x10` pack follower of an elite (minion), `0x0c` champion (named "Champion/Possessed/Ghostly/... X"), `0x08` rare monster with its own name, `0x0a` superunique (Bishibosh; the others have flag 0x08/0x0a and are matched by name). One unit read `0x4c`. `d2.Kind()` tests the boss bit before the champion bit, so a first version of the census counted champions as rares; `reclass.py` redoes the classes from the logged flag byte (the CSV is the reclassified one). The `names` column lists ordinary units and followers only (champions, rares and superuniques are in `boss_names`, champions with their modifier name, unlike d2d which lists the base type).
- Visits: 4 per area (aim was 5; each visit takes about 10 to 13 minutes). Dark Wood: only 2 visits (visits 1 and 4: the walk to level 10 then to Dark Wood failed with "no exit to that level"; the first attempt also wandered into level 10 when `ToArea(5)` was sent from Stony Field, hence the `xN` explore-first leg). Failed rows stay in the CSV (end column `toarea:...`) and are excluded from the means.

## Result (V-orig 4 visits, V-d2d 20 seeds)

| Area | n orig / d2d | total mean orig | total mean d2d | orig min-max | d2d min-max | champion/visit orig | d2d | rare (unique col) per visit orig | d2d | superunique visits orig | d2d |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Blood Moor (2) | 4 / 20 | 90.0 | 69.0 | 57-113 | 48-85 | 0.00 | 0.00 | 0.00 | 0.00 | 0/4 | 0/20 |
| Cold Plains (3) | 4 / 20 | 173.2 | 127.0 | 111-258 | 58-215 | 0.00 | 0.95 | 0.00 | 0.70 | 4/4 | 11/20 |
| Den of Evil (8) | 4 / 20 | 56.0 | 54.4 | 49-62 | 26-76 | 0.00 | 0.00 | 0.00 | 0.00 | 0/4 | 20/20 |
| Stony Field (4) | 4 / 20 | 131.0 | 90.5 | 113-155 | 53-147 | 2.75 | 0.95 | 1.00 | 0.95 | 4/4 | 12/20 |
| Dark Wood (5) | 2 / 20 | 175.0 | 125.8 | 153-197 | 85-175 | 0.00 | 0.90 | 1.00 | 1.05 | 2/2 | 10/20 |

Means per visit of ordinary monsters (normal / minion), orig versus d2d: Blood Moor 90.0 / 0 vs 69.0 / 0; Cold Plains 170.2 / 2.0 vs 120.8 / 4.1; Den of Evil 56.0 / 0 vs 48.4 / 5.0; Stony Field 114.5 / 11.8 vs 79.1 / 8.9; Dark Wood 169.5 / 3.5 vs 117.3 / 6.0. Spread (population sd of the total): Blood Moor 24 vs 11, Cold Plains 60 vs 39, Den 5 vs 13, Stony Field 17 vs 26, Dark Wood 22 vs 27.

## Reading

- Original counts are higher than d2d in four of five areas (Blood Moor +30 %, Cold Plains +36 %, Stony Field +45 %, Dark Wood +39 % of the d2d mean) although they are a lower bound; only the Den matches (56 vs 54). With 4 visits the means are rough (Cold Plains ranges 111 to 258), but the direction is the same in every area and every visit stays at or above the d2d mean in three of them. Which monsters carry the excess: Fallen (Blood Moor 57 vs 38.5, Cold Plains 87 vs 65), Foul Crow (Stony Field 35 vs 18), Vile Hunter (Dark Wood 43 vs 18), Spike Fiend (29 vs 12). Hypothesis (U): d2d's pack size or density roll is lower than Game.exe's, or the original counts pack members spawned for the same room twice; not checked.
- Champions: seen only in Stony Field (3 of 4 visits, packs of 3 to 4, 2.75 per visit); the d2d probe has about 1 per seed in Cold Plains, Stony Field and Dark Wood (0.9 to 0.95). In Cold Plains and Dark Wood the original showed none in 4 and 2 visits (U: the sample is small; would need more visits to call it a difference). Rares (d2d's `unique` column): Stony Field 1.0 per visit (0 to 2 per visit: 2, 0, 1, 1 plus Rakanishu), Dark Wood 1.0, none elsewhere, like d2d (0.7 to 1.05 in Cold Plains, Stony Field, Dark Wood: Cold Plains 0 in 4 visits is again a small-sample gap).
- Superuniques: Bishibosh 4/4 (d2d 11/20), Rakanishu 4/4 (12/20), Treehead WoodFist 2/2 (10/20). **Corpsefire, the Den's superunique and its followers, were NOT seen in 4/4 visits (d2d: 20/20)** and the Den shows 0 followers (d2d 5): UNVERIFIED cause, most likely the explorer judged the cave done ("nothing left to explore" after 66 to 72 s) without activating Corpsefire's room, so this is probably a probe gap, not a game difference. If someone needs a V answer: walk the Den to its far end by hand-planned waypoints and read again.
- The original's spawn count varies with the seed as much as d2d's; no sign of a fixed count.

## Dead ends

- `nav.ToArea(5)` from Stony Field (does not know the exit): wandered into level 10 and then "no route from level 10 to 5". Fixed by the explore-until-exit leg `xN`, after `-10` (travel only).
- `nav.ToArea(3)` from inside the Den failed once ("stuck ... after 6 recoveries"): the driver now travels out to Blood Moor first (`-2`), which worked 4/4.
