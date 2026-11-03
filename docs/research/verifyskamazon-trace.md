# Independent verification: SKAMAZON (first-pass Amazon trace)

> status: card verifyskamazon, 2026-11-02, Sonnet verifier, offline (emulator on the worker host, no game, nothing pushed or posted). Subject: `docs/research/skamazon-trace.md` (branch agent/skamazon, commit 412e861f, not yet on main when this was written) and its three ledger lines. Method: game.exe's functions read first (own disassembly with upstream's `disasm.py`), my own model, then game.exe's real code under the emulator on my own seeds and my own harness; the author's scripts were not run.

Tags: **V run** (own harness, counts and seeds given), **V read** (own disassembly, addresses given; d2d source read at upstream main), **D** (patched 1.14d tables, read through `mpq-table.sh`), **S/I/U** as in the subject doc. Verdicts per claim: VERIFIED (counts), HOLDS WITH LIMITS, NOT VERIFIED, WRONG.

## Verdicts in short

| claim of skamazon-trace.md | verdict | main caveat |
|---|---|---|
| Calc cells of the 30 Amazon rows equal d2d, 0 of 27,800 differ | **VERIFIED** (own 0 of 52,300) | both sides really are real code (game.exe's calc VM, d2d's `eval_calc`), not arithmetic; but the cells depend only on the skill's own level (Valkyrie also on 5 neighbours), so 27,800 / 52,300 cases are about 4,300 distinct inputs; and 2 of the 22 "0-diff" rows (Magic Arrow 6, Power Strike 14) have no calc cell at all |
| Value tables of Q2 and Q5 | **VERIFIED**, one **WRONG** cell | Penetrate at level 40 is 425, not 415 (35 + 10 per level; game.exe's own cell) |
| D1 missile roll top exclusive in game.exe, inclusive in d2d | **VERIFIED for the primitive** (own 20,000 cases) | the 118,000 life-level counts were not re-run (that is the author's harness); the roll step is verified, not the whole hit |
| D2 Pierce: game.exe FUN_0059f940 count 0..4, d2d never pierces | **function VERIFIED (own 99,092 cases); d2d side S; the inference "same count for every arrow" is WRONG-LEANING** | the shooter's stat 328 is incremented by one after a missile is sent (V read 0x5980a1..0x5980b0), so the count is a per-shot stream, not a constant; who calls it stays U |
| D3, D4, D5, Q3, Q4, Q7, Q8 (code of Strafe, Multiple Shot, Jab, Fend, Decoy, Valkyrie) | **NOT VERIFIED** | the author marks them S/U; I did not read them either |
| Ledger line `skills.ama ... TESTED-0-DIFF 27800` | **earned for the calc aspect with two rows vacuous** | rows 6 and 14 have no calc cells; row 33 (Pierce) sits in a "0-diff" line although D2 is a real gap in its behaviour |
| Ledger lines `skills.ama` and `missiles` `TESTED-DEVIATION 118000` | **earned** (roll primitive) | the case counts are the author's harness; my own run proves the rule (20,000) |

## 1. How it was run (own harness, V run)

| piece | what |
|---|---|
| `tools/emu/vam_calc.py` + `vam_test.cpp --calc`, `vam_run.sh N SEED` | game.exe's calc entry FUN_00646ca0 (-> VM FUN_006c0bc0) on every calc pointer of Skills 6..35 (calc1..4, petmax, sumsk1..5 gated by the sumskill ids, delay, aura range / length / perdelay, aurastat1..6, passive1..5), against the real inline `d2d::rules::eval_calc` on the loaded tables. Hooks differ from the author's on purpose: base level != level with items, EVERY stat non-zero (seeded), the three stat getters told apart (`.base` +1000, `.mod` +2000, stat 19 separate), neighbours drawn from all 357 rows, levels 0, 100, 127, 128, 200, 255 included, and every stat or level read is logged |
| `tools/emu/vam_pierce.py` | game.exe FUN_0059f940 called on a fake shooter and missile with three separate stat dictionaries, compared with MY python model written from the disassembly |
| `tools/emu/vam_roll.py` + `vam_roll.sh` + `vam_test.cpp --roll` | game.exe FUN_005a8910 (the element min/max roll of the missile hit) against d2d's own `Rng::range` (rules.hpp:169), each with the same seed state |
| `tools/emu/vam_mrow.py` | the compiled Missiles rows that carry the pierce bit, against the Pierce column of Missiles.txt |
| `tools/emu/vam_vacuous.py`, `vam_dis.sh` | which rows have any non-zero calc cell; upstream's disassembler wrapper |

Runs: `vam_run.sh 6000 424242` = 9,150 cases, DIFF 0; `vam_run.sh 40000 7` = 43,150 cases, DIFF 0 (52,300 together, with sumsk included; a first pass without sumsk gave 0 of 9,150 and 0 of 43,150 too). Read log: **empty for all 29 rows except Valkyrie (32)**: no calc cell of any Amazon skill calls the stat getter or reads a level other than its own. Valkyrie reads the levels of skills 9, 13, 18, 29 (Critical Strike, Dodge, Avoid, Evade: the sumskill calcs), 28 (Dopplezon) and 23 (Penetrate: **U**, the Valkyrie row's text names no Penetrate cell; the read could be a skill-id lookup of the harness's neighbour list, not traced). `vam_pierce.py 11 3000` = 42,546 cases and `vam_pierce.py 23 4000` = 56,546 cases, MISMATCH 0 each. `vam_roll.sh 20000 5` = 20,000 cases.

## 2. Claims, one by one

**A1. Calc cells equal d2d (0 of 27,800).** VERIFIED: 0 of 52,300 on my harness, both sides real code. What the number means: the author's own stat hook returned 0 for everything but clvl and the masteries, and so does d2d's env there; with the read log I can say why that was harmless: **no Amazon calc cell reads a stat** (log empty), so the clvl, mastery and neighbour grid dimensions of the author's run were inert for 29 rows. Distinct inputs: (skill, level) pairs, about 30 x 140 levels hit in my run, i.e. about 4,300 (`vam_vacuous.py`: 131..147 distinct levels per skill); the cases-per-row figure is inflated about 12 times. HOLDS WITH LIMITS as a coverage number; the equality itself is solid. Vacuous rows (no calc cell non-zero in any case, so "0 diff" compares nothing): **6 Magic Arrow, 7 Fire Arrow, 11 Cold Arrow, 14 Power Strike, 16 Exploding Arrow, 20 Lightning Bolt, 21 Ice Arrow** (V run, `vam_vacuous.py`); 7, 11, 16, 20, 21 are covered by the roll line instead; **6 and 14 have nothing behind their 0-diff** (their effect is in the missile row, D5 in the subject, and in the melee do function, both unread).

**A2. Value tables (Q2, Q5).** Sampled from my game-side outputs (grid cases, any clvl): Multiple Shot calc1 2 / 11 / 21 / 24 at L1 / 10 / 20 / 40, calc3 2; Strafe 4 / 5 / 2 (L1), 10 / 50 / 4 (L10), 10 / 100 / 7 (L20), range 35; Fend 12 / 70, 160, 260; Jab -15 / 12 / 42; Impale 300, 525, 775 percent, 46 / 30 / 25 break, durability 1; Decoy 10 / 250 / 50, 100 / 1375, 200 / 2625, resists 4 / 40 / 80; Slow Missiles 33, length 300 / 1650 / 3150, radius 20; Inner Sight -40 / -305 / -815, length 200 / 1100 / 2100, radius 20; passives Critical 16 / 56 / 68 / 76, Dodge 18 / 47 / 56 / 62, Pierce 23 / 71 / 85 / 95, Penetrate 35 / 125 / 225 / **425**; Valkyrie L1 25 / 150 / 25, 12 / 2 / 0 / 40. All equal the author's table except **Penetrate L40: the author prints 415, game.exe's cell is 425 (WRONG cell, arithmetic slip, no consequence beyond the table)**. VERIFIED otherwise.

**A3. D1, the missile roll.** Own read: FUN_005a8910 (unit, min stat, pct stat; EAX = max stat) reads the two stats through FUN_00625480, swaps them when min > max (0x5a894b..0x5a8951), adds the percent part when the pct stat is non-zero, then calls FUN_0045c3e0 with ecx = unit + 0x20 and edx = max - min, and adds the minimum (0x5a8987..0x5a8993). FUN_0045c3e0 (V read, 0x45c3e0..0x45c429): n < 1 returns 0 without a step; otherwise one step `low * 0x6ac690c5 + high` and the result is the new low `& (n - 1)` for a power of two, else `% n`: a value in [0, n), so the roll is `min + [0, max - min)`: **top exclusive**. d2d `Rng::range` = `min + rng(max - min + 1)` (rules.hpp:169, V read): **top inclusive** with the same generator. V run: 20,000 cases (min 1..3,000, spans 0..500, seeds drawn at random; 18,454 with a span): game result == max in **0**, game result outside [min, max - 1] in **0**, game result differs from d2d's own `Rng::range` on the same seed state in **15,113 of 20,000** (every case where the modulus differs and the draws land apart), the seed state after the call is equal in all cases (both take one step). VERIFIED for the rule and its generator. NOT re-run: the 118,000 life-level counts (13,353 and 10,228 differ before, 0 after the fix `23e2baec`), which are the author's harness through `missile_blow`; I take them as the SKMISS result (already verified for the Sorceress rows by that card). The element part only; the weapon's SrcDam part is not in either run.

**A4. D2, Pierce.** Own read of FUN_0059f940 (0x59f940..0x59fa25, stdcall 1 arg, EAX = shooter): gates: missile unit type 3 (`cmp [ecx], 3`), shooter type 0 or 1, the missile row (FUN_0046ace0, row +4) `& byte [0x6ce270]`; chance = FUN_00625480(shooter, 0xa6) + FUN_00625500(shooter, 0x9c) (166 + 156), zero ends; seed = FUN_00650e40 of the shooter's stat 0x148 read through FUN_006253b0; up to four steps of the generator, `low % 100 < chance` (signed compare), stop at the first failure; the pass count is written by FUN_00627260 to the MISSILE's stat 0x148 (0x59fa05..0x59fa1a). The generator's state is a stack copy and is **not** written back. Matches the author's steps 1..5 exactly.

V run (`vam_pierce.py`, own model vs the real function, three separate stat dictionaries so a swapped stat id would show): 42,546 + 56,546 cases (chance -5..120 incl. 0, item pierce 0 / 10 / 20 / 50, shooter stat 0x148 = 0..39, 2^32 - 1, 2^31 - 1 and random 32-bit values), **0 mismatches**; gates: shooter type 2, missile type 2 and a row without the bit give nothing, types 0 and 1 work; the function never writes the shooter. At shooter stat 0x148 = 0: count 0 for chance <= 66 and 4 from 67 (equals the author's number); other seeds give other thresholds (first pierce at chance 52 for stat 1, 37 for 2, 26 for 3, 11 for 4, 100 for 5). The pierce bit: `vam_mrow.py` reads the compiled rows: bit 0x4 of row +4 is set on **60 rows whose ids are identical to the 60 rows of Missiles.txt with Pierce 1** (V run vs D); `guidedarrow`, `chargedstrikebolt` and `lightningstrike` have Pierce empty (D; the author left them unchecked); arrow, javelin, the eight element arrows / bolts, `magicarrow`, `multipleshotarrow` / `bolt`, `strafearrow` / `bolt`, `lightningjavelin`, `lightningfury`, `poisonjav`, `plaguejavelin`, `pilum`, `throwaxe` have it. Collision tail (own read, 0x5ada95..0x5adae4): the row (index * 0x1a4 + table +0xb64, bounds-checked against +0xb6c), `& mask` of row +4, the missile's stat 0x148 read, when non-zero set to value - 1 and the function answers 2 in EDI; the head of that function (where EDI starts) I did not read.

**The inference "the count is the same for every arrow of a character" is not safe, and probably wrong.** The author's dead end 2 lists 16 setter sites for stat 0x148 and read none. I read the one at 0x5980a1..0x5980b0 (own disassembly, 0x597f00..0x598119): in a function that dispatches on a type 0..0xd and, for type 1, builds and sends a packet through FUN_0053bb70, the shooter's stat 0x148 is read through FUN_006253b0 and set to **value + 1** (`add eax, 1; push eax; push 0x148; push esi; call 0x627260`). So stat 0x148 on a unit that sends this action is a counter that grows by one per action: **the seed of the pierce roll advances per shot** (V read for the +1; U for which function this is, which actions it counts and whether the server side sees it). With the counter advancing, my run of the real function on shooter counters 0..2,999 gives, per chance: P(at least one pierce) = **0.102 / 0.201 / 0.302 / 0.400 / 0.502 / 0.600 / 0.695 / 0.800 / 1.000** at chance 10 / 20 / 30 / 40 / 50 / 60 / 69 / 80 / 100, mean pierces per arrow 0.115 / 0.256 / 0.446 / 0.670 / 0.936 / 1.282 / 1.712 / 2.357 / 4.0 (the 0..4 distribution at 69: 915 / 638 / 497 / 295 / 655 of 3,000). That is a chain: the first roll passes with the chance, each next roll again with the chance, at most four. So **if the counter advances, game.exe's Pierce is "chance per unit, up to four units"**, the same marginal law as d2d's one roll per collision (`rng(100) < chance`, no cap, fight.cpp:2971) except for the cap, and the table's 69 percent at Pierce level 9 is a real 69 percent per unit, not a threshold. The author's statements "d2d never pierces (I from S)", "game.exe pierces 0 at levels 1..8, 4 at 9+" (drill A1) and bot finding B3's "threshold 67" depend on the unproven constant-counter reading and should be reworded (I below).

d2d side (S, read at upstream main; NOT run): fight.cpp:2971 `missile_info.collide_kill && !(missile_info.pierce && int(rng(100)) < int(actor->psum[kPierceIndex]))`; `kPierceIndex = 328` with the comment "pierce_idx: chance to pierce" (d2s_items.hpp:190); the passives are passed to the panel as (stat id, value) pairs (fight.cpp:577..578, `passive_stats`), so (I, the panel code was not read) Pierce (stat 166) and an item's piercing attack (156) land in their own ids, and `skill_pierce` / `item_pierce` / 166 / 156 do not appear among d2d's named stat constants (grep of d2s_items.hpp and combat.hpp: only 328 and the 333..336 passives pierces). So d2d reads a percentage from a stat the character never has (S): D2 stands as a d2d gap, not as a proven one.

**A5. Ledger lines (earned?).**
- `skills.ama 6,8-10,12-15,17-19,22-26,28-30,32-34 TESTED-0-DIFF 27800`: the equality is earned (A1) for the calc aspect named in the note; but rows 6 and 14 are vacuous (no calc cell) and row 33 (Pierce) is in a 0-diff line although its effect is the D2 gap. The number 27,800 is cases, about 4,300 distinct inputs. Proposed repair, after the card merges: split 6 and 14 out to `READ-ONLY` or `UNTESTED` (note "no calc cell; effect in the missile row / do function"), and 33 to its own line (calc cells TESTED-0-DIFF, effect D2 U), and say "about 4,300 distinct (skill, level) inputs" in the note of the remaining line.
- `skills.ama 7,11,16,20,21,27,31,35 TESTED-DEVIATION` and `missiles ... TESTED-DEVIATION`: earned: the rule is verified by my 20,000 roll cases and the author's differential through `missile_blow`; 7, 11, 16, 20, 21 are not vacuous for this reason only because the roll aspect is the claim.
I did not edit `docs/coverage/ledger.tsv` here: the three lines do not exist on this branch's base (the generator refuses two lines claiming one row), so the repair above is for the merger or the next card; `coverage.py --write` leaves the table unchanged.

## 3. Deviations (d2d vs game.exe), verified status

| id | rule | tag after this check | differential |
|---|---|---|---|
| D1 (subject's D1) | missile element roll `min + [0, max - min)` (game) vs `min + [0, max - min]` (d2d) | **V run** (own, 20,000 cases, game vs d2d's own `Rng::range`, 15,113 differ), V read both sides | yes (mine: `vam_roll.sh`; author's through `missile_blow`) |
| D2 (subject's D2) | game: chance = stat 166 + 156 of the shooter, chain of up to four rolls on a seed from the shooter's stat 0x148; d2d: one roll per collision against psum[328] as a percentage | game side **V run** (99,092 own cases, 0 vs my model, bit identical to Missiles.txt), d2d side **S** | buildable, see section 5; first settle who increments stat 0x148 (A4) |
| D2b (new) | game's stat 0x148 on the shooter is a per-shot counter (V read +1 at 0x5980a1..0x5980b0), the missile's stat 0x148 a remaining-pierce count; d2d treats the shooter's 328 as a percentage | **V read** (the +1), U (caller, which actions) | needs the caller read; then a `Fight` harness as in the subject's D2 |
| D3, D4, D5 | as in the subject | **NOT VERIFIED** (S / U) | as the subject says |

No PR candidate: D1 is the SKMISS candidate already; D2 has no d2d-side run.

## 4. Bot findings (described, not changed)

Subject's B1 (Paladin's `Charged Strike`, rules.go:238): **VERIFIED by D**: Skills.txt row 24 `Charged Strike` has charclass `ama` (Paladin's Charge is row 107, `pal`). B2 (Amazon list leaves out Lightning Bolt, rules.go:244): **VERIFIED by D** (row 20 `ama`, SrcDam 96, lightning 1..40, missile `lightningjavelin`; the list has only Lightning Fury and Charged Strike for lightning). B3: **needs rewording**: the multishot plan's Pierce rows (amazon_multishot_endgame90.json:450, 463, 476 and six more up to 554, nine rows in all) are worth something in game.exe if the counter advances (a real chance per unit, up to four, A4) and nothing in a d2d soak (S); the "threshold 67 under the stat-328-is-zero reading" is not what a character sees (stat 328 is a counter). B4, B5: not re-checked (cells of Decoy confirmed: 250 / 1375 / 2625 frames, 50 percent).

New (B6): none beyond rewording B3.

## 5. Drill recipe (for later, not run here)

| Drill | Confirms | How |
|---|---|---|
| P1 pierce stream | D2b, A4 | `debug points 33 10`, bow, ten monsters on a line (`debug spawn`), `dump stats` before and after 20 shots: does stat 328 of the player rise by one per shot (V read +1)? Then count units struck per arrow over 200 arrows: P(at least one) should be about 0.7 at Pierce 69 percent, up to four units, if the chain model holds; the subject's A1 prediction (0 pierces below 9, 4 at 9+) is the other reading |
| P2 which action | the U of who calls 0x5980b0 | break-on-write of stat 0x148 of the player (`debug` verbs of docs/forkdbg.md) on a plain bow attack, a Strafe and a melee swing |
| P3 d2d side | D2 | the same drill in the lab binary: units struck per arrow at Pierce levels 1, 9, 20 against the 0..4 chain above |
| P4 roll | D1 | as the subject's A4 |

## 6. Matrix-cell updates (proposed text)

| Row (subject's section 7) | New value |
|---|---|
| all 30 Amazon calc cells | V run equal to d2d, own 52,300 cases (about 4,300 distinct inputs); no cell reads a stat except Valkyrie's sumskill levels |
| Magic Arrow 6, Power Strike 14 | no calc cell exists: calc differential vacuous; effect U (missile row DmgCalc1, melee do function) |
| Pierce | per-shot counter: stat 0x148 +1 at 0x5980a1 (V read); chain of up to four rolls at the chance 166 + 156 (V run function); d2d reads neither (S) |
| Penetrate | passive value at level 40 is 425 (not 415) |
| `guidedarrow`, `chargedstrikebolt`, `lightningstrike` missiles | no pierce bit (D, V run compiled bit) |

## 7. Dead ends and limits

1. Valkyrie's read of skill 23 (Penetrate) has no explanation in the row text; harmless for equality (0 diff), U for meaning.
2. The caller of the +1 at 0x5980b0 is unread (the function looks like the client's action sender; U whether the server's copy of the player counts the same way). Until it is read, "the counter advances per shot" is V read for the instruction and I for the effect on pierce.
3. d2d's real build was run only for the calc cells (D) and for `Rng::range` (D1); the d2d side of Pierce was read, not run.
4. The 118,000 life-level cases of the subject were not re-run.
5. Local interpreter calls were not allowed for tools outside the worker host in this job; the tables were read with the worker host's table dump and a stdlib script.

## Shared-file lines for the merger

See `openspec/changes/verifyskamazon/merge-lines.txt`.
