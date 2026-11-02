# Verification of paldur-trace.md (durability loss per hit and per block)

> status: current (2026-11-02, agent verifypaldur, Sonnet, independent). Offline: emulator only, no game. Tags as in AGENTS.md.

## Method

1. Read `FUN_0057d3d0` (0x57d3d0..0x57d4e3) and `FUN_00559e30` (0x559e30..) in the disassembly first (`vq_dis.py`), before opening the author's scripts.
2. Wrote an own model (`tools/emu/vp_wear.py`: the LCG, the ring walk, the 10/4 percent roll, the durability ladder) and own cases (seeds 101 with 4,000 and 4242 with 30,000 weapon/javelin cases, and the same count of armor cases; random durability, max, quantity, gate, game +0x70 in 0/1/5, random slot presence).
3. Ran game.exe's real `FUN_0057d3d0` (which calls the real `FUN_00559e30` and `FUN_0045c3e0`) under the emulator and compared its end state and the player's seed with the model, byte for byte. Runner class reused from `pal_wear.py` (so the hook set is the author's, see limits); d2d's binary was not used at all.

## Result: oracle executes game.exe code

Yes. In `pal_wear.py` the code run natively is the selection loop, the two seed draws, the roll and the ladder; only the item side is faked (type tests, gate, stat get/set, hand, body slot lookup). So "0 differences vs d2d" is not an arithmetic re-derivation of d2d's formula: both sides are compared with game.exe's instructions. Limit: the faked item side means the claims about item typing are V read only.

## Claim table

| Claim (paldur-trace) | Verdict | Evidence |
|---|---|---|
| Chance armor 10, weapon 4, throwable 10 | VERIFIED | immediates in the disassembly (0x559e72 `4`, 0x559e91 `0xa`), and 0 mismatches in 34,000 weapon cases with the model using them |
| Throwable stack never wears when game +0x70 is 0 (D2) | VERIFIED as game.exe behaviour (0x559e85..0x559e8b, plus 0 mismatches including f70 = 0 cases); live value of +0x70 stays U | |
| Roll = new low dword % 100, `>= chance` returns; LCG lo*0x6ac690c5+hi | VERIFIED | 0 mismatches in 64,000 cases incl. final seed |
| Armor slot: start = rand(7), pick = rand(W), ring walk, weights 3/5/4/4/2/2/2 | VERIFIED (counts) | 34,000 armor cases, 0 mismatches of item state and seed; loss counts per slot scale with weights |
| Right-hand weapon has weight 0 in the walk | VERIFIED for the model of the type test hook; real typing V read only | 0x57d45c tests type 0x32 |
| Ladder: armor breaks (flag 0x100, dur 0), weapon stays at 0, stack loses one and returns to max, last goes out | VERIFIED for the ladder (0x559f15..), break and ammo calls hooked | model match |
| One draw spent when picked item cannot wear (gate 0) | VERIFIED | gate=0 cases in the 34,000 |
| Blocked/dodged blow wears nothing (Q2) | HOLDS WITH LIMITS | not re-run; relies on the sub-step hooks of `FUN_0057ec10` and the unread `FUN_0057dd60`; V read of the 0x57d591 gate not repeated by me |
| 35 callers, single path to the wear (Q1) | NOT VERIFIED (not re-run) | direct-edge scan only |
| D1 (d2d wears on missiles, game.exe does not) | NOT VERIFIED, stays I (as the author says) | indirect calls not followed |
| D3 (broken armor costs a draw in game.exe) | HOLDS | consistent with the disassembly: the selection never tests the gate or the broken flag; only `FUN_00559e30` does, after the pick |
| D4..D6 | NOT VERIFIED here | table checks, not re-run |
| "No PR candidate" | AGREE | D2 depends on U input; D1 is I |

## Error found in the prose

Q3 of paldur-trace.md says an armor piece "counts when ... it passes the durability gate". The disassembly says otherwise: the selection loop (0x57d45c..0x57d46f) counts a slot when the item exists and `FUN_00629bb0(item, 0x32)` holds, nothing more. The gate is tested later in `FUN_00559e30` (0x559e63). The author's own next paragraph and the run (gate-0 piece spends the draws) agree with the code; only that one sentence is wrong. Effect on numbers: none.

## Limits

- Hooked item side as in the author's harness; no run of the missile path, `FUN_0057d4f0`, or the block function by me.
- The statistics (10 %, 4 %) are the seed-exact model equality, not a frequency measurement.
- Dual wielding, merc, repair untouched.

## Matrix-cell proposal

None to change; the paldur-trace cells stand, with the Q3 sentence corrected as above.
