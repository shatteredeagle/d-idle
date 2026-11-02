# Second pass on the resistance rules: Q3, Q7, Q9 (VERIFY-SKRESIST, part 2)

> status: card verifyresistq, 2026-11-02, offline (no game, no d2d slot, nothing pushed). Verifier: Claude Sonnet. The first pass (verify-skresist.md, card verifyskresist) left Q3, Q7 and Q9 at U; this doc covers those three. Order of work: I read FUN_005a1370 and FUN_0057cb00 in the disassembly and wrote my own models first, ran them against game.exe, and only then compared with `sk-resist-trace.md` Q3 / Q7 / Q9. No code is pasted.

Tags: **V read** disassembly of the patched game.exe 1.14d (upstream `disasm.py` through `tools/emu/vq_dis.py`); **V run** game.exe's own function run natively under the emulator against my own model; **D** MPQ table (patched stack); **I** inferred; **U** unverified.

## Harness (mine)

| file | what |
|---|---|
| `tools/emu/vq_dis.py ADDR [COUNT]` | disassemble the patched game.exe |
| `tools/emu/vq_mods.py N SEED` | Q3: FUN_005a1370 run for every modifier id, stat getter / setter FUN_00625480 / FUN_00627260 hooked to a dict, against my model |
| `tools/emu/vq_gethit.py N SEED` | Q9: FUN_0057cb00 run on records with random damage, kind, unit size, helper answers |
| `tools/emu/vq_curse.py SKILLS.TSV` | Q7: the curse values per level from the Skills table (plain arithmetic, no emulator) |

Run: `tools/emu/run-on-worker.sh vq_mods.py 20000 11`, `... vq_mods.py 60000 2025`, `... vq_gethit.py 20000 11`, `... vq_gethit.py 60000 2025`; table: `tools/emu/run-on-worker.sh mpq-table.sh skills > skills.txt` (drop the first labrun line), then `python3 -I tools/emu/vq_curse.py skills.txt`. Scaffolding idea (hooks, `emu.call`) is the same as the first pass; no case or model comes from `sk_resist.py` or the author's trace.

## Q3: boss / champion / unique modifiers on the resistances

**Read (V read, FUN_005a1370, ecx = unit, edx = modifier id, stack arg = a flag).** Nothing happens unless the unit is a monster (type 1) and the flag is non-zero. It reads the six resist stats (0x24 physical, 0x25 magic, 0x27 fire, 0x29 lightning, 0x2b cold, 0x2d poison) and counts how many are at 100 or more; the six are all counted, physical and magic included. For modifier 28 (Stone Skin, `stoneskin`) the stat 0x1f (defence) is first doubled, before the count test. With a count of 2 or more it returns, nothing else is added. Otherwise, by monumod id (D, monumod.txt):

| id | name (D) | effect (V read) |
|---|---|---|
| 8 | resist (Magic Resistant) | cold, fire, lightning in this order, each +40 only if under 100; the count is re-tested before the 2nd and 3rd; stops at count 2 |
| 9 / 18 / 17 | fire / cold / lightning (Enchanted) | +75 to fire / cold / lightning, flat, **even if it is already 100 or more** (no under-100 test) |
| 23 | poisonhit | +75 poison, same flat rule |
| 25 | manahit | +20 **magic** resistance (0x25), flat |
| 27 | spectralhit | cold, fire, lightning in this order, each +20 only if under 75 (so the result is 94 at most and never crosses 100) |
| 28 | stoneskin | +50 on stat 0x24 (physical, "damage resist") |

All writes go through the same setter FUN_00627260 with layer 0 as the other stat code; that this setter writes the unit's own (base) list, which is why the 1/5 rule sees the value as base, is **I** (not run here; the first pass's Q6 run only exercised the 1/5 side).

