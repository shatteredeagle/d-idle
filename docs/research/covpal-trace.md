# Paladin skills, the 15 rows no trace had run (covpal): game.exe 1.14d vs d2d vs the bot

> status: card covpal, 2026-11-02, offline (no game, no d2d slot, no Go change). Coverage-gap first pass for `skills.pal`: the 15 rows of Skills.txt (charclass pal, ids 96..125) that `docs/coverage/ledger.tsv` had no line for. Method of `docs/research/pal-skill-trace.md` and `docs/research/palskill-trace.md` (game.exe's own code under upstream's emulator against the real d2d rules), reused through `tools/emu/covpal*.{py,sh}`.

Tags: **V run** game.exe's code run natively in the emulator by this card and compared line for line with the real d2d build (upstream main, patched 1.14d tables, test modes `tools/emu/pal_test.cpp` and `pal2_test.cpp` linked against a scratch build, built by `covpal_build.sh`); **V read** read in the disassembly (`tools/emu/pal2_dis.sh`, upstream's `disasm.py`); **D** the patched 1.14d Skills.txt (`mpq-table.sh skills`, `covpal_rows.py`); **S** an upstream doc or an earlier trace of ours (named); **I** inference; **U** unverified. A rule known only from reading is not offered as proof of anything (model split of 2026-11-01).

## Which rows

The 15 rows: 100 Resist Fire, 103 Thorns, 105 Resist Cold, 107 Charge, 108 Blessed Aim, 109 Cleansing, 110 Resist Lightning, 111 Vengeance, 116 Conversion, 117 Holy Shield, 120 Meditation, 121 Fist of the Heavens, 122 Fanaticism, 124 Redemption, 125 Salvation (D: ledger lines 12, 13, 69 cover the other 15 of the class). Overlap, said plainly: the aura calc differential of `pal-skill-trace.md` (21,760 cases) already ran 22 aura rows including most of these, but the ledger line named only 10 ids, so these rows stayed UNTESTED. This card repeats the machine with its own seeds, counts per row, and adds the two aspects that run did not cover for them (the four calc slots with the to-hit bonus, and the element min / max / length of the damaging ones).

## Summary

- **Differential, V run, 0 mismatches of 9,200 + 5,200 cases per aspect** (seeds 29 and 11; 14,400 cases of each of three aspects = 43,200 comparisons, 0 differ). Per case: skill row, level 1..60, character level, masteries, 0..8 other skill levels (a fixed ladder of levels 1..40 at two synergy settings per row, then random). Aspect A: aura range, perdelay, auralen, six aurastat calcs, five passive calcs (FUN_00646ca0 on the aura slots); aspect C: the four calc slots (+0x138..+0x144) and the to-hit bonus (FUN_006449f0); aspect E: element min / max / length (FUN_00644d50 / e40 / f20). Counts per row and the rows whose values are non-trivial per aspect are in section 4.
- **Redemption's callback (V read)**: game.exe visits every unit in range each 50-frame pulse and rolls each corpse on its own; d2d takes at most one corpse per pulse and stops after the first eligible one whether or not its roll passed (D1). The numbers it uses are equal on both sides (V run, section 2).
- **Vengeance (V read + V run)**: three element damages, each a percentage of the physical blow, plus the character's element mastery term; the mastery term is absent in d2d (D2, read on both sides).
- **Bot**: its Paladin element table puts Fist of the Heavens under fire (Skills.txt: lightning) and Vengeance under magic (game.exe: fire, cold and lightning) (B1).
- **No PR candidate**: no d2d deviation here is proven by an existing differential (D1 and D2 are read on both sides; the harnesses of this card show 0 mismatches before any change).

## 1. The 15 rows (D: patched Skills.txt)

