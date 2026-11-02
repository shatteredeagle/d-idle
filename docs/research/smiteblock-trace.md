# Smite and the block check: game.exe 1.14d vs d2d vs the bot (SMITEBLOCK, one question from PALSKILL)

> status: card smiteblock, 2026-11-02, offline (no game, no d2d slot, no Go change). Settles a disagreement between two traces of the same item: `docs/research/palskill-trace.md` (ours) D6 says "a block stops Smite"; `docs/research/pal-skill-trace.md` (the community trace, repo PR 23) D8 says "Smite is unblockable, game.exe forces the hit after the check". Authored by an AI coding assistant.

Tags: **V read** read in the disassembly (address and instructions named); **V run** run natively in game.exe's own code under the emulator, with the hooks listed in section 3, and compared with d2d's real `player_blow` built on the worker host; **D** from the patched 1.14d tables (MonStats read with the mpq reader); **S** a source or earlier trace of ours; **I** inference; **U** unverified.

## Summary

- **The community trace is right, ours was wrong (V run).** A block does not stop Smite in game.exe 1.14d. Over **96,000 cases** (three runs: seeds 7 and 1, 40,000 each, and a third of 16,000 with the knockback bit added) the Smite damage record reached the damage consumer in **96,000 of 96,000**, including the **23,347** cases whose check result carried the block bit; the same chain without Smite's one extra instruction applied the damage in 7,467 cases and in **0 of the cases that were blocked**. All three runs: 0 differ from the reading in section 2.
- **Why:** FUN_005ce9f0 calls the to-hit / block check FUN_0057ec10 at 0x5ceba8 and then runs `or ax, 1` at 0x5cebb0. The block clears the hit bit and sets a block bit (0x10) that **no gate on the damage path tests**; the OR puts the hit bit back; the damage consumer needs only that bit. Our D6 read "forces the hit" correctly and then wrote "both let a block stop it" from the d2d source without running the join. Same lesson as its dead end: read to the join of the result.
- **d2d deviates (V run, D1):** `player_blow` skips the hit roll for Smite but still rolls the monster's block (`components/rules/combat.hpp:499` of the scratch pair, 303 of the upstream clone) and `strike.cpp:412` ends the blow on `blocked`: no damage, no stun, no knockback. d2d built from upstream main on 20,000 Smite blows per block chance: **landed 20,000 / 15,075 / 10,009 / 0 at block 0 / 25 / 50 / 100 %** against game.exe's 100 % at every block chance. One-line candidate F measured (0 blocked of 20,000 at every chance, plain attack unchanged).
- **PR candidate: none by the rule** (the harness is new, not the existing one; section 6 keeps the measurement and the draft text for an independent check).
- **Bot:** nothing in the bot depends on it (B1 to B3, described, not changed).

## 1. Behaviour (game.exe 1.14d)

Smite's do function FUN_005ce9f0 (srvdofunc 150; the community trace cites FUN_005cea30, which is not a function start: the prologue is at 0x5ce9f0, V read). Player path (type 0 attacker, V read, `docs/research/palskill-trace.md` section 2 for the damage side):

| step | address | what | tag |
|---|---|---|---|
| damage | 0x5ceb31 | FUN_0057b420 with the shield's min / max, ED = calc1; value stored in the record at +8 **before** any check | V read (damage numbers: palskill-trace, V run) |
| check | 0x5ceba8 | `push 0; push 0; push target; call FUN_0057ec10` (bonus 0, range 0) | V read |
| join | 0x5cebb0 | **`or ax, 1`**: the hit bit is forced on, every other bit of the result is kept | V read |
| result into the record | 0x5cebe9..0x5cec02 | `test al,1` (always true now): `or ax,[skill+0x12e]` into the word at record+4 and `or [rec], [skill+0x130]` (ResultFlags) | V read |
| gate | 0x5cec29 | FUN_0057dbf0(target, record, 0x80) | V read |
| consume | 0x5cec35 | FUN_0057d4f0 | V read |

FUN_0057ec10 (to-hit, block, dodge; V read, 0x57ec10..0x57ed44), after the reach test FUN_00622c40: roll FUN_0057d9b0; a pass sets bit 0 and calls FUN_0057dfb0 (0x57ece8) whose result `al` is mapped: `al & 2 -> 0x100` (0x57ecf1), `al & 4 -> 0x80` (0x57ecfb), `al & 0x10 -> 0x8000` (0x57ed05), **`al & 1 -> 0x10` (0x57ed0f, the block)**; any non-zero result clears bit 0 (`and ebx, 0xfffe` at 0x57ed16), and a 0x10 clears it again (0x57ed21). FUN_0057dfb0 returns 1 for a block (0x57e082): the chance FUN_00622720, one roll on the defender's seed (`% 100`, lcg 0x6ac690c5), a third for a player defender in a moving mode (0x57e00b..0x57e01f, not reached for a monster). So **a block is bit 0x10; dodge / avoid / evade are 0x100, 0x80, 0x8000**.

