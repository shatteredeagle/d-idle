# Meteor's blast: which units a Meteor hits, game.exe 1.14d vs d2d vs the bot (decomp row 22, SKGROUND, third pass)

> status: card skground, 2026-11-02, offline (no game, no d2d slot, nothing pushed or posted). The earlier passes ([sk-ground-trace.md](sk-ground-trace.md), [sk-ground2-trace.md](sk-ground2-trace.md)) left **Meteor's blast shape** (D6, FUN_0056b7e0) open; this pass reads and RUNS it. One topic: the blast (who is found, where, with which radius). Hydra's AI, Frozen Orb and Thunder Storm stay separate items (section 9). Model note: this run was Sonnet; the only PR-relevant number (D1) has both sides running real code, and no candidate is proposed.

Tags: **V read** disassembled with upstream's `tools/emu/disasm.py` via `tools/emu/skg_dis.py` (addresses given); **V run** game.exe's own code run natively under the emulator (`tools/emu/skg_blast.py`, `skg_blast_diff.py`) and, for D1, compared with the REAL d2d `Fight::area` (a hand-built test, `tools/emu/skg_test.cpp`, against the scratch build of upstream main); **D** the patched 1.14d tables; **S** d2d source or an earlier trace; **I** inference; **U** unverified.

## Answer in ten lines

