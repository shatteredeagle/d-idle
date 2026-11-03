# verifystateev2: independent verification of SKSTATEEV (both passes)

Status: independent verifier pass of `sk-state-events-trace.md` (first pass, static, V read) and `stateevrun-trace.md` (second pass, card stateevrun, emulator runs). Order of work: I read the disassembly of FUN_0057dbf0, FUN_0057d4f0, FUN_005c0c30, FUN_0057c1e0, FUN_0057c6c0 and FUN_0057b7d0 myself (`tools/emu/vse_dis.py`) and wrote the model below. I then ran game.exe 1.14d's own code under the emulator with my own harness and seeds (`tools/emu/vse_run.py`, entry `vse_run.sh`). Only after that did I read the authors' `ser_events.py` and compare. Tags as in AGENTS.md: **V run**, **V read**, **D**, **S**, **I**, **U**. No live game, no d2d build was run. No code or disassembly is pasted.

Runs (the worker host): `tools/emu/run-on-worker.sh vse_run.sh 400 4242` and `tools/emu/run-on-worker.sh --ttl 1500 vse_run.sh 20000 90210`. The counts below are from seed 90210. Seed 4242 gave the same shapes, with 0 exceptions in every check.

## Own model (from my reading, before any run)

- **Gate FUN_0057dbf0** (ECX game, EDX attacker; stack: defender, record, extra). If record +4 has any of 0x8380, or bit 0 (hit) is clear, it goes straight to the queue call FUN_0057ca80. Otherwise, if record +0 bit 1 is clear, it first calls FUN_0057b7d0. Then it calls FUN_0057c1e0 (resistance pass, event 11). It sums the element fields (+0x38 too for a monster attacker) and sets record +4 bit 1 when the sum exceeds the defender's life. Then it queues.
- **Resolver FUN_0057d4f0** (ECX game, EDX attacker; stack: defender). It finds the queued record (attacker +0xac list, keyed on both units' type and id) and copies 0x70 bytes. Reach FUN_00622c40(attacker, defender, mode), with mode = 2 x (attacker type == 1) + 1, so **1 for a player and 3 for a monster**. Reach fails: FUN_0057c9f0 only. Hit bit set: if the attacker (type 0 or 1) has unit +0x10 == 0 (I: the death mode), **return at once: no damage, no events, no FUN_0057c9f0**. Otherwise it sets the copy's +0 to 0x20, calls FUN_0057c6c0(defender, **0**, copy), then FUN_00621e40 / FUN_00623c20 side calls and FUN_0057d3d0. Hit and miss then share: dispatcher event **7 with (attacker, defender)**, event **3 with (defender, attacker)**. On a hit it re-checks the dead attacker, then FUN_005d10c0; then FUN_0057cee0 and FUN_0057c9f0.
- **Dispatcher FUN_005c0c30** (ECX game, EDX event id; stack: unit, other, record). It walks the list at **first stack arg +0x90**. On an id match it calls node +0x14 with (unit, other, record, node +0xc, node +0x10). A node whose +2 has bit 1 set, or whose +4 is 0, is unlinked and freed. So the owner is the first stack argument.
- **FUN_0057c1e0**: stats 0x22 / 0x23 pre-reductions, FUN_0057c060 percent scale (rows whose +0x20 flag is non-zero, only when the percent is not 100), **event 11 (defender, attacker, record)**, FUN_0057c140, then the 12-row FUN_0057bf80 loop. Rows with +0x1c set (rows 9..11) are skipped unless the attacker is a monster for which FUN_0063ee90 is true. Total to +0x4c.
- **FUN_0057c6c0** (stack: defender, flag, record). It calls FUN_0057c1e0 only when **flag != 0**. Events, unless record +4 has 0x20 or lacks bit 0: with flag 0, **5 (attacker) then 1 (defender)**. With flag != 0 they need record +0 bit 0x20, and then fire **6 (attacker, skipped when record +0 bit 0x80) then 2 (defender)**: the missile pair. Life is set, and kill events fire when record +4 has bit 1 and lacks 0x20: **10 (defender, attacker) then 9 (attacker, defender)**.
- **FUN_0057b7d0** event 3 site (0x57bcad): owner the defender. It is skipped when record +4 has 0x20. It is also skipped when the attacker is not a player and **FUN_0063ee90(attacker) is 0** (local at 0x57ba5b; V read: whether that local is set on every path before 0x57bc93 was not checked).

## Own runs (V run, game.exe code)

| Part | What runs natively | Cases | Result |
|---|---|---|---|
| X0 | table at 0x732980 read from the loaded image | 12 rows | field offsets 0x08 0x10 0x1c 0x24 0x20 0x30 0x34 0x2c 0x28 0x38 0x3c 0x40; +0x1c set on rows 9..11; +0x20 clear on rows 5..7 (V run of the image load) |
| X1 | real FUN_0057c1e0, 1..4 random element fields, random resistances; **game-vs-game oracle**: P = absorb a_i at event 11; Q = input F_i - a_i, no absorb; R = no absorb | 20,000 | event 11 exactly once, owner defender, other attacker: 20,000. Fields at event 11 equal the raw input: 20,000. P == Q on every field and on +0x4c: 20,000. Discriminating (Q differs from the after-order value R - a): 17,928, all P == Q, 0 equal to the after order |
| X2 | real gate + resolver, markers on the callees; attacker type 0/1 x alive/dead x hit/miss x reach x block bits {0, 0x80, 0x100, 0x200, 0x8000} x record +0 {0, 2}, random extra bits | 160 cells x 100 = 16,000 | every cell has exactly one shape. Reach mode 1 for type 0, 3 for type 1 (8,000 each). In reach, events E7(owner A) then E3(owner D) on every miss (4,000 of 4,000, alive or dead attacker). On a live hit: `RES | R DMG0 D3 E7 E3 ONHIT REACT DROP` (2,000; damage flag 0 in all). Reach lost: no events, no damage, `DROP` (8,000). **Dead attacker, hit, in reach: `R` only, no damage, no events, no drop (2,000; new)**. Block bits: the gate skips FUN_0057b7d0 and FUN_0057c1e0 (no event 11) |
| X3 | gate + resolver + real FUN_0057c6c0 (life_apply.Game hooks), game-vs-game absorb oracle on the life taken | 5,000 + 1,000 x 4 + 500 x 4 | record +0 = 2: `11D 5A 1D 7A 3D` in 5,000 of 5,000 (player attacker) and 1,000 of 1,000 (monster attacker); P == Q: all (4,501 + 899 discriminating). +4 with 0x20: `11D 7A 3D` (1,000). **Lethal hit: `11D 5A 1D 10D 9A 7A 3D` in 394 of 1,000; with 0x20 no 5/1 and no 10/9 (1,000; new)**. Record +0 = 0 (real FUN_0057b7d0): player attacker `3D 11D 5A 1D 7A 3D` (500 + 500); **monster attacker with FUN_0063ee90 = 0: no extra event 3 (500 of 500); = 1: extra event 3 (500 of 500; new)** |

Limit of X3 with FUN_0063ee90 = 1: P != Q in all 1,000 cases. FUN_0057b7d0 adds a part that depends on the input field before event 11, so pre-subtracting the absorb from the input is no longer the same as absorbing at event 11. This is a limit of the oracle, not evidence against the order (with FUN_0063ee90 = 0 and FUN_0057b7d0 running, P == Q holds in 1,000 of 1,000).

What is not run anywhere (mine or the authors'): the Energy Shield node FUN_005ca6c0 itself (all absorbs are a stand-in written into the record by the dispatcher hook); FUN_0057c060 at a value other than 100; FUN_0057c6c0 with flag != 0 on the event side (missile events 6 / 2); events 0, 12, 13; FUN_005d3ac0.

## Verdicts, first pass (`sk-state-events-trace.md`)

| # | Claim | Verdict | Why |
|---|---|---|---|
| F1 | FUN_005d3ac0 is an attacker hit bonus, not the dispatcher | NOT VERIFIED | not re-read or run by me or the second pass; stays V read |
| F2 | Dispatcher FUN_005c0c30: list at unit +0x90, id byte, node +0x14 with 5 args, once / +4 = 0 unlink | HOLDS WITH LIMITS | my reading agrees; every run hooks the dispatcher, so its internals are V read only |
| F3 | Owner = first stack arg: defender for 1, 2, 3, 11; attacker for 5, 6, 7, 9 | VERIFIED for 1, 3, 5, 7, 9, 11 (and 10 = defender), counts X2 / X3; 2 and 6 V read only | contradicts its own Q3 text (F6) |
| F4 | Call-site table (11 at 0x57c2fc, 5 / 1, 10 / 9, 7 / 3, 3 at 0x57bcad) | VERIFIED for those sites (X1..X3); 6 / 2, 12, 13, 0 not run | |
| F5 | Event 3 fires on a miss, after the damage, only in reach | VERIFIED | X2: 4,000 of 4,000 misses in reach; 0 out of reach |
| F6 | Events 7 and 3 "both owned by the defender argument" | WRONG | event 7 is owned by the attacker (X2 8,000, X3 all); the push order at 0x57d600 is record, defender, attacker |
| F7 | Reach mode 3 for a player, 1 otherwise | WRONG (reversed) | 1 for type 0, 3 for type 1 (X2, 8,000 each) |
| F8 | FUN_0057b7d0's event 3 skipped when record +4 has 0x20 | VERIFIED, incomplete | it is also skipped for a monster attacker unless FUN_0063ee90 is true (X3, 500 + 500). "0x20 is what Shiver's own 0x4021 record sets": NOT VERIFIED (nobody built that record) |
| F9 | Event 11 inside FUN_0057c1e0, after the percent scale, before resistances; "FUN_0057c6c0 calls FUN_0057c1e0" | HOLDS WITH LIMITS | before resistances: VERIFIED (X1, 20,000). After the percent scale: V read only (the percent was 100 in every run). FUN_0057c6c0 calls it only with flag != 0; the melee path runs it from the gate |
| F10 | D1: d2d does not model Shiver Armor | WRONG | `Fight::buff_events` lands the Shiver blow for every `melee_by` striker (V read, fight.cpp 1374..1393) |
| F11 | Answer: ES absorbs before resistances | HOLDS WITH LIMITS | the ORDER is verified (whatever node acts at event 11 sees raw fields and resistances run on what it leaves); the ES node's own math is not run |

## Verdicts, second pass (`stateevrun-trace.md`)

| # | Claim | Verdict | Why |
|---|---|---|---|
| S1 | Event 11 once per hit, owner defender, other attacker, before resistance | VERIFIED | X1 20,000; X3 chain |
| S2 | It fires at QUEUE time (gate), and the melee FUN_0057c6c0 call has flag 0 | VERIFIED | X2: `RES` before `|` in every hit cell without block bits; `DMG0` in 2,000 of 2,000 |
| S3 | FUN_0057c6c0 runs FUN_0057c1e0 only with a non-zero flag | VERIFIED (read and run) | own reading of 0x57c793; X2 |
| S4 | A queued swing dropped at resolution has already paid; mana loss is I | HOLDS WITH LIMITS | order VERIFIED (X2 `RES | R DROP`); mana loss I, correctly tagged. They missed the dead-attacker case: same payment, and the record is not even dropped |
| S5 | A miss never runs FUN_0057c1e0; block bits 0x8380 skip it (V read) | VERIFIED | X2 drove 0x80, 0x100, 0x200, 0x8000 (they did not). Note: with bit 0 still set, a block-bit record still gets `DMG0` at resolution in the emulator; whether the game ever queues such a record is U (I: block clears bit 0 earlier) |
| S6 | Q2 counts: 7 then 3 iff reach, damage iff hit and reach; 7 owned by attacker, 3 by defender; mode 1 / 3 | VERIFIED | X2, own seeds, 16,000; the extra grid (dead attacker, block bits) adds shapes their `match` metric did not cover |
| S7 | Hit order `RES, R, DMG, KILLCRED, E7, E3, ONHIT, REACT, DROP` | VERIFIED as a marker order | "KILLCRED" for FUN_0057d3d0 is a name, not shown (I) |
| S8 | Record +4 0x20 suppresses events 5 and 1 as well; damage still applied | VERIFIED | X3, 1,000; it also suppresses 10 / 9 (new) |
| S9 | Record +0 = 0: the gate's FUN_0057b7d0 fires an extra event 3 first | HOLDS WITH LIMITS | true for a player attacker (their only case); a monster attacker needs FUN_0063ee90 true |
| S10 | Life taken equals the BEFORE model, 0 exceptions (A 104,158, C 101,386) | VERIFIED in substance | their oracle is real game code against two arithmetic models (`resisted_fine` copies the MulDiv). My oracle needs no model (game against game) and agrees. The absorb in both harnesses is a stand-in, so "Energy Shield absorbs" is I for the node |
| S11 | D1: d2d absorbs after resistances (`Fight::absorb` on `foe.damage`) | HOLDS (V read) | I read fight.cpp 1344..1372 and 4775: absorb runs on the frame total. The resisted-blow sum (combat.hpp 354) was not re-read by me |
| S12 | D3: Shiver on a miss, no deviation | HOLDS WITH LIMITS | game side V run; d2d side V read (ai.cpp 2554 pushes `melee_by` hit or miss). Not an arithmetic re-derivation, but d2d was not run. Caveat: d2d skips a striker that is not alive (fight.cpp 1380); game.exe fires 7 / 3 for a dead attacker's miss (X2). Whether such a swing ever resolves in the game is U, so this is a possible deviation, tagged I |
| S13 | First-pass retractions (F6, F7, F10) | VERIFIED | as above |

## Deviations (verifier view)

| ID | Rule | Tag | Differential |
|---|---|---|---|
| D1 (= their D1) | ES order: game before resistances (order V run, game-vs-game: 20,000 in X1 and 10,000 chain cases in X3), d2d after (V read). The ES node's math (per-field min of percent and mana limit) is V read only (sk-armor-trace) | game order V run, d2d V read | none built. To build: un-hook FUN_005ca6c0 in X1 with a real ES node and mana in the stat hook, and compare with a d2d test mode applying one blow to a player with ES (life, mana) |
| D2 (= their D2) | ES paid at queue time; dropped swings and dead-attacker swings still paid | order V run, mana I | as their D2 |
| D4 (new) | dead attacker: game.exe fires 7 / 3 on a dead attacker's miss and silently keeps a dead attacker's hit; d2d drops strikers that are not alive | I | X2 is the game side; reachability in the real game is U |

**No PR candidate.** No d2d deviation is shown on the real d2d build. The second pass was right not to claim one.

## Bot findings

Their B1 stands as written (not re-read here; U). No bot code was changed.

## Ledger rows

Neither pass added a row to `docs/coverage/ledger.tsv`, so there is nothing to grant or revoke. Rows 5 (`skills.sor 50,58,60`) and 26 (`states 20,30,88`) stay READ-ONLY: the ES node's math and Shiver's damage were never run. Proposed note for row 5 (coordinator's call): "ES event 11 order before resistances V run (verifystateev2 X1); node math still read only".

## What each pass got right and wrong

- First pass: right on the dispatcher, the event-3-on-miss rule, the 0x20 gate and the ES order, and on its own owner table (Q2). Wrong in Q3 on the owner of event 7 (contradicting its own Q2) and on the reach mode. Wrong on d2d not modelling Shiver. It missed that the melee path runs event 11 at the gate, not inside FUN_0057c6c0.
- Second pass: every counted claim reproduced with my own harness and seeds; tags honest (mana-at-queue I, d2d V read, no PR). It missed the FUN_0063ee90 condition of the extra event 3, the dead-attacker exit, the kill events and their 0x20 gate. Its block bits were V read only. Its "Energy Shield absorbs the raw damage (V run)" wording overstates: the run proves where event 11 sits, and the absorb is the harness's own stand-in.

## Matrix-cell text (proposed, sorc-skill-matrix.md)

- 3.11 Energy Shield: "event 11 runs once per melee hit at queue time (gate FUN_0057dbf0), before the 12-row resistance pass [V run, verifystateev2 X1 / X2]; ES node math still V read".
- Shiver Armor: "event 3 on every in-reach swing, hit or miss, owner the defender [V run]; extra FUN_0057b7d0 event 3 only for a player or FUN_0063ee90 attacker and never with record +4 0x20 [V run]".

## Drill recipe (lab, for the U items; nothing run)

Per docs/forkdbg.md: (1) the ES order drill E1 of stateevrun-trace stands. (2) Dead attacker: let a monster start a swing at the player, `debug` kill it before the hit frame, and read the player's life and the ES mana: life unchanged with mana spent confirms the dead-attacker exit after the gate. (3) Blocked hit: give the player 75 percent block and ES, then read the mana over blocked swings: unchanged mana confirms that block bits skip event 11.

## Reproduce

`tools/emu/run-on-worker.sh vse_dis.py 57dbf0 57d4f0 5c0c30 57c1e0 57c6c0 57b7d0`; `tools/emu/run-on-worker.sh --ttl 1500 vse_run.sh 20000 90210`.

## Shared-file lines for the merger

See `openspec/changes/verifystateev2/merge-lines.txt`.
