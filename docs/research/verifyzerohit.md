# Independent verification of SKZEROHIT: what a zero-damage hit still triggers (verifier, 2026-11-02)

> status: verifier report on two passes: the first (static) pass `docs/research/sk-zero-hit-trace.md` (on main) and the second pass with runs `docs/research/zerohitrun-trace.md` (branch `agent/zerohitrun`, not merged when this was written). Method: game.exe 1.14d functions read first (own disassembly, `tools/emu/mai2_dis.sh`), own model written from that reading, own harness `tools/emu/vzh_run.py` runs game.exe's real code under the emulator with seeds 101, 202 and 303 (seeds the authors did not use), and only then compared with the authors' traces and `tools/emu/zhr_run.py`. Tags: V run = game.exe code run under the emulator against this verifier's model; V read = read in the disassembly; D = data read out of game.exe's image or the MPQ tables; S = read in d2d's source or an earlier proof; I = inferred; U = unverified. Offline, no game, no d2d run, no Go code, nothing pushed.

## 1. Verdict in five lines

1. The core of both passes holds, now checked with an independent harness: in FUN_0057cee0 a hit with total 0 still knocks back a type that has the KB bit, a type without the KB bit has flag 8 turned into flag 4, a get-hit needs at least one point (only a stunned monster takes a get-hit at total 0), and the Lightning Enchanted bolts (id 17) fire on a no-recovery hit. **VERIFIED, 221,952 hit cases plus 6,918 event and bolt cases over 3 seeds, 0 differences** (section 3).
2. The second pass's corrections of the first pass are right: 0x5c is the length passed to a memset (the missile is 0xc3), id 17 also has a slot-2 handler that is FUN_005a29a0 itself with no mode test, and d2d does have the bolts (`Fight::boss_events`, fight.cpp:1679). **VERIFIED** (run for the game side, S for d2d).
3. Both passes left out the same gates. They do not change a zero-damage verdict for a normal monster, but the models are incomplete: state 0x36 makes FUN_0057cee0 return before any effect (0x57cf65); classes 0xf3, 0x14d and 0x2c1 never take block mode 6 (0x57d056); the bolt routine clears data+0x16 bit 0x100 when the cooldown stops it (0x5a29fe); the damage function FUN_0057c6c0 raises its own enchant event 3 on a flag-1 hit (0x57c83f, handlers only for ids 7 and 24) and clears flags 1 and 2 for a target FUN_00554200 rejects (0x57c6e9). The zhr_run harness never set state 0x36, used class 1 only and did not check the 0x100 bit, so its "0 mismatches over 165,296 cases" does not cover these gates.
4. The "no deviation" parts are not just d2d's formula worked out again: both harnesses (the second pass's and mine) execute game.exe's own code. But the d2d side of D1..D3 was only read, never run, so no deviation is V run on both sides. **No PR candidate.**
5. Claims that stay open: FUN_005ad730 (the missile hit) was not run by anyone (Q1, Q2 V read; my reading agrees, with one correction in Q2); "no wake on a hit" was not run by anyone; who raises event 2 and how late is U. I narrowed that last one (section 4, Q8).

## 2. My own model (written before reading the authors' scripts)

FUN_0057cee0(ecx game, edx attacker, [8] unit, [0xc] record), monster target (V read, then V run):
- flag 1: if record byte +0x64 is 0 and (+0x60 & 0xf0) is 0, +0x60 = FUN_0057ce30(); unit +0xb0 = +0x60 (0x57cf41..0x57cf5f).
- state 0x36 on the unit: if flag 2, set state 0x5c (FUN_00639db0(unit, 0x5c, 1)); return (0x57cf65..0x57cf8b).
- unit mode 0 or 0xc: return. Flag 8 on a class without MonStats2 bit 0xd: flag 8 becomes 4 and is written back. Call FUN_005a54f0.
- flag 2: FUN_0057ccb0, return. Flag 8: mode 0xd, return (no hit event). Flag 0x10: last mode 0x13 if flag 0x4000, class 0xf3 / 0x14d / 0x2c1 or no bit 6; else mode 6; return (no hit event).
- flag 4: stunned (state 0x15) or FUN_0057cb00 == 0 -> mode 3, then hit event 4; else the no-recovery tail (FUN_0064c040, unit +0xc4 |= 0x8000, last mode 0x13, hit event 4).
- flag 0x4000 alone: the tail. Flag 1 alone: total > 0 (signed) and |stat 0x160 byte - FUN_005a5650 byte| > 4 -> FUN_0064c040 and +0xc4 |= 0x8000, no event.