FUN_0057dbf0 (V read, 0x57dbf0..0x57dc9a): `test eax, 0x8380` (0x57dc1d) and `test al, 1` (0x57dc24) decide only whether the damage is **prepared** (FUN_0057b7d0, skipped for Smite by record flags bit 2, and FUN_0057c1e0, the defender's resistance step); the queue call FUN_0057ca80 at 0x57dc8f is unconditional. FUN_0057d4f0 (V read) then finds the queued record and tests **only bit 0** of the record word (0x57d591): set -> FUN_0057c6c0 (the damage, the stun at record+0x44 and the knockback pass through it); clear -> a no-damage branch (0x57d5fd). FUN_0057c6c0 reads the word at record+4 at 0x57c7a9 (0x20), 0x57c7ad (bit 1: only the on-hit events), 0x57c835 and sets / clears bit 2 (0x57c8eb, 0x57c93d); it never tests 0x10, 0x80, 0x100 or 0x8000 (V read: every access to that word is listed). The block bit is consumed afterwards by FUN_0057cee0 at 0x57d048 (`test al,0x10` -> FUN_005734c0 mode 0x13, the block animation), after the knockback test at 0x57d01d, so the defender plays a block or a knockback animation and **still has taken the damage** (V read; the animation itself not run).

**The exact rule (V run, section 4): Smite always lands on a monster defender, block or not. Block only changes the defender's animation. A plain attack (no `or ax, 1`) with a block bit is not applied.** Dodge bits (0x8380) with Smite: see D2 / O1.

## 2. The reading the run is compared with

For a monster defender, attacker type 0, hit roll H (FUN_0057d9b0 alone, natively), block chance b, dodge result d (0, 2, 4, 16):

| H | block roll < b | d | result word ax | Smite's word (ax or 1) | Smite lands | plain lands |
|---|---|---|---|---|---|---|
| fail | not rolled | any | 0 | 1 | yes | no |
| pass | yes (b > 0) | any | 0x10 | 0x11 | **yes** | no |
| pass | no | 0 | 5 (hit, no state 0x36: `or 4`, 0x57ed38) | 5 | yes | yes |
| pass | no | 2 / 4 / 16 | 0x100 / 0x80 / 0x8000 | + 1 | yes (bit 0 only) | no |

"Lands" = FUN_0057c6c0 is called (the consumer of the record). The last row is what the run counts; what FUN_0057c6c0 does with a record that skipped FUN_0057c1e0 is V read only (O1).

## 3. The run (V run, `tools/emu/sb_block.py`, `sb_run.sh`)

`tools/emu/run-on-worker.sh sb_run.sh sb_block.py CASES SEED [EXTRA_BITS]`. Native game.exe code: FUN_0057d9b0 (the roll, in a first emulator because a native call of it inside FUN_0057ec10 clobbers EBX in this harness through the hooked helpers; the result is replayed into the second emulator as a hook), then in the second emulator **FUN_0057ec10, FUN_0057dfb0, FUN_0057dbf0, FUN_0057ca80, FUN_0057d4f0** natively. `or ax, 1` is done in Python (one instruction). Hooked, not the claim: the stat getters, FUN_00622560 (attack rating), **FUN_00622720 (the block chance: the cell value; its own cap and stat reads are not exercised)**, FUN_00622d00 (not moving), **FUN_0057dd60 (the dodge result: the cell value; the dodge chance is not exercised)**, FUN_00554200 / FUN_005a4ee0 / FUN_00622c40 (foe, monster, reach: yes), FUN_00639df0, the pool allocator FUN_0040b430, and the consumers **FUN_0057c6c0 / FUN_0057cee0 / FUN_0057c1e0, hooked to count calls** (what they do inside is V read). The attacker is a player (type 0, mode 6), the defender a monster (type 1, mode 1); the record is built by the script (flags +0 = 2, word +4 = the result), not by FUN_005ce9f0 itself (it needs the game's unit tables).

Cells: block chance 0 / 25 / 50 / 100 % x dodge result 0 / 2 / 4 / 16 x N cases, each with its own seeds, attack rating (50..2499) and defense (0..1499), so the hit roll passes in 55 % of cases. Per cell one Smite chain and one plain chain.

| run | cases | hit roll passed | blocked (0x10) | **Smite applied** | **Smite applied and blocked** | plain applied | plain applied and blocked | differ from reading |
|---|---|---|---|---|---|---|---|---|
| seed 7, 2,500 a cell | 40,000 | 22,173 | 9,655 | 40,000 | **9,655** | 3,136 | 0 | 0 |
| seed 1, 2,500 a cell | 40,000 | 22,211 | 9,759 | 40,000 | **9,759** | 3,069 | 0 | 0 |
| seed 3, 1,000 a cell, +knockback bit 8 | 16,000 | 8,978 | 3,933 | 16,000 | **3,933** | 1,262 | 0 | 0 |

Per block chance (seed 7, dodge result 0, 2,500 cases each): Smite applied 2,500 at 0, 25, 50 and 100 %; blocked (bit 0x10) 0 / 377 / 688 / 1,370; plain applied 1,408 / 1,021 / 707 / 0. The resistance step FUN_0057c1e0 ran for **every** blocked Smite case (a block does not skip it) and for 30,618 of 40,000 cases overall (the rest are the dodge cells, D2).

d2d side (`tools/emu/sb_build.sh 20000 7`: d2d's real `player_blow` from the scratch pair of upstream main, Fighter with shield damage 3 to 5 and attack rating 5000, target armor 0, `swing.smite`; no table read):

| block chance | Smite blows | landed (`blow.hit`) | blocked | plain landed | plain blocked |
|---|---|---|---|---|---|
| 0 % | 20,000 | 20,000 | 0 | 18,982 | 0 |
| 25 % | 20,000 | 15,075 | 4,925 | 14,348 | 4,659 |
| 50 % | 20,000 | 10,009 | 9,991 | 9,547 | 9,521 |
| 100 % | 20,000 | **0** | 20,000 | 0 | 19,011 |

game.exe: 20,000 of 20,000 land at every block chance. Mismatches before: 34,916 of the 60,000 blows with block > 0; with F (`sb_after.sh`): **0** of 60,000, plain attack columns unchanged.

## Q and deviations

**Q1. Does a block stop Smite in game.exe?** No (V run, 23,347 blocked cases, all landed; V read for the gates). **Q2. Which trace is right?** PR 23's D8 on the effect; its mechanism text ("block (bit 4)", "defense and attack rating bypassed") is loose: the block bit is 0x10 of the record word, the roll is made and ignored (palskill-trace D6 on the roll). **Q3. What does d2d do?** It lets the block end the blow (V run, section 3).

| id | rule in game.exe | rule in d2d | tag | differential |
|---|---|---|---|---|
| **D1** | block never stops Smite (0x57ecbb..0x57ed1c, 0x5cebb0, 0x57dc1d, 0x57d591) | `combat.hpp:499` rolls the monster's block for Smite; `strike.cpp:412` returns on `blocked` (no damage, stun or knockback) | **V run** (96,000 game cases, 0 differ from the reading; d2d 80,000 blow pairs) | yes, built here: `sb_block.py` + `sb_d2d_test.cpp` / `sb_build.sh` / `sb_after.sh`; oracle = the Smite chain's application count |
| D2 | with dodge bits (0x100, 0x80, 0x8000) the gate skips FUN_0057c1e0 (resistance step) but the record is still applied (called in 9,382, 9,383 and 3,783 of the dodge-cell cases) | monster defenders have no dodge in d2d (I) | I (counter run, interior V read) | could be built: run FUN_0057c6c0 natively with stat 0x22 / 0x23 and 12-entry resistance table hooked; oracle = hp lost with and without the 0x8380 bits. Reach: a monster with a dodge / avoid stat (not looked up) |
| D3 | the block roll is one draw on the defender's seed, taken only when the hit roll passed | with F no block draw for Smite | I | stream only (palskill D6); no gameplay differential |

Reach of D1 (D, `mpq-table.sh monstats`, ToBlock column; that d2d's `Target.block` is this column is I, drill R1): **525 of 703 enabled monsters have ToBlock > 0**, 241 have >= 25 and 86 have >= 50; early Act 1: Skeleton 10, Fallen 9, Fallen Shaman 4, Brute 4, Zombie 3, Rogue scouts up to 75 (rogue1 / rogue3). In the current d2d a Paladin using Smite loses that percentage of its blows, so d2d understates Smite against blockers; in the game it lands every time.

## 4. Bot findings (described, not changed)

- **B1.** Nothing in the bot models the defender's block or the hit chance of a skill: `internal/combat/rules.go:237` lists Smite as a plain physical skill and the ladders (`internal/combat/classdata/profiles.json:106,131,137`) rank it only by level bands. Smite's guarantee (lands on every blocker and every defense value, V run) is a reason to rank it above the plain Attack against blocking monsters (I, not a measured gain: Smite's damage is the shield's, palskill B2). No change proposed here.
- **B2.** Documents that carry the old reading: `docs/research/palskill-trace.md` D6 ("both let a block stop it") and its section 2 table (not checked: whether `paladin-act1-recon.md` repeats it); corrections are proposed in section 7.
- **B3.** The bot's block knowledge for the player side (shield, Dex) is not read anywhere in `internal/combat/*.go` (a grep for "block" finds only unrelated uses: affix.go:150-157, aura.go:165): the player-side block, which this card did not touch, stays unmodelled.

## 5. PR candidates

**None by the rule.** The differential is new (this card's `sb_*`), not the existing harness, and the independent check (d2-reviewer, own harness) has not run. The measurement for a later candidate is in section 3: before 34,916 of 60,000 blows differ, after F 0. F is one line, a copy applied by `sb_after.sh`:

```
-    if (target.block > 0 && rng(100) < target.block) { blow.blocked = true; return blow; }
+    if (!swing.smite && target.block > 0 && rng(100) < target.block) { blow.blocked = true; return blow; }
```

Draft text (stays here): "Smite's blow is not stopped by the target's block: FUN_005ce9f0 ORs the hit bit back in after FUN_0057ec10 (0x5cebb0), the block bit 0x10 is tested by nothing on the damage path (FUN_0057dbf0 tests 0x8380 and bit 0, FUN_0057d4f0 bit 0), so the blow, its stun and its knockback apply. d2d lets a block end it." No local branch was made (no fork checkout is writable from this card).

## 6. Drill recipe (lab d2d verbs, `docs/forkdbg.md`, `docs/d2d-fork.md`, `docs/forkcheats.md`; a drill shows what d2d does, the game.exe side stays V run)

- **R1 (D1, d2d column, the U on `Target.block`).** `debug equip` a weapon and a shield, `debug points 97 1`, `debug spawn` a monster with a high block (rogue3: ToBlock 75, D) and `debug mon set <id> life 1000 maxlife 1000`; `debug mana full`; `debug cast 97 unit <id>` 40 times with `step` between, reading `monsters` life each time. d2d today predicts about 25 % of casts that lower the life by the shield's damage and none after a block (a block animation, no stun, no knockback). The game.exe rule is every cast lowers it (this trace). Control: the same with the plain Attack (`cmd skill` / `debug cast` of Attack): both columns lose the block share. Repeat with a Fallen (ToBlock 9) as the low-block case.
- **R2 (D2).** Not drillable (no dodge stat on a lab monster known); needs the emulator build in D2.
- **E1 (emulator).** Run FUN_005ce9f0 end to end instead of the chain after it: needs the unit tables at 0x744304 and the player's skill list; then the block roll is the defender's own draw and the damage the real FUN_0057b420 call. Not built.

## 7. Matrix-cell updates (proposed text; no Paladin matrix exists, `docs/research/sorc-skill-matrix.md` is untouched)

- `docs/research/palskill-trace.md` D6: replace "Gameplay equal (both always hit; both let a block stop it, S combat.hpp swing.smite)" by "game.exe: Smite is not stopped by a block (smiteblock-trace.md, V run); d2d lets a block end it (smiteblock D1)"; the D6 verdict "not a deviation" stays for the roll alone.
- `docs/research/paladin-act1-recon.md` section 6 row 5 (b): add "Smite vs block: game.exe lands every time, d2d loses the block share (smiteblock-trace.md D1, V run, 96,000 cases; no PR candidate, harness new)".
- `docs/coverage/ledger.tsv` line `skills.pal 96,97,106`: the note gains the block sentence (evidence and cases columns unchanged: skill 97 is already claimed by that line and the generator refuses a second claim).

## 8. Limits and dead ends

- Not run: FUN_005ce9f0 end to end; FUN_00622720 (the block chance formula; d2d's equals game.exe's on the monster side, S `monster-attack-trace.md`) and FUN_0057dd60 (dodge chance); the interiors of FUN_0057c6c0 / FUN_0057cee0 / FUN_0057c1e0 (counted, read); a player defender (PvP: FUN_0057dfb0 has a type-0 branch, a third in moving modes, not run); the Smite stun length and knockback on the blocked blow (they travel in the same record through FUN_0057c6c0: V read, not counted); the real stat getters of the defender.
- Dead end: a native call of FUN_0057d9b0 inside FUN_0057ec10 returned a result word full of garbage (EBX clobbered by the hooked helpers, `sb_dbg.py` shows the trace); the first harness counted FUN_0057ca80 calls as "applied" and showed the plain attack landing on a 100 % block: FUN_0057ca80 is the unconditional queue, the apply decision is in FUN_0057d4f0 (bit 0). Both fixed before the numbers above.
- The 96,000 are generator draws, not distinct gear; counts are per seed.

## 9. Shared-file lines for the merger

In `openspec/changes/smiteblock/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG). `docs/decomp-queue.md` / `docs/research/README.md`: one row each for this doc (a block does not stop Smite in game.exe, V run; d2d D1; no PR candidate).