| id | skill | srvstfunc / srvdofunc | key columns |
|---|---|---|---|
| 100 | Resist Fire | - / 65 | aura (aurafilter 73731), aurarange ln12 (16 + 2 per level), perdelay 50; stats fireresist + maxfireresist, passive maxfireresist |
| 103 | Thorns | - / 65 | thorns_percent; range ln12 |
| 105 | Resist Cold | - / 65 | as 100 with cold |
| 107 | Charge | 31 / 67 | reqlevel 12, mana 9, Param3 100, Param4 25 (ln34), Param5 2, Param8 20; calc1 `ln34 + (Vigor + Might) x par8` |
| 108 | Blessed Aim | - / 65 | item_tohit_percent; range ln12 |
| 109 | Cleansing | - / 65 | item_poisonlengthresist + hitpoints; range ln12 |
| 110 | Resist Lightning | - / 65 | as 100 with lightning |
| 111 | Vengeance | 35 / 2 | reqlevel 18, mana 16, lvlmana 1, manashift 6, EType cold, calc1..3 `ln12 + (Resist Fire / Cold / Lightning) x par8 + Salvation x par7` |
| 116 | Conversion | 32 / 79 | reqlevel 24, mana 4, Param1 400, calc1 dm34, auratargetstate conversion, auralencalc ln12 |
| 117 | Holy Shield | 36 / 18 | reqlevel 24, mana 35, Param1 750, Param2 625, calc1 `ln34 + Defiance x par8`, aurastate holyshield, aurastat toblock, auralencalc ln12 |
| 120 | Meditation | - / 65 | manarecoverybonus + hitpoints; range ln12 |
| 121 | Fist of the Heavens | - / 80 | reqlevel 30, mana 25, EType ltng EMin 150 EMax 200, calc1 ln34, calc2 ln56, calc4 ln12, aurarange 20 (literal), aurafilter 42375 |
| 122 | Fanaticism | - / 65 | range ln12 (Param1 11, Param2 1); stats attackrate, item_tohit_percent, damagepercent; passive damagepercent |
| 124 | Redemption | - / 82 | reqlevel 30, mana 0, aurarange ln12 (16, 0), aurafilter 4354, calc1 dm34, calc2 ln56, calc3 ln56, perdelay 50 |
| 125 | Salvation | - / 65 | fireresist + coldresist + lightresist; range ln12 |

The do-function table (V run: `tools/emu/covpal_tab.py` reads game.exe memory at 0x7322b0 + 4 n): 2 FUN_0056f1f0, 18 FUN_005c9480, 65 FUN_005cf010, 66 FUN_005cf3a0, 67 FUN_005cf900, 73 FUN_005d0040, 79 FUN_005d0350, 80 FUN_005d0670, 81 FUN_005d0920, 82 FUN_005d0e90. Entries 65, 66, 73, 80, 81 equal the addresses the two earlier traces and upstream's `skills.md` name.

## 2. Values game.exe's calc machine gives (V run, equal d2d on every case; clvl 30, no synergy)

Levels 1 / 10 / 20 (aura rows: range subtiles; perdelay 50 for every row with an aura):

| skill | slot | L1 | L10 | L20 |
|---|---|---|---|---|
| Resist Fire / Cold / Lightning (100, 105, 110) | resist (aurastat 1) / its max (2) / passive max | 52 / 1 / 0 | 113 / 10 / 5 | 131 / 20 / 10 |
| same | range | 16 | 34 | 54 |
| Thorns (103) | thorns_percent | 250 | 610 | 1,010 |
| Blessed Aim (108) | item_tohit_percent aura / passive | 75 / 5 | 210 / 50 | 360 / 100 |
| Cleansing (109) | poison length stat / hitpoints | 61 / 0 | 30 / 0 | 20 / 0 |
| Meditation (120) | manarecoverybonus / hitpoints | 300 / 0 | 525 / 0 | 775 / 0 |
| Fanaticism (122) | range; attackrate / tohit / damage % ; passive damage % | 11; 14 / 40 / 25; 50 | 20; 30 / 85 / 101; 203 | 30; 35 / 135 / 186; 373 |
| Salvation (125) | each of the three resists | 60 | 97 | 108 (L30: 113) |
| Holy Shield (117) | auralen frames (25 a second) / toblock | 750 (30 s) / 14 | 6,375 (255 s) / 30 | 12,625 (505 s) / 35 |
| Holy Shield | calc1 | 25 | 160 | 310 |
| Charge (107) | calc1 (ED %) / to-hit bonus | 100 / 50 | 325 / 185 | 575 / 335 |
| Vengeance (111) | calc1 = calc2 = calc3 (% of the physical, fire, cold, lightning) / to-hit / cold length | 70 / 20 / 30 | 124 / 110 / 165 | 184 / 210 / 315 |
| Conversion (116) | calc1 (chance %) / auralen frames | 7 / 400 | 34 / 400 | 42 / 400 |
| Fist of the Heavens (121) | lightning min .. max / calc1 / calc2 / calc4 | 150..200 / 1 / 6 / 6 | 315..365 / 19 / 24 / 15 | 675..725 / 39 / 44 / 25 |
| Redemption (124) | calc1 chance % / calc2 = calc3 life and mana per corpse | 23 / 25 | 71 / 70 | 85 / 120 (L2 34 / 30, L30 91 / 170, L40 95 / 220) |

