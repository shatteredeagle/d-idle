# fid-missile-player (TASK-83, matrix rows SK03 SK12 SK08): player missiles, d2d against game.exe, re-checked on current upstream main

> status: finished 2026-11-08 by agent decomp6 (Sonnet). Result: **flight, range, roll and the Nova ring are equal (0 differences); one real deviation found and proven: Lightning (a LastCollide row without NextHit) strikes every touching monster once per cast on d2d, game.exe hands ONE unit a frame to the hit and skips only the last unit hit.** Local PR candidate `fid/missile-last-unit` (fork remote, head f45e45c8, two commits on upstream a5b970d5), not posted. Open (U): game.exe's order of two units found in one subtile; size-2 missiles against walls.

Tags: V run (game.exe's own code run by the emulator and compared with d2d's real code, byte for byte), V read, D (patched 1.14d tables), S (an upstream proof), I inferred, U unverified. Worker host, scratch tree = upstream main a5b970d5 (2026-11-08); the final-head runs use the candidate branch (a5b970d5 plus its two commits). Scripts: `tools/emu/d6_base.sh missile|light`, `tools/emu/sk6_flight.py`, `tools/emu/sk6_light_hit.py` (own prefix `sk6_` / `d6_`).

## Cases covered (so "0 differences" means something)

