# Verification of alignment2-trace.md: alignment (stat 0xac) and the undead-blow gate (card verifyalign, 2026-11-02)

> status: research, independent re-check. Offline: no game, no d2d slot, nothing pushed or posted. Own disassembly reading (`tools/emu/vpr_dis.py` over game.exe ranges), own harness `tools/emu/vfal_run.py` (own seeds; game.exe's real code under the emulator), compared with `docs/research/alignment2-trace.md` (the "author"). Verdicts: VERIFIED (counted run of game.exe's own code, comparison independent of the author's model), HOLDS WITH LIMITS, NOT VERIFIED, WRONG. d2d was not read; the bot's melee component rests on the game.exe rule only.

## Verdict (short)

The core rule is right: in `FUN_0057b7d0` the stat 120 / 121 / undead (`FUN_0057b660`) / stat 180 block runs when the defender is a monster and the attacker (EDX) is a player **or** has `FUN_006259b0 == 2`. Mercenaries, Shadow Warrior / Master and all other player-side summons carry alignment 2 at spawn (the Align 1 -> 2 map was run natively for all 734 MonStats rows), so they pass the gate. No claim was found WRONG. Open points are listed per claim; none changes the rule a melee PR component needs.

## What I did differently

- Reading first: I read 0x6259b0, 0x57b7d0..0x57b882, 0x5543b0, 0x57b660, 0x5b2e80..0x5b2ecc, 0x5c55c0, 0x573270, 0x535bc0 myself and wrote the model below before running.
- Model: `align(u) = 0 if u null or +0x5c null; 2 if type > 1; else stat 0xac of the state-0x69 list (0 if none)`. Gate: `defender type == 1 and (attacker type == 0 or align(attacker) == 2)`; the attacker is **EDX** (`mov edi, edx` at 0x57b7d7; `push edi; call 0x6259b0`), the defender is the first stack argument; ECX is only saved to `[ebp-0x10]`.
- The first pass stubbed the raw stat read `FUN_00625420` in parts A and B and called the gate with ECX = EDX; mine stubs nothing of the stat-list path (units get their state-0x69 list from the real setter `FUN_005543b0`, the unit's own list from `FUN_00626d40`), and passes a different type-0 **decoy** in ECX so that a swapped-register reading would show.
- Own seeds: 20261102 and 77 (random unit type / class / alignment / stat-list presence / defender type).

## Runs (V run; `tools/emu/vfal_run.py SEED`, output of seed 20261102 summarised)

| part | what | cases | differ |
|---|---|---|---|
| G | real setter `FUN_005543b0` then real getter `FUN_006259b0`; unit type 0..5 (setter only for 0 / 1, it faults for 2+ in `FUN_00639db0` without game state), 8 classes, alignment 1 / 2, 10 percent with the stat list removed; `getter(NULL)` | 400 (seed 77: 400) | 0 (0) |
| H | real `FUN_0057b7d0` to 0x57b882; random attacker (EDX) type 0..5 / alignment / list presence, defender type 0..3, ECX = type-0 decoy; counts entries into `FUN_0057b660` (hooked); the stat getters 0x625500 / 0x625480 / 0x627260 / 0x63e940 / 0x57b6a0 / 0x535d10 are hooked to 0 so the run reaches 0x57b882 | 600 (seed 77: 600), block entered 171 (162) | 0 (0) |
| S | the spawn mapping 0x5b2e94..0x5b2ecc run natively for every record of game.exe's loaded MonStats table (record +0x1a8 * i, byte +0x4c), the setter hooked to record DL, flag `+0xc4 & 0x20000` read back; compared with the Align column of MonStats.txt | 734 | 0; DL histogram {0: 619, 2: 113, 1: 2} (same on both seeds) |
| X | raw `E8 rel32` scan of game.exe for calls to `FUN_005543b0` (independent of the author's list) with the 3 instructions before each | 21 sites | count equals the author's 21 |

Harness fault found and fixed on the way: part S replaces the setter by a recorder, and my first ordering ran it before H, so H's setter was a no-op and 20 align-2 cases "differed"; the order is now G, H, S (written here because it is the kind of error that hides a real result).

## Claim by claim

| Author's claim | Verdict | Basis |
|---|---|---|
| 1. Gate is defender type 1 and (attacker type 0 or `FUN_006259b0(attacker) == 2`), 0x57b805..0x57b828 | **VERIFIED** | read (0x57b805..0x57b82a: `je 0x57b824` skips the call for type 0, `cmp eax,2`, `cmp [ebp-8],1`); H: 600 + 600 runs, 0 differ, attacker taken from EDX with a decoy in ECX |
| 2. Getter: null -> 0, null stat list -> 0, type > 1 -> 2, else stat 0xac of state 0x69, 0 without such a list | **VERIFIED** | read 0x6259b0..0x6259ee; G 400 + 400 runs with the real stat-list walk and real raw read |
| 2b. "the verifier's note misses the null-list exit" | HOLDS (reading) | 0x6259c0/6259c4: the null-list exit precedes the type test |
| 3. One setter `FUN_005543b0`, 21 call sites | **VERIFIED (count)**; site table HOLDS WITH LIMITS | X finds exactly 21 direct calls. DL at the sites (read): 2 at 0x53495a, 0x5732b3, 0x5c08fb, 0x5c564c, 0x5d04ef, 0x5d5feb, 0x5d74f9; 0 at 0x5432c5, 0x5c3b0a, 0x5c3dba, 0x5d01b9, 0x5d7320, 0x5dd6f5; 1 at 0x5c3edb; map at 0x5b2eab / 0x5b2eb4 / 0x5b2ebb (0 / 1 / 2); variable at 0x5428f4, 0x5dd76f; copy of the caster's at 0x5cab05. The author's text does not name 0x5c08fb and 0x5d74f9 (two more value-2 setters): not needed for the rule, owners unidentified (U); 0x5428f4 and 0x5dd76f (variable value) were not followed by either of us |
| Setter early exit | new note | 0x5543ca..0x554416: value >= 3 aborts; for value 0 and a class other than 0x15f..0x161 the setter returns before building a list when `FUN_00451f30(unit, 0x80000000)` is nonzero. Harmless to the rule (no list reads as 0) and does not touch value 2 |
| 4. Spawn map `+0x4c`: 1 -> 2 (+ `|= 0x20000`), 2 -> 1, else 0; 113 rows 2, 2 rows 1, 619 rows 0 | **VERIFIED** | read 0x5b2e94..0x5b2ec2; S ran the real instructions for all 734 records: same histogram, flag set exactly when value 2, 0 differ from the Align column |
| Q2 five hireling classes give 2 (rows 271, 338, 359, 560, 561) | **VERIFIED for the spawn mapping**; HOLDS WITH LIMITS for "a live merc carries 2" | S per-class output (all five: 2). `FUN_005b2a00` as a whole and the merc creation path were not run by either of us; `FUN_00573270` read (sets 2 and `+0xc4 |= 0x4020200` when the owner `[edx]` is type 0 and a flag argument is set) but that the merc goes through it is I (pets-merc-trace.md, not re-read) |
| Q3 Shadow Warrior 417 / Master 418, necroskeleton 363, necromage 364, golems -> 2 | **VERIFIED (spawn map, S)** | Shadow rows 417 / 418, 363, 289 printed 2; Revive's own class-independent `mov dl,2` at 0x5c5648..0x5c564c read, `FUN_005c55c0` is a function whose entry I read from 0x5c55c0; the revived unit's later stats and the do-58 dispatch were not run: NOT VERIFIED beyond the read |
| Q4 Valkyrie 357, Hydra 351, wolf 430, Dopplezon 356, Blade Sentinel 413 -> 2; Lightning Spire 371 -> 0; cow / camel -> 1 | **VERIFIED (spawn map, S)** | per-class S output: Valkyrie, hydra1, wolf, dopplezon, bladecreeper 2; lightningspire 0; cow, camel 1. The other druid / sentry classes the author lists were not each printed by me (the 113-row total is VERIFIED); `FUN_005d5e10` do 44 / 45 and conversion `FUN_005d0350` / undo `FUN_005d01a0`: only the setter sites 0x5d5feb (2), 0x5d04ef (2), 0x5d01b9 (0) are VERIFIED as read; the enclosing function identities are the author's (S) |
| Q5 player type 0: block runs without reading alignment | **VERIFIED** | H: type 0 attackers with alignment 0 / 1 / 2 and with the list removed all enter the block against monsters |
| Q6 "bonus size" depends on the unit's stat 122 and a Blunt weapon (`FUN_0057b660`); a monster without inventory has no weapon | HOLDS WITH LIMITS | `FUN_0057b660` read: target undead (`FUN_0063e990`) -> `esi = 50 if weapon (FUN_00535bc0(attacker)) is ItemTypes 57, else 0; esi += attacker stat 0x7a (122)`; add to record +0xc when over 0. `FUN_00535bc0` read: attacker null -> 0, `[unit+0x60]` null (no inventory) -> 0, else walks the inventory; the merc's hand-weapon result and the monster path were not run (I). Numeric ED sizes are not a V run here (the author's 60 / 60 for `FUN_0057b660` is S from sk-melee-trace.md; not re-run) |
| Gate also admits unit types 2..5 with a stat list | VERIFIED as a game.exe property (H: types 2..5 with a list enter the block, without a list not); whether such a unit is ever an attacker here: U (agree with the author) |

## Deviations d2d vs game.exe

The author's D1 / D2 / D3 stand as stated; d2d was not read here either (tag S, via the verifier text), so I add no d2d claim. A differential for the merc / Shadow rule would need d2d's blow for those attackers, which is not in this sandbox. No PR candidate from this card (no before / after run).

## Bot findings

- B1 (author) holds: the bot models the player's blow only (`internal/combat/melee.go` Blunt from the wielded weapon, `internal/resource/melee.go` `BluntUndeadPct = 50`). Nothing to fix; a later merc damage model must use "alignment 2 (player side)", not "player only".
- B2 (author) holds as a doc note.

## Not covered by either pass (what a live drill must still settle)

1. Stat 0xac read on a live merc, Shadow, summon and revived monster (author's drill recipe items 1 and 3 apply unchanged: forkdbg stat-read verb; expect 2).
2. The merc's own undead ED with a Blunt weapon or a stat-122 item against a skeleton (recipe item 2): `FUN_00535bc0` for a merc was only read.
3. Owners of the two unattributed value-2 setter sites (0x5c08fb, 0x5d74f9) and of the variable sites (0x5428f4, 0x5dd76f).

## Matrix cells

- Author's proposed cells (sk-melee-trace.md "stat 122 / Blunt vs undead" row; merc rows of pets-merc-trace.md) can be taken over; add the tag "gate V run (verify-alignment2.md, 1200 runs, 2 seeds); spawn alignment V run for all 734 rows".

## Glossary

Same missing term as the author's: "alignment (stat 0xac)".
