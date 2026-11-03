# Sorceress rest of the skills, part 2: Inferno, Blaze, Enchant run in game.exe; the burst rim on the real d2d build

> status: card sorcrest2, 2026-11-02, offline (no game, no d2d slot, no Go). Continues `docs/research/sor-rest-trace.md` (D1-D6, B1-B4), whose Inferno, Blaze and Enchant rows were V read only. The earlier doc is not edited; the corrections it needs are listed in section 5.

Tags: **V run** game.exe's real code run under the emulator by this agent (hooks named per script); for d2d, the real `Fight` of the scratch build (upstream main 7d5b3a5c plus the SKGROUND2 commits, the same build as sor-rest-trace); **V read** disassembly addresses; **S** d2d source; **I** inference; **U** unverified. Scripts: `tools/emu/sr2_*` (run with `tools/emu/run-on-worker.sh`).

## Answer

- **D4 Inferno (V run, game side only).** One call of the do FUN_005c8ca0 makes **at most one flame**, and none when the current-skill record's field +0x18 is 0 (48 cases: caster type 0 and 1, frames 1 / 2 / 100 / 100000, field 0 / 1 / 2 / 7 / 1000 / 200000: 0 flames in the 8 field-0 cases, 1 flame in the other 40). The call never schedules a next call by itself: 0 of 48 cases schedule an event, because the field is set to 1 before the frame is compared with it (a frame at or over 1 ends the channel). So game.exe's flame cadence is the rate of do calls, set by the caller (the player's cast sequence events), not by the game frame. d2d launches one flame per missile frame while the cast mode lasts (S). The rate of game.exe's do calls for a held cast: U (needs the player sequence, section 6).
- **The earlier Q8 had the unit type backwards (V run).** The `+0x44 = 0xb00` write and the schedule at frame + 2 sit in the branch for unit type 1 (a MONSTER), not the player (type 0): a player's call ends in FUN_00553c70 and returns (48 cases: type 0 gives t553 = 1, type 1 gives ended = 1 and `+0x44 = 0xb00`, 0 events either way).
- **D5 Enchant (V run, game side; d2d side S).** FUN_005ca030 puts the state on the picked unit when the ally test FUN_00554de0 passes, else on the caster (6 cases: picked none / ally / foe x test 0 / 1: the state lands on the caster in 4, on the picked unit in the 2 with test 1). The state is state 16 with stats 48, 49 (`firemindam`, `firemaxdam`) and 119 (`item_tohit_percent`), the three aurastat rows with a nonzero calc; the calcs are evaluated for the CASTER, whoever carries the state; duration is the row's `auralen` calc. d2d: caster only (S, `fight.hpp` self_cast). Not run on d2d: no command reaches an ally target.
- **Blaze (V run, game side).** The do FUN_005c9c10 makes the state object (state 13 `blaze`, owner and target the caster, skill 46, the level, duration = the `dm12` calc) with stats 0x15e / 0x15f (skill id, level); it makes **no missile and no client event** (3 levels: 0 missile makers, 0 event calls; the row's three client-slot words are 0xffff on the loaded table). The earlier Q9's "client overlays `firewall`, `firesmall`, `firemedium` from the do" is wrong for the loaded row. Whatever lays the trail runs elsewhere (U, section 6).
- **D2 burst rim (V run on d2d; the game side is the integer-subtile rule, I).** A Fire Ball burst on the real d2d `Fight` with a monster standing between subtile centres: **16,875 cases (3 sizes x 15 x 15 offsets x 25 sub-subtile positions), 549 differ from `dx^2 + dy^2 <= 16` on the monster's integer subtile** (333 in d2d, out in game; 216 out in d2d, in game; 183 per size). The same count for Glacial Spike level 8 (radius 4 by `aurarange`). At subtile centres the two are equal (sor-rest-trace Q4, 3,375 cases).
- **PR candidate: none.** D2 and D5 are not proven as deviations of d2d against game.exe on both sides (section 4).

## 1. Questions answered

| Q | Question | Answer | Where | Tag |
|---|---|---|---|---|
| Q11 | Inferno: how many flames does one do call make? | At most one: `FUN_005c8ca0` calls the missile maker FUN_0059fa30 once (flags 0x8020, Range = calc1 clamped to 1, owner = caster) only when FUN_006444a0 (the record field +0x18 of the unit's current skill, FUN_00620250 -> FUN_006439a0) is nonzero. 48 cases | `tools/emu/sr2_inferno.py`; 0x5c8d1b (`call 0x6444a0`, `je 0x5c8d8d`), 0x5c8d88 (`call 0x59fa30`) | **V run** (game) |
| Q12 | Does the do re-arm itself? | The field is written 1 by FUN_00644560 at 0x5c8d93 on EVERY call, flame or not (flag_after = 1 in 48 of 48 cases). So the first call after the field was 0 makes no flame and arms the next. What sets the field to nonzero before the first call, and what clears it (FUN_005c8b41 elsewhere clears it after use): U | 0x5c8d90..0x5c8d93; other users of the field: 5c8b41 / 5c8b4d (`call 0x6444a0`, `push 0`, `call 0x644560`) | V run (set), U (the writer before the first call) |
| Q13 | Does a call schedule the next one? | Never in 48 cases. After the setter the compare `cmp [game+0xa8], FUN_006444a0(rec)` (0x5c8dc2..0x5c8dcd, `jge 0x5c8e0f`) sees field = 1, so every frame >= 1 ends the channel through FUN_005c8c10 (type 1 only; a player returns at 0x5c8da1..0x5c8db7 before the compare). The schedule branch (`call 0x5417d0` with frame + 2, 0x5c8df2) needs frame < 1: not reachable in a real game (frame 0 not run: I) | `sr2_inferno.py` | **V run** |
| Q14 | Who calls the do again for a held Inferno? | Not the do. The player's skill sequence repeats its action event (Skills `seqnum 6`, `seqinput 10`, `repeat 1`, anim SQ, read in sor-rest-trace section 1) and the start function FUN_005c8fa0 with the state-0xc timer present moves the timer to frame + 6 (V read, sor-rest-trace Q8). The call rate is therefore the sequence's action-event rate (U: not run) | sor-rest-trace Q8 | V read, U for the rate |
| Q15 | Enchant: whose state? | `FUN_00553540` (the cast's picked unit) and `FUN_00554de0(game, caster, picked)`: pass -> picked, fail or none -> caster. 6 cases | `sr2_enchant.py`; 0x5ca09c..0x5ca0c7 | **V run** |
| Q16 | Enchant: which stats and whose calcs? | Six slots (ids at Skills +0x54, calc pointers +0x68): a slot counts when the id is >= 0 and below the stat count and the calc is nonzero; here stats 48, 49, 119 (3 slots), calcs evaluated with the caster (`push ebx` at 0x5ca167). The state id is 16 | `sr2_enchant.py` (calc hooked to a fixed 1000, the calcs themselves are V run in sor-rest-trace Q7) | **V run** |
| Q17 | Blaze: what does the do make? | State 13 with stats 350 / 351 (0x15e / 0x15f), duration = calc `dm12`; 0 missiles, 0 client events. The passive-stat function FUN_005c6dc0 runs once; the aura event path FUN_005c0b50 / FUN_0056e740 is skipped because Skills +0x84 is 0xffff | `sr2_blaze.py` (levels 1, 8, 20; duration calc hooked to the V run values 117 / 329 / 428); 0x5c9d17 (`cmp word ptr [esi + 0x84], 0`, `jl`) | **V run** |
| Q18 | Burst rim: does d2d take in the same set when a monster is between subtile centres? | No: 549 of 16,875 cases differ from the integer rule, in both directions (d2d reaches 3.99 subtiles from the missile centre at most and misses a monster the integer rule takes in at up to 4.53) | `tools/emu/sr2_test.cpp` (d2d `Fight::area`, hypot x 5 <= 4); `sr2_run.sh`, `sr2_rimsum.sh` | **V run** (d2d), I (game maps a unit to its integer subtile) |

## 2. Deviations (d2d vs game.exe)

| id | what | d2d | game.exe | tag | differential | candidate |
|---|---|---|---|---|---|---|
| D1 | burst roll (carried over from sor-rest-trace D1) | one draw per monster | one roll per burst | V run both sides (400 trials, see that doc) | exists | no (a restructure; section 4) |
| D2 | burst rim between subtile centres | float position, `hypot x 5 <= r` from the missile's float position | integer subtile of the unit, `dx^2 + dy^2 <= r^2` | d2d V run (549 / 16,875); the game's use of the integer subtile for the test: V read + I (FUN_006492a0 on the units' subtile words), the centre case V run (3,375) | d2d side built (`sr2_test.cpp`); the game side for off-centre positions is the rule itself because a game unit cannot stand between subtiles. To close it: run FUN_0056b7e0's unit search on a unit whose position words are set from a pathing run (a walked unit) | no (see section 4) |
| D4 | Inferno flame cadence | one flame per missile frame while the cast mode lasts (`player_shots`, once per frame) | one flame per do call; no self-scheduling (V run); rate = the sequence's action events (U) | game V run (48 cases); d2d S; the rate U | game side built; d2d side: a `Fight` with `channel` set and the cast mode held, count `launch`es per frame (not built: the cast mode needs the player's animation state); the oracle for the game rate: the cast-rate harness `tools/emu/cast_rates.py` / `cast-trace-gameexe.md` hooking FUN_005c8ca0 | no |
| D5 | Enchant target | caster only | the picked unit when FUN_00554de0 passes, else the caster | game V run (6 cases); d2d S | game side built; d2d side not reachable by a command (`debug cast 52 unit <id>` in the lab drill R5) | no |
| D7 | Blaze's cast itself | "a self state with a trail" (the trail is d2d's reading of missile do 5) | the cast makes only the state and stats; no missile, no overlay event | game V run (3 levels); d2d S (reading) | game side built; the d2d side: count missiles right after the cast (a `Fight` with the Blaze skill) | no |
| D3 / D6 | as in sor-rest-trace | | | | | |

## 3. Bot findings

B1-B4 as in sor-rest-trace section 6 (unchanged). New:

| id | finding | where | tag | impact |
|---|---|---|---|---|
| B5 | no Inferno channel model: if a plan ever casts Inferno, one cast is NOT one flame per frame; the flame count is the sequence's action-event count, and the first do call after the record field is 0 makes none | `internal/combat/castgap.go:19-24` (no entry), `internal/combat/skills.go` | V run (rule), S (code) | none today (no call path, B3) |
| B6 | the bot's blast-radius reasoning, if added for Fire Ball / Glacial Spike, must use the integer-subtile disc of radius 4 (about 0.8 cell) measured from the missile's subtile, not a float circle; a float circle misclassifies about 3% of off-centre positions (549 of 16,875) at the rim | `internal/combat/castfirst.go:95` (the crowd rule), `internal/combat/skills.go` `PickAttack` | V run (d2d side), I (game rule) | none today |

## 4. PR candidates

**None.** D2: the d2d side is V run, but the game side for positions between subtile centres is the rule applied to a unit's integer subtile (I): game units are placed by integer subtile words, and a fix would be a floor in `Fight::area`. The step that would make it a candidate is a game.exe run with units at subtile positions produced by a pathing run, so a d2d monster's float position can be mapped to a game subtile (a statement of the mapping, `floor((position x 5))`, is itself U). D5: needs a d2d run of an ally target, which no command can make. D4: no d2d run, and the game's rate is U. D1 stays a follow-up card with SKGROUND D5 (sor-rest-trace section 7).

## 5. Corrections to sor-rest-trace.md (proposed text)

- Q8: replace "For a player it then writes unit field +0x44 = 0xb00 and either schedules the next event ... or ends the channel (FUN_005c8c10 ...)" by: "For a MONSTER (unit type 1; V run, sr2_inferno.py) it writes +0x44 = 0xb00 and ends the channel; a player (type 0) ends the call in FUN_00553c70. Neither schedules a next call (0 of 48 cases)."
- Q9: delete "FUN_005c0b50 and, for each of the three client slots ..., FUN_0056e740 (the aura events ...)" as the shape of Blaze's cast: the three client-slot words of the loaded row are 0xffff (V run, sr2_blaze.py) and the do makes no missile and no event.
- D4 row: game side V run (48 cases), the flame rate is the sequence's, U; D5 row: game V run (6 cases); add D7 (Blaze).
- Section 11 "Not done": a real-build run for D2 is done in part (this doc); D4 d2d side not.

## 6. Drill recipe additions (lab binary; not run here)

Verbs as docs/forkdbg.md, lab binary as docs/d2d-fork.md / docs/forkcheats.md.

- R4b (Inferno rate): `debug points 41 1`, `debug cast 41 <x> <y>` held for 50 frames, `debug missiles` each frame: on d2d one `infernoflame1` row per frame (S); the number game.exe makes needs the real game: count rows per second at fcr 0 and compare with 25 per second; U.
- R7 (D2 rim): `debug spawn` a monster, move it by `debug mon set <id> x <x> y <y>` (if the verb takes a float position; U) 4.2 subtiles from the cast point; `debug cast 47` at the cast point; life drop yes or no; the integer rule says in when the monster's subtile is within 4 by `dx^2 + dy^2`.
- R5b (D5): as R5 of sor-rest-trace; read the state on both the merc and the caster.
- R6b (Blaze): `debug cast 46`, `debug missiles` at once: expect 0 missile rows from the cast itself (game.exe, V run); walk; a trail's rows are the unexplained part.

## 7. Matrix cell updates (docs/research/sorc-skill-matrix.md, proposed text for the merger)

- Row 41 Inferno: (a) the do makes at most one flame per call and does not re-schedule (V run, 48 cases, sorcrest2); the call rate is the sequence's (U); (b) d2d one flame per frame (D4, S).
- Row 46 Blaze: (a) the cast makes the state only (state 13, stats 350 / 351, duration `dm12`), no missile, no overlay (V run, 3 levels); (b) d2d trail from missile do 5 (S; D7).
- Row 52 Enchant: (a) the state goes to the picked unit if the ally test passes, else the caster; stats 48 / 49 / 119 with the caster's calcs (V run, 6 cases); (b) d2d caster only (D5, S).
- Rows 47 / 55: (b) burst rim between subtile centres: 549 of 16,875 off-centre cases differ from the integer-subtile rule (V run, d2d side; D2).

## 8. Differentials (V run)

Worker host, nice, 1 core. The d2d probe is built from `tools/emu/sr2_build.sh` (one file, `ninja -j2`, 3 steps, on the SORCREST scratch build); the game scripts run next to upstream's `tools/emu`.

| What | Command | Result |
|---|---|---|
| D4 Inferno do | `tools/emu/run-on-worker.sh sr2_inferno.py` | 48 cases: flames 0 for field 0 (8), 1 for the other 40; events 0 in 48; type 1: ended 1 and `+0x44 = 0xb00`; type 0: FUN_00553c70 1 |
| D5 Enchant | `run-on-worker.sh sr2_enchant.py` | 6 cases: landed caster 4, picked unit 2 (ally test 1); stats 48, 49, 119; state 16 |
| Blaze | `run-on-worker.sh sr2_blaze.py` | 3 levels: state 13, stats 350 / 351, 0 missiles, 0 events |
| D2 rim, d2d | `run-on-worker.sh sr2_run.sh fireball 47 1`, `sr2_run.sh glacialspike 55 8`, `sr2_rimsum.sh fireball 1` | 16,875 cases each, 549 differ each; 333 in d2d / out in game, 216 out in d2d / in game |
| disassembly | `run-on-worker.sh sr2_dis.sh ADDR COUNT [ADDR COUNT ...]` | ranges in section 1 |

Hooked, not the claim (all runs): the target resolver FUN_0056d2c0, the current-skill lookup FUN_00620250, the calc FUN_00646ca0 (fixed values; the calcs are V run in sor-rest-trace Q7), the missile maker FUN_0059fa30 (counted), the scheduler FUN_005417d0 and FUN_00540e60 / FUN_00553c70 / FUN_00639db0 / FUN_00639df0 / FUN_005c8c10 (recorded), the state maker FUN_0056e970, the stat adder FUN_00627150, the state setter FUN_00639e30, the picked-unit lookup FUN_00553540 and the ally test FUN_00554de0 (the case's values). The units are fake blocks (only the type word and the fields read by the unhooked code are set), not real player or monster units: a run of the whole sequence engine is not covered.

## 9. Not traced and dead ends (U)

- Dead end: Inferno's "frame vs FUN_006444a0" comparison, left open in sor-rest-trace Q8, turned out to be a compare with the field the function just set to 1 (V read at 0x5c8d93 and 0x5c8dc2, V run): it is not a duration. The field's real writer before the first call is not found (85 call sites of the setter FUN_00644560 in the whole disassembly, about 50 of the getter FUN_006444a0; not read one by one).
- Not run: the player's cast sequence (seqinput / repeat) that decides the do call rate; Blaze's trail (state tick); the Enchant ally lookup FUN_00553540's choice of unit; Enchant stats on the receiving unit's damage; d2d side of D4, D5, D7; frame 0 for Inferno.
- `tools/emu/sr2_*` use fake unit blocks, as sk_mana.py does for the mana check.

## Shared-file lines for the merger

See `openspec/changes/sorcrest2/merge-lines.txt`. Ledger: row 55 (41, 46, 52) re-pointed here with the new evidence, row 53 note extended (D2); `docs/coverage/rule-coverage.md` regenerated.