FUN_0057cb00(unit, record, hit class) = 1 for no recovery: frozen (state 1); record +0x28 nonzero and equal to the total (all poison); total < 0x100; total < maxlife / d; when total < maxlife / (d/2) a roll of 2 must be nonzero; when total < maxlife / (d/4) a roll of 4 must be nonzero; a monster without MonStats2 bit 3. d by hit class from the jump table at 0x57cbf8 (D, read out of the image): classes 2, 6, 10, 11 -> 8; 4, 8 -> 0x20; 5 -> 0x40; every other class -> 0x10.

Event walk FUN_005a4270: the unit must be a monster with a data block; flag = data+0x16 bit 3; the list (data+0x1c, nine bytes) is skipped whole when its FIRST byte is 0, otherwise all nine bytes are walked (a 0 in the middle does not stop it); handler = table 0x73c0b8 [id*6 + event]. Table (D, ids 0..42; the dwords after id 42 are other data): slot 4 only id 17 (FUN_005a2ba0); slot 2: ids 9, 10, 17 (= FUN_005a29a0), 18, 20, 21, 31, 32, 33, 34, 41, 42; slot 3: ids 7 and 24; slot 5: 19, 27, 29.

FUN_005a2ba0: needs the flag and unit mode != 3. FUN_005a29a0: needs the flag; if |game+0xa8 - data+0x18| < 10 it clears data+0x16 bit 0x100 and stops; else it stores the frame, sets bit 0x100, level = stat 0xc / 2 (signed, at least 1) and launches missile 0xc3 eight times (4 directions x 2, offsets from 0x6e2178 / 0x6e2188 = (-1,0) (0,1) (1,0) (0,-1), D), owner = the unit, flags 0x21, path function 0x5c9290.

FUN_005ad730 (V read only): flags 4 / 0x4000 by the missile row's byte +4 against 0x6ce280 = 0x40 / 0x6ce284 = 0x80, flag 8 by a d100 roll under row byte +0xae, all skipped when the target has state 0x36; then FUN_0057dfb0: a nonzero result with no recovery class and no KB percent clears flag 1 and sets 0x4000; ANY nonzero result clears flag 1 (0x5ad855); result bits 0x11 -> flag 0x10, 8 -> 0x200, 2 -> 0x100. Then FUN_0057c6c0 only with flag 1, FUN_0057cee0 always (0x5ad8b3..0x5ad8d3).

## 3. Run: counts (`tools/emu/vzh_run.py SEED NRANDOM`)

Native: 0x57cee0, 0x57cb00, 0x5a43a0, 0x5a4370, 0x5a4270, every handler the walk reaches, 0x5a2ba0, 0x5a29a0 up to and including the launch loop. Hooked: state and stat getters, MonStats2 bits, the rolls 0x472210 (scripted values, so the exact branch of the size test is checked, not only "one of two"; the RNG itself is S pr-get-hit-rule-proof.md), 0x620bb0, 0x5a54f0, 0x57ccb0, 0x5734c0, 0x64c040, 0x639db0, 0x57ce30, 0x5a5650, the mode fill 0x5a7e60 and apply 0x5a7c20 (apply modelled as unit +0x10 = mode in about 80% of the random cases and not at all in the rest), memset (real), launch 0x59fa30 (struct recorded), position getters. Compared fields per case: mode calls, last-mode calls, rolls asked, launch count and every launch struct, state set, FUN_005a54f0 / FUN_0057ccb0 / FUN_0064c040 calls, record flags after, unit +0xb0, +0xc4, +0x10, data+0x16, data+0x18, record +0x60.