The max-resist and passive columns are the table's operands read as the stat the table names; what a unit does with them is not run (U). The Resist and Fanaticism stat values are the table's raw numbers (resists as percentages as in the stat); Fanaticism's passive damage percent is twice its aura damage percent on every level run (50 / 25, 203 / 101, 373 / 186): whether a Paladin gets both, one, or the passive only while the aura is off is **U** (upstream's `skills.md`: FUN_00646d60 holds the passive state off while the aura state is on, so the passive stats hold either way: S).

## 3. Rules read in game.exe (V read)

**Q1. What does Redemption do (do 82, FUN_005d0e90, callback FUN_005d0d70)?** The do puts the aura self-state like do 65 (slots from FUN_00646ca0 when FUN_0044be50(caster) is non-zero or the caster's mana covers the cost, 0x5d0f80..0x5d0f8f; what FUN_0044be50 tests is I: "not a player"), then calls the area search FUN_0056b7e0 with the callback 0x5d0d70, radius `aurarangecalc` (+0x64), filter +0x50 (4354 = 0x1102, not decoded: U). FUN_0056b7e0 (0x56b7e0..0x56b9b9) walks the rooms in range and every unit of each room, keeps a unit whose squared distance (FUN_006492a0) is within radius squared and that passes FUN_0056b3e0 (the filter), calls the callback and **continues with the next unit whatever the callback returns** (its return only adds to a counter at [ebp-0x1c]: 0x56b98c..0x56b993, then the loop jumps back at 0x56b997). The callback (0x5d0d70..0x5d0e8f): FUN_006456d0 must accept the unit (U: not decoded), then `rand(100) < calc1` (FUN_0045c390 on the caster's seed at +0x20: the roll `0x5d0dd7..0x5d0dea`), then life += `calc2 << 8` capped at max life (stat 6, FUN_00625d10, FUN_00627260), mana += `calc3 << 8` capped at max mana (stat 8, FUN_00625d60), state 0x63 (99) put on the corpse and its flags `&= ~6` (0x5d0e6a..0x5d0e77), the counter in the do's record +8 incremented. After the search the do pays mana only when the cost is above 0, the caster is a player and the counter is above 0 (state 0x55 put and FUN_0056c110, 0x5d1024..0x5d1048; with a counter of 0 it clears 0x55): **Redemption's mana is 0 (D), so it pays nothing.** So: every corpse in range, each its own roll, each gives life and mana.

**Q2. Vengeance (start 35, FUN_005cfe10)?** After the to-hit (FUN_006449f0, FUN_0057ec10) a hit: the physical from FUN_0057b420 (edx = 1), record +0x12e flags, then for each of calc1, calc2, calc3 (+0x138, +0x13c, +0x140): `v = FUN_00646ca0(...)`; when `v != 0` and the stat is non-zero, `v += FUN_00483360(v, stat, 100)` (read as MulDiv(v, stat, 100), I: the "MulDiv shares" of skill-damage-trace.md) with stat 0x149 (fire mastery) for calc1, **0x14b (cold mastery) for calc2, 0x14a (lightning mastery) for calc3** (0x5cff04..0x5cffb3); `MulDiv(physical, v, 100)` is added to the record's fire (+0x10 side), cold and lightning fields; cold length += FUN_00644f20 (ELen); an overlay cycles fire / cold / lightning (FUN_006444a0, FUN_00644560); the record goes to FUN_0057dbf0 with SrcDam 0x80. Skills.txt says EType cold only because of the length; the damage is all three. Row 111's element min / max is 0 (V run: e column 0 0).