**Run (V run):** my model written from the table above against FUN_005a1370 on random stat sets (boundaries -100, 0, 34, 35, 59, 60, 74, 75, 76, 99, 100, 101 mixed with random), all eight active ids plus 14 inactive ids (0..3, 5..7, 10, 13, 16, 26, 29, 32, 48), unit type 0 / 1 / 2 / 3 and flag 0 / 1: seed 11, 20,000 cases, **0 mismatches** (5,838 changed the stats); seed 2025, 60,000 cases, **0 mismatches** (17,319 changed). Only getter layer 0 was ever passed. Inactive ids, a non-monster and a zero flag never change anything.

**Verdict Q3: HOLDS WITH LIMITS** against the author's row ("applied once at spawn by FUN_005a2120 -> FUN_005a1370 ..., nothing more once two immunities exist"). Differences and limits:

- The author's list omits modifiers 23 (+75 poison), 25 (+20 magic) and 27 (+20 on the three elements, under 75); only "Spectral" is mentioned. The bot's fallback knows Mana Burn's +20 magic (affix.go) but not poison or Spectral.
- "+75 by difficulty rows": the function adds a flat 75; I found no difficulty input in it (U for whether the caller chooses the id by difficulty; FUN_005a2120 not read).
- The count test is on all six resistances before ANY add (count 2 stops even a +75 that would not cross 100), and the enchanted +75 is applied on top of an existing immunity; it is not a "would create a third immunity" test on the result.
- Not run: the caller FUN_005a2120 (which ids a boss / champion / unique gets), Stone Skin's defence side beyond the doubling model, d2d's `resist()` lambda (the author's S1 differential is theirs, not re-checked).

## Q7: the curse values

**Data (D, Skills.txt patched, `vq_curse.py`):**

| skill | stat and calc column |
|---|---|
| Amplify Damage (66) | damageresist `-par5`, Param5 100 -> -100 |
| Decrepify (87) | damageresist `par5`, Param5 -50 -> -50 (also velocity, damage %, attack rate `par5`) |
| Lower Resist (91) | fire, light, cold, poison `-dm56`, Param5 25, Param6 70 |
| Conviction (123) | fire, cold, light `-min(ln34,150)` with Param3 30, Param4 5; `skill_armor_percent` `-dm56` with Param5 40, Param6 100 |

All four names and Param values agree with the author's row. Magic resistance is on no curse row (D, all four rows read), agreeing with "no 1.14d curse lowers magic resistance".

**Arithmetic (I; the engine's calc evaluator FUN_00646ca0 was not run):** `dm56 = P5 + (P6 - P5) * dm(L) / 100` with `dm(L) = 110 L / (L + 6)` integer division, `ln34 = P3 + P4 (L - 1)`:

| L | Lower Resist | /5 on an immune | Conviction | /5 | Conviction defence % |
|---|---|---|---|---|---|
| 1 | -31 | -6 | -30 | -6 | -49 |
| 2 | -37 | -7 | -35 | -7 | -56 |
| 5 | -47 | -9 | -50 | -10 | -70 |
| 10 | -55 | -11 | -75 | -15 | -80 |
| 20 | -62 | -12 | -125 | -25 | -90 |
| 30 | -65 | -13 | -150 | -30 | -94 |

Lower Resist L1 = -31 and Conviction L1 = -30 (+5 per level) match the player-facing values I remember from the public skill tables (I; not looked up here). With the first pass's V run of FUN_005c3540 (change of 0 or less on an immune base / 5, truncated toward 0) an immune takes -6 at L1: a base of 100 falls to 94 (no longer immune) and a base of 110 falls to 104 (still immune; matches the author's figure).

**Verdict Q7: HOLDS WITH LIMITS.** Values: D, holds. The dm56 / ln34 arithmetic is I (formula from the upstream notes, agrees with the author's -31 and with the public figures); the calc evaluator was not run, so the numbers are not V run. Amplify -100 and Decrepify -50 on an immune physical base are cut by the same 1/5 rule (-20, -10): this follows from the first pass's Q6 run but the physical case was not exercised against these two skills.

## Q9: what an immune hit still does

**Read and run (FUN_0057cb00, unit, record, kind; returns 1 = skip the get-hit reaction).** The record's final damage in 256ths is at +0x4c. V read: returns 1 at once when FUN_00639df0(unit, 1) is true, when +0x28 is non-zero and equal to +0x4c, and when +0x4c is under 0x100 (one point). Otherwise the damage is compared with fractions of the unit's max life (FUN_00625d10 divided by 8, 0x10, 0x20 or 0x40 by kind) and two flag helpers (FUN_00472210), so a hit of one point or more may or may not start a get-hit.

V run (helper FUN_00639df0 held at 0, others random): seed 11, 20,000 cases: 9,893 with damage under 0x100, **0 returned anything but 1**; seed 2025, 60,000 cases: 29,944 under 0x100, **0 returned anything but 1**. Control: of 10,107 and 30,056 records at 0x100 or more, 557 and 1,615 returned 0 (a get-hit starts), so the check is not a constant. The kind table (12 entries) was not decoded; the 0x100 and above branch is not claimed.

An immune hit has final damage 0 after the resistance (first pass Q5: res >= 100 gives 0), so it is under 0x100 and never starts a get-hit. That the record's +0x4c holds the post-resistance damage on the normal missile path is **I** here (the author's S: pr-get-hit-rule-proof.md); not traced by me from FUN_0057c6c0 into +0x4c.

**Verdict Q9: HOLDS WITH LIMITS** for "no get-hit under one point of final damage" (V run, 80,000 cases, 0 exceptions). Chill, freeze, Static Field, knock-back and the on-hit events (wake-up, Lightning Enchanted's bolts) on a zero hit are still S / U (not looked at).

## Deviations d2d vs game.exe

None new. Differential possibilities: Q3, d2d's `resist()` lambda vs FUN_005a1370 over the same stat lists (the author did this for ids 8 / 9 / 17 / 18 / 28; add ids 23, 25, 27 and "count 2 stops the +75" cases, game.exe side is `vq_mods.py`); Q9, d2d's get-hit rule vs FUN_0057cb00 for damage under 0x100 (game.exe side is `vq_gethit.py`). Neither was built (d2d not run).

## Bot findings (described, not changed)

- B1 `internal/combat/affix.go:21` (`AffixResists`, fallback only): the third-immunity test is on the result of each add (`nr[t] >= 100 && immuneCount(r) >= 2`), game.exe tests the count BEFORE the whole modifier (count 2 or more adds nothing, even a +75 that stays under 100). With two immunities a non-crossing Fire Enchanted +75 is applied by the bot, not by the game. Tag V read + V run (game side).
- B2 `affix.go:57` (`immuneCount`) leaves physical (0x24) out; game.exe counts all six (V read, V run).
- B3 `AffixResists` has no Poison-hit (+75 poison), Spectral-hit (+20 under 75) or Stone Skin handling (V read).
- B4 `add` skips an element that is already 100 or more; the game adds anyway (+75 flat), so a stat read after it differs in value but not in immunity; no behaviour effect found (I).

## Not covered

FUN_005a2120 (which modifiers a boss gets); d2d's side of all three questions; the calc evaluator for dm56; the kind table of FUN_0057cb00 and its 0x100-and-above branch; chill / freeze / knock-back on zero hits; whether the monumod rows differ by difficulty.

## Matrix cell proposals (docs/research/sorc-skill-matrix.md, resist cells)

Add "second pass (verifyresistq.md): modifier adds V run 80,000 cases 0 mismatches (ids 8, 9, 17, 18, 23, 25, 27, 28), curse values D, no get-hit under one point V run 80,000 cases" to the resist / immunity cells; no value changes.

## Shared-file lines for the merger

See `openspec/changes/verifyresistq/merge-lines.txt`.
