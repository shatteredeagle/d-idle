# fid-chain-lightning-hop (decomp batch 7, matrix row SK12): Chain Lightning hop target selection and damage per hop, d2d against game.exe

> status: finished 2026-11-11 by agent decomp11 (Sonnet), offline (emulator on the worker host; no game, no d2d slot, nothing pushed or posted, no fork branch). Result: **the hop's PICK ORDER (the lowest unit id above the struck one, else the lowest up to it) is EQUAL in game.exe and d2d's real `rules::ring_pick`: 150,000 random cases, 0 differ (V run).** The candidate SET (which units reach the pick) and the per-unit `justhit` rule stay as the earlier trace left them (read, not run). No new d2d deviation, no PR candidate. Follows [sk-lightning-trace.md](../../research/sk-lightning-trace.md) (SKLIGHT, 2026-11-01, D1 to D5, B1 to B4) and the SK12 row of the [fidelity matrix](../../research/fidelity-matrix.md).

Tags: V run (game.exe's own code under the emulator against d2d's real code), V read, D (tables), S (an existing proof), I, U. Scripts: `tools/emu/ch_pick.py`, `ch_pick_test.cpp`, `ch_pick_run.sh` (this card).

## What the question was

"Chain Lightning hop open" in SK12: the hop pick was never compared; only the pierce of plain Lightning (decomp6, fid-missile-player.md) and the NextHit rule (model) were. Three parts: which unit is picked, how many hops and how far, damage per hop.

## Q1. The pick among the units found: EQUAL, V run