**Q3. Fist of the Heavens (do 80, FUN_005d0670)?** Needs a target unit (FUN_00553540: none, return 0), reads the missile id of the row (FUN_005d3cf0), makes the missile at the **target's** position (FUN_0045adf0 / FUN_0045ae20 into the missile record, FUN_0059fa30), sets its path point (FUN_0064a710 / FUN_0064a760) and plays the row's srvoverlay (row +0x4e) on the target (FUN_00621e40). The lightning damage and the holy bolts at the undead are in the missile's hit function (S: upstream's `skills.md` line 884 "the damage within aurarange of its target, then holy bolts at the undead", FUN_005ab630; the missile row's hit roll is covmissiles-trace.md's row 233 with 17,174 cases, 0 differ). Mana: paid by the dispatcher, not in the do (no payment call in 0x5d0670..0x5d0762); the cost 25 is the SKMANA row (S).

**Q4. Holy Shield, Charge, Conversion.** Not re-read here. Holy Shield (st 36, do 18 FUN_005c9480): the holyshield state (0x65) for auralencalc ticks and toblock via the do; S upstream `skills.md` lines 427..431; our V run gives the numbers in section 2. Charge (st 31 FUN_005cf6b0, do 67 FUN_005cf900) and Conversion (do 79 FUN_005d0350, end FUN_005d01a0): S upstream `skills.md` and `monster-ai.md` line 735; V run for their calc slots only.

## 4. Counts (V run)

`tools/emu/covpal_run.sh N SEED` (seed 11: N 4,000; seed 29: N 8,000; each run 1,200 ladder cases + N random, 15 rows). The compare prints cases and mismatches per row and aspect; cases per row (calc / aura / elem each the same number), seed 11 then seed 29:

| row | seed 11 | seed 29 | mismatches |
|---|---|---|---|
| 100 | 338 | 582 | 0 |
| 103 | 346 | 588 | 0 |
| 105 | 337 | 604 | 0 |
| 107 | 374 | 598 | 0 |
| 108 | 374 | 615 | 0 |
| 109 | 341 | 621 | 0 |
| 110 | 338 | 604 | 0 |
| 111 | 350 | 631 | 0 |
| 116 | 331 | 616 | 0 |
| 117 | 384 | 641 | 0 |
| 120 | 338 | 609 | 0 |
| 121 | 349 | 640 | 0 |
| 122 | 362 | 637 | 0 |
| 124 | 305 | 613 | 0 |
| 125 | 333 | 601 | 0 |
| total | 5,200 | 9,200 | 0 |

Rows whose game values are non-trivial in an aspect (not all 0 or x; the compare counts them, all cases of the row in both seeds): aura record (range, length or slots) 100, 103, 105, 108, 109, 110, 116, 117, 120, 122, 125; calc slots and to-hit 107, 111, 116, 117, 121, 122, 124; element min / max / length 111 (cold length only) and 121. The other aspect cells of a row are all-zero or x on both sides, which proves only that d2d too has no slot there. For the ledger, the claim per row group is the aspect with non-trivial values: aura rows 100, 103, 105, 108, 109, 110, 117, 120, 122, 125: 9,593 cases (3,491 + 6,102); rows 107, 111, 116, 121: 3,889 cases (1,404 + 2,485, calc slots, to-hit and element); row 124: 918 cases (305 + 613, calc slots). Files and md5 of the cases and both sides were printed by the run (seed 11 game and d2d files equal: aura fe9c0785, calc 54b0a8da, element 0d9e61c1).

## 5. Deviations d2d vs game.exe