Skills: Fire Bolt, Ice Bolt, Ice Blast, Charged Bolt, Lightning, Chain Lightning, Nova, Frost Nova (the Sorceress missile set); missile rows firebolt, icebolt, iceblast, chargedbolt, lightningbolt, chainlightning, nova, frostnova (Missiles.txt, D). Paladin Holy Bolt is not run here (it is a different srvdofunc; its row is covered by the generic flight check only when its Missiles.txt row is in `missiles.py`'s list: it is not, U).

## Q1. Is the flight (speed, range, per-frame position, collision, end) equal? V run

| Run | Cases | Differ |
|---|---|---|
| `missiles.py 1500 7` (upstream's: arrow, spikes, Fire Bolt, Ice Bolt, Charged Bolt wiggle, Andariel's bolt, random Vel/Accel rows; game.exe FUN_0059fa30 + FUN_005ae1f0 against the port of `rules/missiles.hpp`, every frame's 16.16 position and how it ended) | 1,500 flights | **0** |
| `sk6_flight.py` (the same harness with rows Frost Nova, Nova, Ice Blast added; Accel -1000 every fifth frame is in the random rows too), size-1 rows, seed 11 | 2,500 flights | **0** |

Reach by game.exe's own run (`sk6_flight.py 4000 5`, flights that ran to their range, straight-line rows; distance in subtiles, frames):

| Row | Reach | Frames | Arithmetic (Vel, Range, D) |
|---|---|---|---|
| firebolt | 46.88 | 50 | 20 x 0.75 x 50 / 16 |
| icebolt, iceblast | 28.12 | 50 | 12 x 0.75 x 50 / 16 |
| chargedbolt | 26.0 to 34.2 (wiggle, 77 frames) | 77 | random walk |
| lightningbolt, chainlightning (per hop) | 35.16 | 25 | 30 x 0.75 x 25 / 16 |
| frostnova | 12.09 | 14 | Accel -1000 |
| nova | 11.45 | 13 | Accel -1000 |

Q3 (AC #3): `internal/combat/classdata/skills.json:1` ranges (Kolbot's, UNVERIFIED in the file) Fire Bolt 20, Ice Bolt 20, Ice Blast 20, Charged Bolt 10, Lightning 25, Chain Lightning 25, Nova 7, Frost Nova 5 are all inside these reaches: **confirmed as safe (conservative), V run**. They are not game.exe's numbers; the bot walks closer than a missile needs (finding B1).

## Q2. The damage roll and resistances (AC #1, roll) V run

`missile_hit.py gen 60000 1`: 104,000 cases (8 skills x levels 1..20 x synergy levels x mastery x resistance -100..99), min / max / life after the hit: **0 differ**; `gen 20000 7`: 64,000 cases, **0 differ**. The deviation of the earlier trace (d2d rolled the top 256th; SKMISS D1) is merged upstream (`tools/emu/missile_hit.py`, `test_missile_hit`).

## Q3. Frost Nova / Nova hits on a crowd, units per frame (AC #2, SK08) V run

`test_next_hit --next-hit 1 400` replayed through game.exe's FUN_005a8850 / FUN_005adf10 / FUN_005adb00 (NextHit state 0x56, NextDelay 4): **598 casts, 0 differ** (hits per monster per frame). The ring itself (64 missiles, tables 0x6e1288) is sk-nova-trace.md (S).

## Q4. Lightning: pierce and hit order (AC #2, SK12) V run, one deviation

Missiles.txt (D): `lightningbolt` LastCollide 1, CollideKill 0, NextHit 0, Size 2, Vel 30, Range 25. game.exe (V read, sk-lightning-trace.md Q3; V run below): FUN_005ae1f0 hands ONE unit a frame (the first that FUN_005a8730 passes) to FUN_005adf10 and the frame is over; the filter skips only the missile's last unit hit (FUN_0064a470), which FUN_005adf10 sets (0x5adfae). d2d: every monster touched in the frame is struck, each at most once per missile (`struck` list), only NextHit rows are limited to one a frame.

Differential (new, built on upstream's `next_hit.py` / `test_next_hit.cpp`): `tests/game/test_light_pierce.cpp` (d2d's real `Fight::fire` / `Fight::world` / `skill_missile_hits` on a Lightning cast at 1..6 standing monsters of sizes 1..3, 8 headings, levels 1..20, 3 x 11 x 3 x 8 boundary cases plus random packs; prints the contacts per frame and the hits per monster) and `sk6_light_hit.py` (replays those contacts through game.exe's own filter and hit). Command: `tools/emu/d6_base.sh light 1 400` (seeds 1 and 7).

| Build | Casts | Casts that differ | Hits game.exe / d2d | Frames with 2+ monsters hit | Casts where one monster is hit in two frames |
|---|---|---|---|---|---|
| upstream main a5b970d5, seed 1 | 1,192 | **44** | 1,239 / 1,206 (+2.7 %) | game.exe 0, d2d 36 | game.exe 36, d2d 0 |
| same, contact order reversed (`SK6_REVERSE=1`) | 1,192 | 41 | 1,243 / 1,206 (+3.1 %) | 0 / 36 | |
| candidate `fid/missile-last-unit` f45e45c8, seed 1 | 1,192 | **0** | 1,239 / 1,239 | 0 / 0 | 36 / 36 |
| candidate, seed 7 (400 -> 600 packs) | 1,392 | **0** | 1,595 / 1,595 | | 51 / 51 |

Both structural counts (no frame with two hits in game.exe; a monster hit again after another unit took the slot) hold in either contact order. The tests of the neighbours on the candidate: `test_next_hit` (0 differ, as before), `test_game_missiles`, `test_monsters`, `test_combat`, `test_game` all rc 0.

**Not proven (U):** which of two units in the same subtile game.exe finds first (its room's unit list order; the replay uses d2d's monster order, or its reverse, which moves 41 to 44 casts but not the structure). Pierce charges (Missiles.txt Pierce, stat 0x148, FUN_005ada80) are not in play for Lightning (`Pierce` is empty on the row, D); the Paladin's Blessed Hammer row has Pierce 1 and is outside this card.

## Q5. Mana timing, Charged Bolt count, Chain Lightning hops (carried, not re-run)

- Mana paid at the action frame, not at the command (old D2): upstream's `cast_mana.py gen 1` against `test_cast_mana --cast-mana` (a Sorceress self cast, with a get-hit cut): **53,628 cases, 0 differ** (V run, `tools/emu/d6_final2.sh`). The old SKMISS D2 / B3 deviation is merged and equal on the d2d side; the bot's `ManaBurnTracker` booking at the command (B2 below) is the bot-side remainder.
- Charged Bolt count min(24, 3 + lvl - 1) and the wiggle: count V read + D (FUN_005c9300), the wiggle is in `missiles.py` (the chargedbolt flights above, V run). The count itself on d2d: S (components), no oracle (U).
- Chain Lightning hop choice (FUN_005aa730 -> FUN_0056bd10: lowest unit id above the struck one's): d2d implements the same words (S, fight/missiles.cpp); no differential (U). Buildable: replay FUN_005aa730 natively with a unit list (as next_hit.py) against `Fight`; blocked on game.exe's unit id order for the test.

## Deviations (d2d against game.exe)

| id | what | d2d | game.exe | tag | differential | PR |
|---|---|---|---|---|---|---|
| D1 | a LastCollide row without NextHit (Lightning) | strikes every touching monster once per missile, several in one frame | one unit a frame, the last unit hit skipped, a unit may be struck again after another took the slot | V run (44 of 1,192 casts, +2.7 % hits) | yes, built and run (above) | candidate `fid/missile-last-unit` (U: unit order) |
| D2 | size-2 missiles (Lightning rows) against walls | the port in `missiles.py` tests the subtile alone | game.exe ends the flight at a wall earlier or on another frame: 370 of 2,500 flights (`sk6_flight.py` size-2 rows, all game-side wall ends), none in range or unit ends | V run against the PORT (upstream lists "Size 2/3 missiles" as not yet), I for d2d's own code | build: run game.exe's FUN_00650150 with a size-2 missile (collision mask) and d2d's `missile_fly` on the same wall patterns | none (U) |

Not deviations: roll, flight geometry/range of the size-1 rows, Nova hits (0 differences above).

## Bot findings (described, not changed)

B1 `internal/combat/classdata/skills.json:1`: Kolbot ranges 20 / 20 / 10 / 25 / 25 against reaches 46.9 / 28.1 / 26 to 34 / 35.2 / 35.2 (V run above). Casting from closer than needed costs walking time (walk-speed-trace.md: walking dominates a kill). Candidate: derive the bolt ranges from Missiles.txt (Vel x Range) data per build.
B2 `internal/combat/skills.go:175` (`PickAttack`) and `internal/combat/fight.go:92` (`ManaBurnTracker`) are the SKMISS B2/B3 findings; unchanged by this card (the mana timing is upstream-proven separately).
B3 Lightning gain: the plan's Lightning builds (`internal/plandata/build-fallback-lightning-v3.json`) would see about 3 % more hits per cast in packs on game.exe than on d2d; the bot needs no change, only d2d parity.

## PR candidate

`fid/missile-last-unit` in the fork scratch repo (remote: the private fork, pushed 2026-11-08, head f45e45c8): commit 1 the oracle (`tools/emu/light_pierce.py`, `tests/game/test_light_pierce.cpp`, a README row; before: 44 of 1,192 differ), commit 2 the claim (`components/game/fight/missiles.cpp` +7 lines, `tick.cpp` 1 line, a doc paragraph in `docs/research/re/missiles.md`; after: 0 of 1,192 and 0 of 1,392). Author and committer the numeric noreply identity of the fork's other PR branches, body `Agent: Claude Code`, no trailer. `pr_guard upstream/main`: 0 FAIL, 0 WARN; `presubmit --quick --base upstream/main`: 0 FAIL, 3 WARN (siblings: unit kinds; hit paths; doc-claims "never"), to answer in a PR body, none posted. Needs before any send: an independent verifier (d2-reviewer, own harness), the unit-order U above (or a PR claim that says so), the user's go.

Overlap check (read-only `gh` on the upstream repository, 2026-11-08): no open PR touches Lightning or the one-unit-a-frame rule; #158 (merged) did it for NextHit rows only, which is why this is the remaining half; the open issue #225 ("Missile pierce: game.exe counts down a missile stat, d2d rolls a percent", FUN_005ada80) is a different rule (Pierce flag rows, Lightning has none).

Usefulness for the bot: LOW. Only Lightning (and the other LastCollide pierce rows: Inferno, Blessed Hammer, Fist of Ares, mummy poison) are touched; Chain Lightning, Fire Bolt, Ice Bolt, Charged Bolt and the Novas end on the first unit (CollideKill 1) or are NextHit rows already handled. Early Sorceress play (Charged Bolt, Fire Bolt, Ice Bolt, Frost Nova) is unaffected; Lightning shows in the fallback Lightning builds only after clvl 12 or so (U).

## Drill recipe (lab binary, docs/forkdbg.md)

1. D1: Sorceress with Lightning, `debug spawn` three Fallen in a row 4 subtiles apart (`debug spawn fallen1 3`), `debug mon set <id> life <big>`, `debug cast <Lightning id> x y`, `step` and read each monster's life per tick (`dump stats` raw[6]); expect on game.exe at most one monster hit per frame and, when the bolt overlaps two adjacent monsters, a monster hit on two frames; on d2d today one hit each, several on one frame. Compare with game.exe on an instance of the bot using the same layout (monster life read each tick).
2. D2: Lightning at a wall one subtile from the caster in a corridor, `debug missiles` per frame: the last position and the frame it vanishes.
3. Unit order: two monsters in the same subtile cannot be placed; instead put two size-2 monsters side by side across the line and note which is hit first on game.exe (compare with d2d's index order).