1. **What makes the blast (V read):** Meteor's hit function 14 = FUN_005aabb0 (missile do 1 ends, `meteorcenter` Range 60) rolls ONE record (FUN_005a89a0) and calls FUN_0056bad0(game, owner, x, y, radius, record, 0) with x, y = the missile's own integer subtile; FUN_0056bad0 supplies the default filter flags 0x8583 and the callback FUN_0056ba90 (a copy of the record to every unit found) and calls the unit search FUN_0056b7e0.
2. **The shape is a disc on integer subtiles (V run):** a unit is found when `dx^2 + dy^2 <= r^2` (FUN_006492a0 = the squared distance, `ret 0x10`), dx, dy from the **integer words of the two paths** (FUN_006488c0 / FUN_00648900 read the words at +2 / +6; the fraction words +0 / +4 are never read). 18 (site, radius) cases of a 19 x 19 block of monsters, radius 1..9 (5, 13, 29, 49, 81, 113, 149, 197, 253 units): **0 differ from the disc**; the same blocks with random fraction words: 0 changed. A unit's size plays no part (centre test only, V read).
3. **The radius is 6 subtiles at every level (V run):** FUN_005aabb0 on a real `meteorcenter` (FUN_0059fa30, skill 56, level 1..20): the callback sees r^2 = 36 at 20 of 20 levels (sHitPar1 blank, aurarangecalc `ln12` with Param1 6, D).
4. **One roll, shared (V run):** every callback of a blast gets the same record pointer (`one_record 1` in all 18 cases); FUN_0056ba90 copies it (0x70 bytes) and hands it to FUN_0056b9c0 per unit (V read). d2d D5 of sk-ground-trace.md (a roll per monster) stands.
5. **The search starts from the CASTER's room (V run):** FUN_0056b7e0 takes the rooms from `FUN_00620bb0(owner)` (the OWNER's room1) and its near list (FUN_00619790: list at room +0, count at +0x24; the list holds the room itself, 83 of 83 rooms), skips a room whose rectangle misses the blast's bounding box (FUN_0056b770) and walks each room's unit list (+0x74, next +0xe8). Blast in the caster's room or on its near list: found 28 of 28; blast in any other room: **0 of 168** (34 of them within the 100-subtile cast cap, none found; smallest such distance 62 subtiles; Blood Moor sample, 15 free sites).
6. **The filter, flags 0x8583 (V run + V read):** a monster is found only when it is **hostile** (FUN_00554200 against the owner), its mode is not 0 or 0xc (death, dead; modes 1..11 and 13..17 all found) and bits 0x4 and 0x8 of its unit flags (+0xc4) are set; a created monster has 0x8200e; clearing any other of bits 0..15 changes nothing. The owner itself is never found (V read, `cmp unit, owner`). Other bits of 0x8583 (0x100: the unit's room FUN_0061ab00 must be 0; players, objects) read only.
7. **The return value (correction, V read + V run):** FUN_0056bad0 returns 1 whatever was found (the `mov eax, 1` after the call; 20 levels x {unit in range, none}: eax 1 in all 40), so FUN_005aabb0 never takes its `3` path: [sk-ground-trace.md](sk-ground-trace.md) Q9 ("else 3") is wrong.
8. **d2d (S):** `Fight::area` takes every `targetable()` monster with `hypot(float dx, float dy) * 5 <= radius` on float cell positions. Against game.exe's floor-and-disc rule on the same random float pairs: **7,276 of 80,000 cases differ (9.1 %)**, 3,724 of 40,000 (9.3 %) in a second run; both sides hit about the same number of units (34,576 vs 34,622): no bias, a per-unit jitter at the rim (D1).
9. **No PR candidate** (section 7): D1 is proven by a differential on the real `Fight::area`, but the claim hangs on one inference (a unit's integer word is the floor of its true position, I), its effect is a one-subtile jitter at the rim with no change in the mean, and `area` is shared by other hit functions whose game callers were not traced.
10. **Bot:** nothing in the bot depends on the blast shape; aiming at the monster's spot (B1) and the 30-subtile range (B2) are consistent with the rule.

## 1. Questions answered (Q26..; Q1..Q25 are in the two earlier traces)

| id | question | answer | where | tag |
|---|---|---|---|---|
| Q26 | What does the hit function do with the blast? | FUN_005aabb0(game ecx, missile edx, arg): row valid, owner FUN_00552fd0, the missile's skill and level (FUN_0064a280 / FUN_0064a210), skill record (stride 0x23c); centre = FUN_0045adf0 / FUN_0045ae20 of the missile (its path words +2 / +6 = integer subtiles); radius = row +0x4c (sHitPar1) if > 0 else FUN_00646ca0 of the skill's +0x64 (aurarangecalc) at the level, at least 1; one 0x70-byte record by FUN_005a89a0, OR'd with row +0xa8 / +0xac; FUN_0056bad0(game, owner, x, y, radius, &record, 0); the result is 1 whatever it returns (Q31) | 0x5aabb0..0x5aad3b | V read |
| Q27 | What is FUN_0056bad0? | ecx game, edx owner, stack x, y, radius, record, flags: flags 0 becomes 0x8583; owner 0 -> nothing; else FUN_0056b7e0(game, owner, x, y, radius, flags 0x8583, callback FUN_0056ba90, record, 0 (the extra flag), file and line strings); then `mov eax, 1` | 0x56bad0..0x56bb12 | V read |
| Q28 | How does FUN_0056b7e0 search? | (ecx game, edx owner): game, owner, callback and radius > 0 required; the room = FUN_00620bb0(owner) (a unit's path room: type 2 / 4 / 5 read +0x2c -> +0, other types FUN_00648a80); a zero x or y is replaced by the OWNER's own position (FUN_0045adf0 / 0045ae20; the blast passes the missile's, never zero in practice: I); r^2 = radius^2 stored; the room list = FUN_00619790 (list +0, count +0x24); per room: flag 0x2000 set -> skip rooms where FUN_0061ab00 != 0 (not set for 0x8583); FUN_0056b770(room, x, y, radius): the room rectangle (FUN_00619730 copies room +0x4c, 8 dwords: x, y, w, h) against the box [x - r, x + r] x [y - r, y + r], overlap or skip; per unit of the room (+0x74, next +0xe8): skip the owner; position = path words +2 / +6 (types 2, 4, 5: +0x10 / +0xc of the object's path, a different layout); `FUN_006492a0(x, y, ux, uy)` <= r^2 (signed compare `jl` skips when r^2 is less) and FUN_0056b3e0(owner, unit, flags) non-zero; then (the extra flag 0 is set by no caller here) the callback with ecx = a 0x14-byte block {game, owner, count, record, r^2} and edx = the unit | 0x56b7e0..0x56b9b9, 0x56b770, 0x6492a0 | V read; positions and disc **V run** |
| Q29 | What is FUN_006492a0? | `(x2 - x1)^2 + (y2 - y1)^2` in 32-bit integers, `ret 0x10`, nothing else | 0x6492a0..0x6492ba | V read |
| Q30 | What does FUN_0056b3e0 test with 0x8583? | jump table on the unit type (0x56b6ac): type 0 player, 1 monster, 2 object, 3 missile, 4 item (entries not all read; monster entry 0x56b43c). Bits of 0x8583 = 0x8000 hostile (FUN_00554200 owner / unit), 0x400 (+0xc4 bit 3), 0x100 (the unit's room FUN_0061ab00 == 0), 0x80 (+0xc4 bit 2), 0x2 (monster: mode not 0, not 0xc), 0x1 (players, own branch). The monster branch: bit 0x2 set, mode (+0x10) must differ from 0xc and from 0 (flag 0x1000 unset), then the shared tail 0x56b4e9 tests the +0xc4 bits | 0x56b3e0..0x56b6a8 | V read; the mode and +0xc4 effects **V run**; the names of the +0xc4 bits U |
| Q31 | What does the blast return? | FUN_0056bad0 always 1; FUN_005aabb0 therefore always returns 1 (its `3` branch is dead code on this path) | 0x56bb0c, 0x5aacf5..0x5aad00 | V read + **V run** (40 of 40 calls) |
| Q32 | Which radius does Meteor use? | 6 subtiles at levels 1..20 (r^2 = 36 seen by the callback, every level) | `skg_blast.py` stage radius | **V run** + D |

Dead ends: the unit types of the FUN_0056b3e0 jump table other than the monster were not read (players, objects: the Meteor test is monsters); the **names** of the +0xc4 bits (0x4 / 0x8 are required, 0x2 is not tested by the blast) were not found; FUN_0056b9c0 (the per-unit damage) was hooked, not run; the owner in the end-to-end run is a monster (class 63), so the record's values were not compared; the cast's own distance cap and the 0x64 limit are SKGROUND Q2; a size above 1 was not run (the search reads only the centre, V read); Blood Moor is the only level tried for the rooms test.

## 2. The rule as a formula

```
blast(owner, point P = the missile's integer subtile, r = 6):
   rooms  = near list of the OWNER's room (itself included); keep a room whose rectangle meets the box P +- r
   unit u in those rooms' unit lists: found iff u != owner, hostile(owner, u), mode(u) not in {0, 12},
          flags(u) has 0x4 and 0x8, and (floor(ux) - Px)^2 + (floor(uy) - Py)^2 <= r^2
   every found unit gets a copy of the SAME record (one damage roll per blast)
   returns 1 always
```

Worked numbers (V run): a radius-6 blast covers 113 subtiles (the disc of integer points), a radius-1 blast 5; on a block of monsters a radius-6 blast hands the callback 113 of 361 units.

## 3. d2d (S, upstream main 7d5b3a5c; read for this pass)

- `Fight::burst` (components/game/fight.cpp:3039-3049) case 14: `area(missile, missile.x, missile.y, radius(hit_par1, aurarange))` then the fire spread. `Fight::area` (fight.cpp:3004-3007): every monster with `targetable()` (ai.hpp:349: alive, killable, not buried, not aloft, not riding) and `hypot(unit.x - x, unit.y - y) * 5 <= max(radius, 1)` on FLOAT cell positions; `strike` rolls a record per monster (SKGROUND D5).
- No room or near-list rule, no integer positions, no hostility test beyond the monster list.

## 4. Deviations (this pass's are numbered M1..; D1..D13 are in the two earlier traces)

| id | rule | game.exe | d2d | tag | reach | differential |
|---|---|---|---|---|---|---|
| M1 | position model | integer subtile words of the unit and of the blast centre; the disc `dx^2 + dy^2 <= r^2` on them | float cell positions, `hypot * 5 <= r` | **V run both sides** (game side 18 disc cases 0 differ; d2d side the real `Fight::area`; comparison 120,000 cases, section 5); the floor mapping of a true position to the word is **I** | a unit within about one subtile of the rim is found by one side only: 9.1 % of random (centre, unit) pairs differ (12 % of those placed within 1.5 subtiles of the rim, 6 % uniform in the +-(r + 2) window); hits per cast equal on average (34,576 vs 34,622 of 80,000), so no change in the mean damage | built: `skg_blast_diff.py gen / game / cmp`, `skg_test --blast`, `skg_diff.sh` |
| M2 | which rooms are searched | the owner's room and its near list only (62 subtiles and more away: 0 of 34 within the cast cap found a unit) | every monster of the level | V run (game side, Blood Moor sample) + S | casts at a point 62 to 100 subtiles from the caster with the target in a room off the caster's near list: game.exe hits nothing, d2d everything; a screen is about 40 subtiles wide, so rare (I) | game side built (`skg_blast.py` stage rooms); a real-game confirmation needs a long cast: G3 below |
| M3 | unit filter | hostile, mode not 0 / 0xc, +0xc4 bits 0x4 and 0x8 | `targetable()`: alive, killable, not buried, not aloft, not riding | V run (game) + S; mapping of the d2d predicates to the bits **U** | none known for plain monsters; a buried maggot, an aloft vulture, a riding imp, an unkillable monster may or may not be in the blast (U) | not built: put a unit of each of the four kinds under the game's own creation code and read +0xc4 / mode, then the blast |
| M4 | return value | always 1 | no value | V read + V run | doc only: [sk-ground-trace.md](sk-ground-trace.md) Q9 said "else 3" | none |

Not deviations at this depth: the radius (6 at every level on the game side by run; d2d's `calc(aurarange)` is S, not run, so equality is read depth), the one shared record (D5 of sk-ground-trace.md, already listed).

## 5. The differentials (V run)

Commands (the worker host; the wrapper copies nothing, `tools/emu` is mirrored; every output lands in the agent's scratch directory):

```sh
tools/emu/run-on-worker.sh skg_dis.py 56b7e0 , 56ba90 , 56bad0 , 5aabb0       # the reads (Q26..Q28)
tools/emu/run-on-worker.sh --ttl 1200 skg_blast.py 1                           # the game-side stages: shape, rooms, filter, radius
tools/emu/run-on-worker.sh --ttl 2400 skg_diff.sh 1 40000                      # gen + game + real d2d + compare
tools/emu/run-on-worker.sh --ttl 2400 skg_diff.sh 7 80000
```

`skg_blast.py` (stages, one process, hooks only the hostility test FUN_00554200 and the callback FUN_0056ba90; FUN_0056bad0, FUN_0056b7e0, FUN_006492a0, FUN_0056b3e0, FUN_00619790, FUN_0059fa30, FUN_005aabb0 and FUN_005a89a0 run as game code): `shape` (a 19 x 19 block of size-1 monsters, class 190, radius 1..9, twice, then with random fraction words), `rooms` (every pair of rooms with a free 25 x 25 spot: owner in A, one monster at the blast centre in B, radius 2), `filter` (modes 0..17, each of 16 flag bits of +0xc4, a non-hostile unit), `radius` (a `meteorcenter` per level 1..20, with and without a unit on the centre). Random inputs come from `random.Random(SEED)` (default 1).

`skg_blast_diff.py gen SEED N OUT`: case `id kind r cx cy ux uy` in milli-subtiles, r uniform 1..9, the centre's fraction (cx, cy) uniform in 0..999, kind U = unit uniform in the +-(r + 2) window, kind B = unit within 1.5 subtiles of the circle (`xorshift32`, seed multiplied by 2654435761). Game side: the unit's path words are set to the subtile floor(u / 1000) relative to the base subtile, the blast centre is the base subtile (`FUN_0056bad0`, hostility hooked to this unit): it also checks every case against `dx^2 + dy^2 <= r^2` (**0 differ in 120,000**). d2d side: the real `Fight::area` on the float positions `(base + c / 1000) / 5` (the monster has a million life and the missile's damage is fixed at 10: hit = life lost).

| differential | cases | result |
|---|---|---|
| shape, game side (2 sites x radius 1..9) | 18 blocks of 361 units | 0 differ from the disc; 0 changed by random fractions; one record in every block; r^2 = r^2 |
| rooms (Blood Moor, seed 1) | 14 own + 14 near + 168 far pairs | found 14 / 14, 14 / 14, 0 / 168 (34 far pairs within 100 subtiles: 0 found; far minimum 62, near maximum 57 subtiles) |
| filter | 18 modes + 16 flag bits + not hostile | modes 0 and 12 out, 1..11 and 13..17 in; bits 2 and 3 of +0xc4 out when cleared, the other 14 in; not hostile out |
| radius | 20 levels x 2 | r^2 = 36 at 20 of 20; eax 1 in 40 of 40 |
| M1 seed 1 | 40,000 (20,000 U + 20,000 B) | game 17,202 hits, d2d 17,238; only-game 1,844, only-d2d 1,880 (9.3 %); U 6.0 %, B 12.6 %; md5 (8): cases 0cbf3628, game 4fc3571d, d2d 787b6bba |
| M1 seed 7 | 80,000 | game 34,576, d2d 34,622; only-game 3,615, only-d2d 3,661 (9.1 %); U 5.9 %, B 12.3 %; md5: a61e4f4b, 349d5878, 2ab657c4; radius 6 alone (kind B) 240 / 287 only-game / only-d2d of 4,436 |

Not covered: sizes 2 and 3 (read: none), monsters in motion, a game with a player owner (the owner here is a monster), the damage values (FUN_0056b9c0 hooked), levels other than Blood Moor, the real game's unit-flag values for buried / aloft / riding monsters, whether the d2d float position and the game's integer word correspond by floor (I).

## 6. The bot (described, not changed)

| id | finding | file:line | tag | impact |
|---|---|---|---|---|
| B1 | Meteor is `kind: ground` aimed at the monster's spot, the blast is a 113-subtile disc of integer subtiles around the target's subtile with one shared roll: aiming at a crowd's centre instead of the first target would hit more units for the same cast; the bot does not choose a centre | internal/combat/skills.go:29 (KindGround), internal/combat/classdata/skills.json (Meteor entry) | S (our code) + V run (rule) | none for a single target; for packs a design option (I) |
| B2 | Meteor's range in the bot is 30 subtiles; game.exe finds nothing when the target room is off the caster's near list, observed from 62 subtiles; 30 is below that | internal/combat/classdata/skills.json (Meteor entry, range 30) | V run + S | none: no change needed |
| B3 | the d2d backend finds units by float distance (M1): a soak on d2d sees a blast edge about 1 subtile off game.exe's, equal on average | internal/d2denv (backend) | S + V run | none in the mean |

## 7. PR candidate

**No PR candidate.** M1 is proven by a differential on the real `Fight::area` (mismatches before 7,276 of 80,000, and a one-line change cannot be tested against game.exe without fixing the mapping from float positions to game.exe's integer words, which is the inference I). A fix would be to compare floor(unit.x * 5) and floor(unit.y * 5) with floor(centre * 5) in `area`; whether upstream wants integer positions in that function (and which of the other hit functions that call it, cases 1, 3, 9, 13, 20 of `burst`, game.exe also routes through FUN_0056bad0: U) is a design question, the effect is a rim jitter without a bias, and the independent verifier (d2d-reviewer) has not seen it. If a later card proves the callers, the one-claim change plus `skg_test --blast` before / after (7,276 before, 0 after on the floor model) is the form.

## 8. Drill recipe (lab binary, fork verbs; for a later live card, not run here)

Verbs: docs/forkdbg.md, docs/forkcheats.md, docs/d2d-fork.md (`debug spawn`, `debug goto`, `cmd skill <id> X Y`, `step N`, `monsters`, `debug missiles`). Missing: a verb that pins a monster's life (`debug mon set`).

- G1 (M1, Q28): Meteor at a flat open point X, Y; for a ring of zombies at integer offsets 5, 6, 7 around the centre (each with known life, parked one per cast), count which lose life: game.exe loses life exactly where `dx^2 + dy^2 <= 36` on the zombie's subtile (use `monsters` positions as subtiles); d2d within 1.2 cells by float.
- G2 (M2): a Meteor aimed 62 to 100 subtiles away (use a teleport or a far click) at a zombie standing in a room not adjacent to the caster's: game.exe leaves it untouched (the fire still burns it), d2d hits it.
- G3 (M3): a buried maggot / an aloft vulture / a riding imp under a Meteor: hit or not on game.exe, compared with `targetable()`.
- G4 (Q31): nothing visible (the return value); skip.

## 9. Matrix cell updates (docs/research/sorc-skill-matrix.md; proposed text for the merger)

| Row | Cell | New value |
|---|---|---|
| 56 Meteor (line 46) | blast shape | V run (skground-trace.md): disc `dx^2 + dy^2 <= 36` on the integer subtiles of the units and the centre, centre test only, 113 subtiles; hostile, mode not 0 / 12, flag bits 0x4 / 0x8; the search covers the caster's room and its near list only; one roll shared; d2d: float hypot (M1, 9.1 % of rim cases differ, no bias), no room rule (M2) |
| rank 9 SKGROUND (line 255) | open U lines | Meteor blast shape: done. Open: Hydra's AI and HydraMissile, Frozen Orb and Thunder Storm native runs, the skill-delay refusal (G3 of sk-ground-trace.md) |

## 10. Not traced (U)

- The jump-table branches of FUN_0056b3e0 for types 0, 2, 3, 4, and the names of the +0xc4 bits.
- Other callers of FUN_0056bad0 (Fire Ball, Frost Nova, Nova and the like probably share it: `disasm.py --all` and a search for `call 0x56bad0`), which decides whether M1 is one area-rule for all of them.
- Whether d2d's float positions floor onto game.exe's words (I): a live drill, G1.
- Meteor's fire spread placement (FUN_005aaa90) and the fire's per-frame damage; Hydra's AI; Frozen Orb and Thunder Storm.

## Shared-file lines for the merger

(also in `openspec/changes/skground/merge-lines.txt`)

docs/decomp-queue.md, row 22: append to the status cell `; Meteor blast shape done (docs/research/skground-trace.md)`.