| id | rule on both sides | tag | differential |
|---|---|---|---|
| D1 | Redemption: game.exe tries **every** corpse in range each pulse (own roll `rand(100) < calc1`, life calc2, mana calc3 per corpse; FUN_0056b7e0 continues after the callback). d2d (`fight.cpp:3106..3118`, its comment: "its callback isn't traced; one corpse a pulse") takes the first eligible corpse in `monsters` order and `break`s after it whether or not the roll passed, with calc2 for both life and mana (identical values on this table: calc2 = calc3 = ln56) | V read (game), S/V read (d2d source) | possible, not existing. Build: game side = call FUN_005d0e90 under the emulator with a hand-made room whose unit list holds k corpses and a counting callback result (the stat 6 / 8 writes are the oracle), seeds 7 and 1, k 1..6, levels 1..30; d2d side = a `Fight` test with k corpses of mode DD and one `aura_pulse`. Oracle: corpses redeemed and life and mana gained per pulse. Needs the unread FUN_006456d0 and the filter 0x1102 decoded first (U) |
| D2 | Vengeance: game.exe adds the element mastery to each of fire / cold / lightning (calc + calc x stat / 100, stats 0x149 / 0x14b / 0x14a). d2d `fight.cpp:815..819` uses the three calcs only (upstream's `skills.md` lists "the masteries on Vengeance" as not built) | V read (game) + V read (d2d) | possible: game side = FUN_005cfe10 with a fixed physical and hooked mastery stats, oracle = the three element fields of the hit record; d2d side = the swing's `fire_pct` / `cold_pct` / `ltng_pct` after `skill_element`. Matters only for a character with an element mastery stat on the gear |
| - | Redemption mana: game pays nothing (cost 0); d2d pays nothing | V read | not a deviation |
| - | Aura calcs, ranges, lengths, passives, charge / Vengeance / Conversion / Holy Shield / Fist numbers | V run, 0 differ | not a deviation |

Not compared and still open (each U, none read in this card): Holy Shield's aura events and its effect on Smite (S upstream: "not in them: Holy Shield's aura events"); Conversion's effect (S monster-ai.md: alignment 2, level scaling); Charge's movement and knock-back (d2d `fight.cpp:734`, `2332`, `2364`, `831`; game do 67 FUN_005cf900 unread); Fist's bolts at the undead; every aura's apply (the unit filters 0x12003 / 0x1102 / 0xa783 are undecoded) and the Paladin aura mana per pulse and reach already listed as D5 and D6 of `pal-skill-trace.md`.

## 6. Bot findings (described, not changed)

| id | where | finding | tag |
|---|---|---|---|
| B1 | `internal/combat/rules.go:237..238` (`classOptions`, Paladin) | Fist of the Heavens is listed under fire and Vengeance under magic; Skills.txt gives Fist the lightning element (EType ltng, 150..200 at level 1) and game.exe's Vengeance does fire, cold and lightning (Q2). The comment above the table already says "U: names from memory". Effect: the element-resist pick and the immune workaround would treat Fist as a fire skill and Vengeance as a magic skill. Low impact today: neither skill is in the Paladin profile ladders (`profiles.json`, the plans train neither) and the table only offers skills the character knows and has bound | D + V read |
| B2 | `internal/combat/necro.go:42..43` (`buffStates`), `buffLimit` (`:492..`), `internal/data/skills.go:291` (`AuraFrames`) | Holy Shield (profile `buffs`, `profiles.json:157..161`) has no entry in `buffStates`, so it is recast by the clock only: the duration formula `Param[a] + Param[b] x (level - 1)` frames is game.exe's (V run: 750 + 625 per level, 30 s at level 1, 255 s at level 10, 505 s at level 20, equal d2d), but an early loss of the state is not seen. The state id is 0x65 (S upstream `skills.md` line 427; not verified in this card) | V run (duration) + S (state) |
| B3 | `internal/combat/classdata/profiles.json` Paladin aura ladder (Prayer, Concentration, Holy Fire, Might) | none of this card's auras is used by the bot; Redemption and Meditation have values worth nothing to the plan today; no finding beyond: if Redemption is ever added, d2d's one-corpse rule (D1) understates it | I |

## 7. PR candidates

None. D1 and D2 are read on both sides and have no harness that shows mismatches before a change; the existing differentials of this card show 0 mismatches. A candidate needs the D1 or D2 harness above built and run (mismatches before > 0, after 0) and an independent verifier.

## 8. Drill recipe for the fork verbs (docs/forkdbg.md, docs/forkcheats.md; written, not run)

Lab binary and a throwaway character as in the M1..M5 recipe of `forkdbg.md`.
- **D1 Redemption.** `debug points 124 30` (calc1 91 %, 170 life and mana per corpse at level 30, V run above), `debug spawn` 6 weak monsters of one type near the player, `debug mon set` life to 0 or kill them so the corpses stay (U: whether the lab keeps the corpse at mode DD long enough), `debug stat 6 <256 x 100>` and `debug stat 8 <256 x 100>` for a known life and mana, select Redemption and `debug cast 124 near` once (a Paladin with the skill selected as an aura works too), `step 60`, `dump stats`. Expected game.exe: about 5.5 of 6 corpses used in the first pulse (life and mana up by about 5.5 x 170 each, capped); d2d today: at most 170. Repeat with the corpses placed beyond 16 subtiles for the range edge.
- **D2 Vengeance.** `debug stat 329 20` (fire mastery 20) on the throwaway, `debug points 111 10`, `debug mon set` a target with a very large life, `debug cast 111 unit <id>`, read the life drop of three hits at the same weapon: game.exe per-hit element parts are 124 % x 1.2 for fire (L10, V run 124); d2d without the mastery term. U: the stat-329 write is an INSTRUMENTATION write that must be ledgered (AGENTS.md).
- **Holy Shield duration and state.** `debug points 117 1`, `debug cast 117` on self, `debug states` read every 10 s: the buffs line should list 117 for 30 s (750 frames) at level 1; the bit of state 0x65 (U).
- **Conversion, Charge, Fist.** `debug cast 116 unit <monster>` (level 1: 7 %; use `debug points 116 20`, 42 %), `debug cast 107 unit <monster>` and `debug cast 121 unit <monster>` with `debug missiles` per frame: Fist's missile must appear at the target's position (Q3).

## 9. Matrix-cell updates (proposed text)

For `docs/classes/ranking.md` and the Paladin matrix rows (no committed Paladin matrix doc exists; `docs/research/sorc-skill-matrix.md` is the Sorceress one): Paladin skills at a glance, each cell `ours / d2d / game.exe`:
- 100, 103, 105, 108, 109, 110, 120, 122, 125 (auras): numbers V run equal (9,593 cases); apply, reach, mana per pulse U or D5 / D6.
- 107 Charge, 116 Conversion, 117 Holy Shield, 121 Fist: calc slots, to-hit and (Fist) element damage V run equal (3,889 cases); effects U.
- 111 Vengeance: numbers equal; D2 (mastery term) V read; bot element class wrong (B1).
- 124 Redemption: numbers equal (918 cases); D1 V read.

## 10. Dead ends and limits

- The first run showed identical md5 for the game and d2d files in all three aspects; I checked that the files carry real values (5,200 lines; 2,455 calc lines, 699 element lines, 2,761 aura lines are not all-zero or x) before trusting the 0.
- Rows 100, 105, 110 and 124's cells in the aspects without a slot read zero on both sides; that is a statement about the table, not about the skill.
- I did not read FUN_006456d0 (the corpse accept test of Redemption's callback), FUN_0056b3e0 (the aura filter), the Charge and Conversion do functions, or FUN_005c9480's Holy Shield path; D1 stands on the callback and the area loop (both read), but "which corpses are eligible" is U.
- A game-side differential of Redemption and Vengeance would need a hand-built room with units under the emulator; not built here.
- No live check of anything: no game, no d2d slot.

## Glossary term (for the report, not edited)

None missing.

## Shared-file lines for the merger

See `openspec/changes/covpal/merge-lines.txt`.