game.exe, FUN_0056bd10 (read 0x56bd10..0x56bd83): builds a result block on its stack (+0 best-above unit 0, +4 its id 0xffffffff, +8 lowest unit 0, +0xc its id := the struck unit's id, +0x10 the struck id, +0x14 count 0), calls the room walk FUN_0056b7e0 with the callback **0x56bcd0**, and returns +0 if set, else +8 (0x56bd6a..0x56bd71). The callback (read 0x56bcd0..0x56bd0c, unsigned compares): id above the struck id and below the best-above id: new best-above; else an id at or below the struck id and not above the running lowest: new lowest (ties replace). A null unit (end of the walk) changes nothing (id 0xffffffff).

`ch_pick.py game` runs **the callback itself** (address 0x56bcd0, the emulator's native call, context word +0xc pointing at the result block as FUN_0056bd10 builds it) over random sets, in random order, then reads the result as 0x56bd6a does. d2d side: `ch_pick_test` compiled against the scratch checkout of d2d (`rules::ring_pick`, components/rules/rules.hpp:2051, the function `Fight::skill_missile_hits` calls at missiles.cpp:652).

| seed | cases | picked "none" | differ |
|---|---|---|---|
| 11 | 50,000 | 5,500 | **0** |
| 29 | 100,000 | 11,079 | **0** |

Cases: 0 to 25 units, id pools of 40 / 300 / 70,000 / 2^31, the struck id among the found ones (55 %), one off it (25 %), or random; input sorted ascending, descending or shuffled; the null end-call in a third of the cases. Domain note: d2d's ring_pick holds ids in an `int` and treats a negative one as "none" (`above < 0`), so ids at or above 2^31 would pick wrongly; real unit ids are small sequential numbers, and d2d passes the index into `monsters` (D4 of the earlier trace: the index is the id only while the vector is in creation order, I; not run).

Not covered by this run (V read only): which units reach the callback. FUN_0056b3e0 with flags 0x8a783 (earlier trace Q2): hostile, targetable, alive (modes 0 and 12 refused), not in town, **not in state 0x56 justhit**, the line from the CASTER clear of mask 4, squared distance in subtiles within the radius of the hit spot. d2d's candidate set (missiles.cpp:631-656 of the scratch tree): rooms near the player, targetable, `align != 2`, squared subtile distance from the missile, `sight_blocked` from the player with mask 4, no justhit (D1 of the earlier trace). Equal except justhit.

## Q2. Hops per cast, radius, no to-hit (V read, S)

calc1 = (26 + level - 1) / 5 hops (5 at level 1, 9 at level 20; FUN_005ca1b0 stores it in the missile, FUN_005aa730 stops at 1 or fewer); radius 20 subtiles at every level (Chain Lightning's aurarange `par1`, `chainlightning` sHitPar1 0); the hop starts at the missile's subtile and aims at the next unit's; no to-hit roll (Missiles ToHit empty). V read and D (earlier trace Q1, Q2); d2d the same (missiles.cpp:637-657, `fire` `.hops = calc1`). **Stale d2d note:** the older d2d checkout on the worker host (the plain `src` tree) still has a "nearest monster" pick ("ponytail: the nearest one, not FUN_0056bd10's pick", fight.cpp:2953); the scratch checkout used by every batch since SKLIGHT has the ring pick (commit 527f6f30). I did not find out which of the two is upstream main today (U: `git log` is not available in the runner); the pick comparison above is on the ring-pick tree, and **if upstream main still has the nearest pick the deviation is real and the same harness proves it** (swap the d2d side to the nearest-pick lines; expected mismatches in most multi-monster sets, I).

## Q3. Damage per hop (S, V read)

Each hop is a new missile; min and max are built at its creation from the caster's current stats (FUN_0059fa30 -> FUN_0064b860: Lightning and Charged Bolt and Nova synergies, Lightning Mastery); rolled by FUN_005a8910 (min + rand(max - min), 0 unless both ends are > 0), resisted less pierce. No decay per hop. d2d: the roll and mastery are the merged rules (SKMISS D1 / #154 on current code, S). Immune targets: the hit function runs before the damage, a lightning-immune struck monster (0 damage) still hops (V read 0x5ae0e6..0x5ae18e; d2d equal).

## Deviations (d2d against game.exe)

| id | what | game.exe | d2d | tag | differential |
|---|---|---|---|---|---|
| D0 | pick order among found units | lowest id above the struck, else lowest up to it | `ring_pick`, same | **V run equal** (150,000 cases) | built here |
| D1 (carried) | NextHit `justhit` | unit state 0x56 for NextDelay frames, skipped by every NextHit missile and by the hop search | per-missile `hit_at`: a hop re-hits the struck monster in 73 % of hops (model) | V read both, model I | not built (stateful `Fight` test mode + FUN_005adf10 / FUN_0056bd10 with stubbed rooms; recipe in sk-lightning-trace D1) |
| D2 (carried) | Lightning (and any LastCollide missile) units per frame | one unit a frame, last unit skipped | every touching unit once | V read both | decomp6 built the pierce harness for Lightning (SK12 matrix row: 44 of 1,192 casts differ, candidate fid/missile-last-unit); Chain Lightning's own first strike is the same code |
| D3 (new, unproven) | d2d candidate id = index in `monsters` | unit id +0xc | index | I | one check that the vector stays in id order after deaths and spawns (could be run with a Fight test that kills and respawns) |

## Bot findings (described, not changed)

B1 (carried, SKLIGHT B1) `CastGapMs` 650 is shorter than the 760 ms Chain Lightning / Lightning cast: a command dropped in the cast. `internal/combat/classdata/skills.json` / the knob's own comment.
B2 (carried, SKLIGHT B2) `internal/combat/skills.go:179`, `target.go:102`: a lightning-immune target drops Chain Lightning from the pick although its hops would reach the pack behind; opportunity from Nightmare. Consumer: any skill-pick card (proposed `fid-sorc-pick`, batch7.md).
B3 (new) Nothing in the bot models the hop order: a hop goes to the next unit id, not the nearest. For "kill the pack with one Chain Lightning" estimates, the number of distinct monsters hit is min(hops, units in 20 subtiles that are in the caster's line), not nearest-neighbour chains (I).

## PR candidate

None. D0 is equal; D1 and D2 have no flight / hit harness in the existing set for Chain Lightning (D2 has one for plain Lightning, held as a candidate by decomp6).

## Drill recipe (lab binary, docs/forkdbg.md)

L1 (D1 hop re-hit): two Fallen 6 subtiles apart with life 1000 and lightning resistance 0 (`debug mon set ...`), Chain Lightning at level 1, read both lives each `step 1` and the hop missiles with `debug missiles`: game.exe expects A, B, then A again only after 4 frames; d2d today A twice in a row in most casts. L2 (pick order, D0/D3): three Fallen A, B, C spawned in that order within 20 subtiles of one another in a row away from the caster; cast at the middle one: expect the hop to go to the unit spawned after it (id order), not the nearest; kill B first and respawn it, then cast again: expect it to get the highest id, so the order changes (checks D3). Writing only; no game was run.

## Matrix-cell update (proposed text for the SK12 row of fidelity-matrix.md)

"UPDATE 2026-11-11 (decomp11, fid-chain-lightning-hop.md): Chain Lightning hop PICK compared: game.exe's callback 0x56bcd0 vs d2d's real `rules::ring_pick`, 150,000 cases, 0 differ (V run); candidate set equal except the justhit rule (D1, model only); radius, hop count and damage per hop equal (S / V read). Open: D1 justhit and the Lightning per-frame unit rule (flight harness)."
