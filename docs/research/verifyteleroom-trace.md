# VERIFY-TELEPORT-ROOM: independent re-check of the Teleport room rule (card verifyteleroom)

> status: card verifyteleroom, 2026-11-02, offline (no game, no d2d slot, no d2d run). Re-checks the claim of docs/research/sk-teleport-trace.md (SKTELE) that game.exe refuses a Teleport hop whose target lies outside the caster's room and that room's near list. Model: Sonnet. The SKTELE doc was read AFTER my own disassembly of FUN_005ca360 and FUN_00554ea0 (the order the item asks for), but BEFORE I wrote the harness, so the harness is independent in code, not in knowledge of the claim.

Tags: **V read** disassembled by me with upstream's `tools/emu/disasm.py` (through `tools/emu/vtr_dis.py`); **V run** game.exe's own functions run under the emulator by me (`tools/emu/vtr_room.py`, `tools/emu/vtr_cmp.sh`), on the worker host; **D** from a patched 1.14d table; **S** secondary (upstream notes, the SKTELE trace); **I** inference; **U** unverified. No game code is pasted; functions are named `FUN_xxxxxxxx`.

## Verdicts

| Claim (SKTELE) | Verdict | Where |
|---|---|---|
| C1 a target outside the caster's room and its near list is refused (do returns 0, nothing moves) | **HOLDS** | 70,225 grid cases, 80,000 irregular cases: 0 violations (section 2) |
| C2 so the range is a room rule, not a distance (the caster's place in its room does not matter; no skill-level term) | **HOLDS WITH LIMITS** | acceptance identical for 25 caster positions (V run); the real near list of a real level was not rebuilt (S, U); the span in subtiles is only as good as the room sizes (section 3) |
| C3 a per-level Teleport 0 refuses; 2 adds a line test (FUN_006229f0, mask 0x804, the clicked point) | **HOLDS WITH LIMITS** | V read + V run on the branch (section 4); the line walk itself and the meaning of Levels byte +4 were not re-derived (U) |
| C4 (part of the claim's wording) "the target must lie in the room or its near list" is the whole destination rule | **HOLDS WITH LIMITS** | it is the necessary test for the target; the landing can lie OUTSIDE the caster's near list (858 of 3,000 snapped landings), and the footprint cells need rooms too (section 3) |
| C5 class-generic | **HOLDS** | no class term anywhere on the path; a player's size is 2 for every class (section 5) |
| C6 arrival bookkeeping outline (warp spot, 20-step ring, timer at frame + 50, pets told) | **HOLDS** (outline, as run on hooked callees) | section 6 |
| the author's numbers (seed 7, 20,000 cases) | **REPRODUCED exactly** | section 7 |

## 1. What I read first (V read)

- `FUN_005ca360` (srvdofunc 27): target point from FUN_0056d2c0; the caster's room FUN_00620bb0 -> FUN_0061a1b0 -> FUN_0061db70 (Levels row); row null or room null: return 0. Byte +4 of the row: 0 returns 0; 2 and FUN_006229f0(caster, x, y, 0x804) non-zero returns 0; then the result of `FUN_00554ea0(game, caster, room 0, x, y, no-snap 0, flag 0)`. Reads at 0x5ca395-0x5ca3c9.
- `FUN_00554ea0`: the room-argument is 0, so it looks the target up. **Its first lookup passes a NULL room to FUN_00463740 and so always returns 0** (0x554efd-0x554f09; FUN_00463740 returns 0 for a null room at 0x463750). The effective lookup is the second: `FUN_00463740(caster's room, x, y)`; 0 returns 0 (0x554f0b-0x554f27). Then, with the no-snap flag 0, `FUN_0064e7b0(room, &xy, FUN_00620510(unit), mask 0x1c09, 0)` (which calls FUN_0064dea0 with 50 rings, step 1); 0 returns 0; then `FUN_00650be0` sets the position; 0 returns 0. Only then the player tail (type word 0) runs.
- `FUN_00463740(room, x, y)`: null room 0; the room itself when x is in [room+0x4c, +0x4c + room+0x54) and y in [room+0x50, +0x50 + room+0x58) (half open; room +0x4c/+0x50 origin, +0x54/+0x58 size, subtiles); else `FUN_00619790(room, &array, &count)` and the first list entry that holds the point; else 0 (0x46374c-0x4637e7).
- `FUN_00619790`: array pointer = room +0, count = room +0x24 (0x6197b9-0x6197c6). **The SKTELE harness hooked this function with a synthetic 8-neighbour list; mine reads the struct natively** (dead end: my first attempt had array and count swapped and crashed in FUN_00463740; the upstream note monster-ai.md already has the right order).
- `FUN_0064dea0` (snap): tests the target with FUN_0064d9b0 first; free: done. Else rings of Chebyshev radius 1.. up to the radius argument 50, every candidate looked up with FUN_00463740 starting from the room of the last candidate that had a room (the room argument is overwritten on each hit, 0x64e043 and 0x64e13c), kept the one of least |dx| + |dy| in its ring. So **candidate rooms are found by chaining through near lists, starting at the target's room, not the caster's** (V read, then V run in section 3).
- `FUN_00620510` (unit size): type word 0 (player) returns 2 through the jump table entry 0 (0x620521-0x620528); types 1 to 5 look something up by the unit's class or record (not read further).

## 2. The room rule on grids (V run, `tools/emu/vtr_room.py`)

World and unit: my own layouts. A **player-typed** unit (the player tail of FUN_00554ea0 runs; its callees are hooked, see the script docstring), size from game.exe's own FUN_00620510 (not hooked), near lists stored in the room structs as game.exe reads them, FUN_00619790 native. Hooked and not part of the claim: the target point, the caster's room, the Levels row, the collision grid fetch, the position set, the bookkeeping calls. Model (written by me from the disassembly): accept iff the target lies in the caster's room or a room of its list.

| Run | Command (worker host, `tools/emu/run-on-worker.sh [--ttl 1500] vtr_room.py ...`) | Cases | Accepted | Refused | Violations |
|---|---|---|---|---|---|
| lattice | `grid` (5 x 5 rooms of 40 subtiles; casters at x, y in {80, 90, 100, 110, 119}; targets x, y in -4..204 step 4) | 25 x 2,809 = 70,225 | 22,500 (= 25 x 30 x 30: exactly the 3 x 3 rooms round the caster, [40, 160)) | 47,725 | 0 (md5 5b52dea1) |
| irregular rooms, near lists = rooms under 30 subtiles apart on both axes (the upstream near-list rule, S) | `mixed 20000 11` | 20,000 | 9,529 | 10,471 | 0 (md5 5bf1e091) |
| same, other seed | `mixed 20000 23` | 20,000 | 9,530 | 10,470 | 0 (md5 2e06db7b) |
| irregular rooms, random near lists (random subset of rooms within 120 subtiles, the caster's own room dropped in 30 %) | the same two commands (second half of each) | 20,000 + 20,000 | 6,778 + 6,938 | 13,222 + 13,062 | 0 (md5 0dfda8d8, 8f3e9fdc) |

Rooms in `mixed`: rows of height 16..48 and widths 16..48 over about 200 x 200 subtiles, caster anywhere in a random room, half the targets within 2 subtiles of a zone room's edge, the rest anywhere from -10 to 214. Seeds are the `random.Random` seeds in the commands; the case stream is a function of the seed only.

Classes in `mixed` (the script prints them): **A** target outside the zone: game.exe refuses in every case (touch lists 10,471 + 10,470 cases, random lists 13,105 + 12,940 cases); **B** target inside the zone and the target's four neighbour cells (the plus footprint) all found by the room lookup: lands exactly on the target in every case (8,987 + 8,998 touch; 4,319 + 4,431 random); **C** target inside the zone but a footprint cell has no room: game.exe snaps (2,459 + 2,507 random, 542 + 532 touch) or, with the random lists only, refuses (117 + 122). Class C with touch lists is the map border: a target on the outermost row or column lands one subtile inward, still accepted. So **acceptance of the target by the room test is exact; refusal inside the zone can only come from the footprint or the snap** (I for why 117 + 122 refuse: the ring search finds no spot whose five cells have rooms through the chain; not traced per case).

## 3. What the claim leaves out (V run)

- **Caster position inside its room does not matter** (C2): the 25 caster positions of the lattice, corners of the room (80 and 119) included, accept exactly the same 900 targets each, as the room model says. SKTELE's "40 to about 80 subtiles depending on where the caster stands" is a statement about distance, true but not a rule of the game.
- **The landing is not limited by the caster's list** (C4): `chain 3000 5` (centre caster in the 5 x 5 grid; target on the border of the 3 x 3 zone; a blocked square of radius 2 to 14 round it): 3,000 of 3,000 landed, **858 outside the caster's zone**, 0 outside the TARGET room's zone, 0 on a blocked footprint cell (md5 fd28fee7). A hop can therefore end one room beyond what a distance reading of the rule would give; with the snap radius 50 and the lookup chain, a landing is tied to the target's room list, not the caster's. The bot never aims at a blocked cell (SKTELE B3), so this is a limit of the rule's description, not a bot finding.
- **The near list is data of the level.** My grids supply it; game.exe builds it (upstream notes, S: FUN_0066c370, the level's rooms under 6 tiles apart on both axes, itself included, plus close flagged rooms of next-door levels). I did not run the room builder, so "40 to 80 subtiles on 8 x 8-tile outdoor rooms" is an inference (I) from room size 40 subtiles x the touching-neighbour rule; on a level with small rooms (the maze levels) the list may reach two rooms away, and a hop is then longer. U for any real level.

## 4. Levels Teleport column and the line test (V read, V run)

- Levels.txt (patched stack, `mpq-table.sh levels`, D): the Teleport column is 1 on 135 rows, 2 on Act 2 - Duriel's Lair (73), 0 on the Null row only (id 0). This agrees with SKTELE.
- V run, branch behaviour of FUN_005ca360 with a faked Levels row (`misc`): Teleport 0 refuses (player and monster typed unit); Teleport 1 never calls FUN_006229f0; Teleport 2 calls it with (clicked x, clicked y, mask 0x804) BEFORE the snap and refuses when it answers non-zero, lands when it answers 0; no caster room: refuses. Target off the map (300, 300): refuses.
- U: that byte +4 of the real Levels record is this column (SKTELE takes it from d2d's `gamedata.hpp` comment; I did not read the loader, so it rests on the column order D + the d2d comment, S); and what FUN_006229f0 -> FUN_0064e260 treats as blocked (hooked to a fixed answer in both runs, the same limit as SKTELE).

## 5. Class-generic (V read, V run)

Nothing on the path reads a character class: FUN_005ca360 and FUN_00554ea0 take the unit pointer and use only its type word, its room and its path. FUN_00620510 returns 2 for type 0 whatever the class (V read, and V run natively in the player-typed oracle, where the footprint and ring sizes follow from it). So the rule applies to any class that has a Teleport-like srvdofunc 27 skill (the Sorceress skill 54; other classes reach FUN_005ca360 only if Skills.txt gives them `srvdofunc` 27, which I did not scan: U). The same acceptance results hold for a monster-typed unit (the SKTELE oracle's choice), run in `misc` and in about half the `mixed` layout batches (unit type drawn per batch from the seed).

## 6. Arrival (V run, outline of SKTELE row "player tail")

`misc` with a player-typed unit (callees hooked and recorded): the warp spot (+0x148 / +0x14c of player data from FUN_006221a0) equals the landing; the 20-step ring entry 0 holds the landing and the cursor byte advances once per landing (6 landings: 6); the timer call FUN_00541800 gets event 0xe, frame + 50 (game +0xa8 = 1000 gives 1050) and callback 0x554570; unit +0xc8 gets 0x10000; FUN_005754b0 is called with the landing x, y. A monster-typed unit takes the short tail: none of these. Not verified: what the pets do with it (the hook only records the call; U, as SKTELE).

## 7. Differences from the author's numbers

- SKTELE's `sk_tele_spot.py gen 20000 7` rerun with `tools/emu/run-on-worker.sh --ttl 1500 vtr_cmp.sh 20000 7`: 13,822 same, 4,902 game refuses / d2d lands (room not near), 938 both land elsewhere (monster), 308 game lands / d2d refuses (blocked), 25 + 5 others. **Identical to its table** (13,822 / 4,902 / 938 / 308 / 25 / 5).
- Differences of method, not of result: SKTELE hooked FUN_00619790 with 8-neighbour lists (a symmetric block); I read the lists natively and use irregular rooms and lists that are not symmetric or geometric. SKTELE typed the unit a monster and hooked FUN_00620510; I ran the player tail and the native size too.
- Differences of statement: (1) "range up to about 80 subtiles" is a distance reading; the rule is the zone (section 3). (2) The landing may leave the caster's zone (C4). (3) Refusal inside the zone is possible when the footprint cells have no room (class C), which SKTELE's grid (no unroomed cells inside the map) could not show.
- No difference in the verdict of D1 to D3 of SKTELE for the room test.

## 8. Deviations d2d vs game.exe (from this check)

No d2d run was made; nothing here confirms or retracts SKTELE's D1-D7 (they stay as SKTELE tagged them: V run against a line copy). One addition from my run:

| id | rule on both sides | tag | differential |
|---|---|---|---|
| D1 | a target whose footprint reaches off the map (outermost row or column): game.exe lands one subtile inward (the unroomed cell blocks the plus footprint, the snap finds the neighbour: 542 + 532 of 40,000 in touch-list runs); d2d `unit_blocked` treats off-level as blocked and refuses (S: SKTELE's reading of `spot()`, not read by me) | V run game side, S d2d side, so U as a finding | yes, could be built: add map-edge targets to `sk_tele_spot.py`'s case generator (its d2d line copy already refuses off-level plus cells), or the real `spot()` once SKTELE's C++ test mode exists; oracle = game landing from this harness's `mixed` batch |

## 9. Bot findings

None new. SKTELE B1 (the 18-subtile hop is the screen clamp) stays an inference; its "40+ subtiles per axis" side holds only for 8 x 8-tile rooms (section 3). `internal/nav/teleport.go` and `teleport_islands.go` are untouched and were not re-read for this card.

## 10. PR candidate

None. No d2d deviation was proven with the real build here (the d2d side is again a line copy in the author's harness), and this card changed no Go or C++.

## 11. Limits (not covered)

- Real near lists of real levels (the room builder, FUN_0066c370, was not run); any claim in subtiles on a real level is I.
- The footprint walk (FUN_0064d9b0 -> FUN_0064d100) and the line walk (FUN_006229f0 -> FUN_0064e260) were not read; the snap order with obstacles (ring, then |dx| + |dy|) is SKTELE's, not re-derived (my `chain` run checks only where landings fall and that they are free).
- Whether any class other than the Sorceress can reach srvdofunc 27 (Skills.txt scan not done).
- The client side of the click, the pets' placement, the mana payment (SKMANA), the real Levels loader.
- Town: refused by the skill start (FUN_0056f640) before this function; not re-read here.

## 12. Drill recipe (live, later; lab binary with the fork debug verbs, docs/forkdbg.md; I ran nothing)

- R1 zone edge: open ground, `debug goto` a field level, `dump` the caster's cell; `debug cast 54` at a target 2 rooms away (more than 80 subtiles along x) from the caster's position: expect refusal and mana kept in game.exe; then a target in the next room along x at 20 and at 55 subtiles from the caster: expect a hop in both cases whatever the caster's place in its room (C2).
- R2 landing past the zone: a target 5 subtiles inside the zone border with a tree or wall cluster round it: the landing may lie beyond the caster's zone (C4); read the landing cell with `dump`.
- R3 map edge: target on the outermost walkable column of a small level: game.exe lands one cell inward (D1); d2d refuses.
- R4 Duriel's Lair (73): a hop along a clear line lands, across a door does not (C3).
- The game.exe side is the expected value from this document; the d2d side reads the same by `dump`.

## 13. Matrix cell updates (docs/research/sorc-skill-matrix.md section 3.7, Teleport row, cell (a) "rules in game.exe")

Replace "the target must lie in the caster's room or its near list (40+ subtiles each axis on 8 x 8-tile rooms) [V run]" with: the target must lie in the caster's room or a room of its near list [V run, 70,225 + 80,000 cases, verifyteleroom-trace.md]; the caster's place in its room is irrelevant [V run]; the landing's rooms chain from the TARGET's room and may leave the caster's zone [V run]; a hop whose footprint reaches off the map lands one cell inward [V run]; the span in subtiles depends on the level's room sizes and near-list rule [I]. Cell (b): add D1 (map edge) of this trace [V run game side, S d2d side].

## 14. Reproduction

Worker host, from the card's worktree: `tools/emu/run-on-worker.sh --ttl 1500 vtr_room.py grid`; `... --ttl 2400 vtr_room.py mixed 20000 11`; `... mixed 20000 23`; `... vtr_room.py chain 3000 5`; `... vtr_room.py misc`; `... --ttl 1500 vtr_cmp.sh 20000 7`; disassembly: `... vtr_dis.py 5ca360 554ea0 463740 619790 64dea0 620510` (upstream emulator checkout, patched 1.14d MPQ stack of the worker host). All outputs are deterministic; the md5 prefixes above hash the case results, not game files.

## Shared-file lines for the merger

See `openspec/changes/verifyteleroom/merge-lines.txt`. The ledger row for Teleport (skills.sor 54) stays as SKTELE wrote it (one line may claim a row; this card adds no new d2d comparison), so `docs/coverage/` is unchanged.