| Seed | Part A cases (grid 13,824 + roll grid 20,160 + random) | differ | with a roll | zero-total cases (live mode, no flag 2, no state 0x36) | of them: hit event / bolts / mode 0xd / get-hit | state 0x36 cases | special-class flag-0x10 cases | Part B (walk, ev 2 and 4) | Part C (bolt routine) |
|---|---|---|---|---|---|---|---|---|---|
| 101 | 53,984 | 0 | 5,059 | 3,304 | 1,868 / 201 / 671 / 170 | 7,952 | 2,686 | 2,240 / 0 (56 faults of other ids' handlers, not counted) | 66 / 0 |
| 202 | 73,984 | 0 | 5,090 | 6,312 | 3,464 / 326 / 1,348 / 312 | 8,936 | 5,152 | 2,240 / 0 | 66 / 0 |
| 303 | 93,984 | 0 | 5,114 | 9,014 | 5,050 / 477 / 1,941 / 364 | 9,905 | 7,833 | 2,240 / 0 | 66 / 0 |

Part A total 221,952, plus parts B and C 6,918: 0 differences. Bolts never fired with the unit in mode 3 after the event-4 walk (`fire_gh` 0); the event-2 walk fired 4 times with the unit in mode 3 (part B, every seed). A list holding id 17 twice fired once (154 / 267 / 431 cases), and bit 0x100 was cleared by a cooldown stop in 91 / 197 / 308 cases.

My own fault on the way: the first run showed 2 differences, both in the roll branch; my harness gave the run and the model the same roll list, and the hook used it up. After copying the list: 0. Recorded as a harness fault, not a game.exe fact.

## 4. Claim by claim

First pass (sk-zero-hit-trace.md):

| Claim | Verdict | Why |
|---|---|---|
| 1. FUN_0057cee0 is called whether or not the record holds damage | HOLDS WITH LIMITS | My reading agrees (0x5ad8b3..0x5ad8d3); nobody ran FUN_005ad730. Missing: FUN_0057c6c0 can clear flags 1 and 2 first (0x57c6e9), and state 0x36 skips the flag rolls (0x5ad79e) |
| 2. Knockback does not look at damage; a non-KB type turns 8 into 4 | VERIFIED (V run) | 1,348..1,941 zero-total mode-0xd cases per seed; limit: state 0x36 and mode 0 / 0xc stop everything |
| 3. Get-hit needs a total of at least 0x100; frozen, all poison, stun exception | VERIFIED (V run), incomplete | The MonStats2 bit 3 test (0x57cbd2) is missing in the first pass (the second pass caught it); it runs only after the rolls |
| 4. No wake / aggro in this path | NOT VERIFIED | FUN_005a54f0 was hooked by both harnesses; nobody ran it or checked the aware state. Stays V read (first pass) |
| 5. One on-hit enchant (id 17, slot 4), bolts only when not in get-hit | slot 4: VERIFIED (table D, walk V run); "only when not in get-hit": WRONG as a statement about the bolts | slot 2 of id 17 is FUN_005a29a0 with no mode test (part B: fires in mode 3). "Skill id 0x5c": WRONG (memset length; missile 0xc3, part C) |
| 6. d2d has no on-hit bolts | WRONG | fight.cpp:1679 (S, read on the worker host's upstream checkout) |
| Q2 "a block / avoid result with no recovery class and no KB clears flag 1 and sets 0x4000" | HOLDS WITH LIMITS (V read) | any nonzero FUN_0057dfb0 result clears flag 1 (0x5ad855) and bits 0x11 set flag 0x10; so a blocked missile with a KB percent can still roll flag 8, which FUN_0057cee0 tests before 0x10 (I: a block that still knocks back) |
| Q4 "slots 0..3 belong to other events (spawn, state change)" | WRONG in part | event 3 is raised by the damage function (0x57c83f) on a flag-1 hit; ids 7 and 24 have slot-3 handlers (D). Not part of the zero-damage answer, since a zero hit with flag 1 still reaches it (I) |

Second pass (zerohitrun-trace.md):

| Claim | Verdict | Why |
|---|---|---|
| Q1 flag 1 alone, total 0 | VERIFIED | same rule, my run (signed total > 0, the byte difference > 4) |
| Q3 model equals the run in 165,296 cases | HOLDS WITH LIMITS | equal on what it covered; its model has no state-0x36 gate and no special classes, and its harness never set them (0x639df0 hooked to 0 for 0x36, class 1 only). Mine covers both, 0 differences |
| Q4 size test is "mode 3 or the tail" | VERIFIED, stronger | with scripted rolls the exact branch matches in 15,263 roll cases over the three seeds |
| Q5 slot 4 only id 17; first byte 0 skips the list; flag and mode-3 refusal | VERIFIED | my table dump covers ids 0..42 (theirs 0..39; 40..42 have no slot-4 handler either) |
| Q6 bolt routine: 10-frame cooldown, stored frame, bit 0x100 set, level stat 0xc / 2, 8 x 0xc3, offsets | VERIFIED, incomplete | a cooldown stop CLEARS bit 0x100 (0x5a29fe), not covered by them; launch struct fields (owner, 0x21, 0x5c9290, the per-direction counter 0..1) checked here |
| Q7 event 2 fires with no mode test | VERIFIED | part B, 4 fires in mode 3 per seed |
| Q8 who raises event 2: not found | NOT VERIFIED, narrowed | FUN_005a4370 has no direct caller; its only pointer is entry 7 of the unit event dispatch table 0x6e2490 (read at 0x5a7fbf, the dispatcher skips a frozen unit that FUN_005541b0 rejects); timer type 7 is queued through FUN_005417d0 from more than ten sites (0x54ea84, 0x588acf, 0x5894ff, 0x58a59c, ...). Which one follows a get-hit, and the 2-frame delay d2d's comment states, stay U |
| D1 (d2d keys the bolts on a life drop, every 10 x 40 ms, boss = unique or superunique) | HOLDS (S) | fight.cpp:1653, 1679..1686 and fight.hpp:676..683 read on the worker host; the game side is V run. Extra difference not in their list: d2d turns each bolt up to 25 degrees at random (fight.cpp:1685), game.exe uses fixed offsets plus the path function 0x5c9290 (not run, U) |
| D2, D3 | HOLDS as I (read only) | no d2d run |
| B1 (bot spacing rule for Lightning Enchanted) | not re-checked | bot file not read in this pass |
| "No PR candidate" | AGREED | no d2d differential exists |

## 5. Ledger rows

The first pass added no ledger row; it only had a BACKLOG line ("static, V read"), which is accurate. The second pass adds `missiles 195 READ-ONLY` (88 cases, event walk 11,456 per seed). Earned as READ-ONLY: the game side was run, d2d was not, and READ-ONLY is the right class. Two corrections for the merger: the row describes who launches `lightunique` and with what level and cooldown, not the missile's flight or hit roll (the subject of the missiles subsystem), so its note should say "launcher rule only"; and its note should name the gates it did not cover (state 0x36, bit 0x100 clear). No TESTED-0-DIFF or TESTED-DEVIATION row is earned by either pass.

## 6. Deviations, bot findings, PR

D1..D3 as in the second pass (all d2d sides S / I), plus D4 (I): d2d's bolt directions are randomised by up to 25 degrees, game.exe's are fixed compass offsets steered by 0x5c9290. Differential for D1 (buildable, not built): run `Fight::boss_events` with a unique carrying umod 17 under scripted hits (total 0, 1 point, big; frames 0, 9, 10, 40) and compare the launch frames with `vzh_run.py` part A's outcomes. No bot findings of my own (bot not read). No PR candidate.

## 7. Drill recipe (to settle the U items; nothing run)

1. Event 2 timing: in the lab (docs/forkdbg.md; lab binary docs/d2d-fork.md, docs/forkcheats.md) set an execute break on 0x5a4370 and on 0x5a43a0 while hitting a Lightning Enchanted unique with a big blow; log the frame of each and the queuing site (return address of FUN_005417d0 with type 7).
2. Wake: run FUN_005a54f0 natively in `vzh_run.py` (unhook it) with a target whose AI state is known, and read which fields it writes.
3. State 0x36: name it from states.txt row 54 (`mpq-table.sh states`) and check which skills set it.

## Reproduce

```sh
tools/emu/run-on-worker.sh --ttl 1500 vzh_run.py 101 20000 > /tmp/d2agent-<job>/r101.txt 2>&1   # also 202 40000, 303 60000 (--ttl 2400)
tools/emu/run-on-worker.sh --ttl 300 vzh_xref.py 5a4370 d6e2470                                   # event-2 pointer and the dispatch table
tools/emu/run-on-worker.sh --ttl 120 mai2_dis.sh 57cee0 170                                       # also 57d0b6, 57cb00, 5a4270, 5a29a0, 5ad7a0, 57c6c0, 5a7f90
```

## Shared-file lines for the merger

See openspec/changes/verifyzerohit/merge-lines.txt.
