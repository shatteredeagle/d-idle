# Attack rating from gear: stats 115, 116, 123, 124, 179 (and 120) in game.exe 1.14d vs d2d vs our bot (SKGEARAR)

> status: current (2026-11-02, agent SKGEARAR). Follows docs/research/sk-melee2-trace.md (row D3) and sk-melee-trace.md Q2. Offline: no game, no d2d slot, nothing pushed or posted. Tags: **V read** = read in the disassembly (upstream `disasm.py`); **V run** = game.exe's own function run under unicorn against a written reading; **D** = tables; **S** = an earlier trace; **I** = inference; **U** = unverified. No game code is pasted: function tags and short formulas only. Limits are listed in the last section.

## Summary

1. FUN_0057d8a0 (V run, 100,960 cases, 0 differ from the reading below) applies four stats to a player's to-hit: 115 (defense to 0), 116 (-% defense), 123 / 124 (flat AR vs demon / undead).
2. It is called from the to-hit FUN_0057d9b0 only when the attacker is a player (unit type 0); a monster attacker skips it (V read, 0057da1f..0057da30 branch on the attacker's type).
3. Stat 179 is a percent AR term per target MonType, summed into the same percent as mastery and stat 119 (V read; not run, Q4). Stat 120 is not an AR term to the roll: it raises the target's defense inside the damage build (Q5, V read).
4. d2d has no input for any of them (S sk-melee2-trace.md D3). Differential for 115 / 116 / 123 / 124 on the game side exists now (`tools/emu/sk_gearar.py`); the d2d side is "term absent", so the before count is the share of rolls with the stat live.
5. No PR candidate (Q6).

## Q1 FUN_0057d8a0: order and rules (V read; V run 100,960 cases)

Signature (V read): attacker unit and target unit in registers, pointers to the AR and to the defense that FUN_0057d9b0 holds; stdcall, two stack arguments. Stats read through FUN_00625500 (attacker), in this order:

1. **Stat 115** (nonzero): if the target is a monster (type 1) and not "boss-ish" (FUN_005a0180 with mask 10 clear in the flag word of the monster data at target + 0x14 / +0x16, FUN_0063e9f0 clear, FUN_0063ee90 clear), the defense becomes 0. The value is not a percent; any nonzero value does it. A player target or a boss-ish monster: no effect. I: the mask-10 test is the "boss/unique/champion" family; which bit is which was not resolved (U).
2. **Stat 116** (only when over 0): p = the stat; halved (C division, 1 becomes 0) when the target is a player (type 0), or a monster that is boss-ish by a second test (FUN_005a0180 mask 2, or FUN_0063e9f0, or FUN_0063ee90 non-zero); a null target is not halved; capped at 100. defense = defense - MulDiv(defense, p, 100) (FUN_00483360). It runs after 115, so with both, defense stays 0. A negative 116 does nothing.
3. **Stat 123**: nonzero and the target is a demon (FUN_0063e940, the monster's flag byte +0xd & one global mask): AR += stat (flat, can be negative).
4. **Stat 124**: same for an undead (FUN_0063e990; two masks). AR += stat.
All four work on the values FUN_0057d9b0 passes, which are: AR = FUN_00622560 (stat 19 + dex part + CharStats factor) and defense = FUN_006223f0 + the melee-defense stat (0x21, 0x20 for missiles), both from the target. The 123 / 124 terms are added BEFORE the percent (Q3); 115 / 116 act on the defense BEFORE the sign fixes.

Boss-ish helpers (V read): FUN_0063ee90 returns nonzero for exactly the class ids 0x10f, 0x152, 0x167, 0x230 and 0x231 (1, 2, 3, 4); which monsters those are was not looked up (U, a monstats read would say; D-tag possible later). FUN_005a0180(unit, mask) = (flag word & mask) != 0 for a monster with data.

Run: `tools/emu/run-on-worker.sh --ttl 1500 sk_gearar.py run 100000 7` (seed 7, 960 boundary cases: stat 116 in {1,2,3,99,100,101,200,-5} x 5 boss kinds x 8 defenses x target types 0 / 1 / 2, then 100,000 random). Result (V run): 0 differ from the reading in 100,960. Stubbed: the stat getter, FUN_0063e940, FUN_0063e990, FUN_0063e9f0. Real: FUN_005a0180, FUN_0063ee90, FUN_00483360. Stat 115 on a plain monster zeroed the defense in 15,296 of 15,306 cases (the other 10 had defense 0 already); on mask-2, mask-8, FUN_0063e9f0 and class-0x167 targets it did not (the few "zeroed" there come from 116 at 100 percent or a defense already 0).

## Q2 What changes for the roll (V read + I)

After Q1, FUN_0057d9b0 (V read) takes percent = mastery to-hit (weapon, melee) + stat 119 + the bonus argument + the stat 179 entries (Q4); AR' = AR + MulDiv(AR, percent, 100); defense under 0 moves to AR; chance = AR' x 100 / (AR' + def), x 2 x alvl / (alvl + dlvl), 5..95 (S sk-melee2-trace.md Q1, V run there with the terms hooked to 0). So 123 / 124 enter the percent base, and 115 / 116 only matter through the defense, where a lower defense lifts the chance toward the 95 cap.

Share of rolls that differ (I, not a character census): in the random grid the live terms changed AR or defense in 83,243 of 100,960 cases, but that grid draws the stats on purpose. For a real character the share is the share of gear with the stat: the count for gear-built characters cannot be had from this card (no inventory sample; U). The 1.14d LoD item tables hold these as affixes on rare/unique items only (I from sk-melee2 D3: "gear only; none on the starter gear"); a table count with `mpq-table.sh itemstatcost` and a properties join is the way to put a number on it (open).

## Q3 Order summary (V read)

target defense (stat + melee defense stat) -> 115 -> 116 -> (AR + 123 / 124) -> x percent (mastery + 119 + bonus + 179) -> sign fixes -> chance. Percent 0 and gear off give exactly d2d's `hit_chance` input (S).

## Q4 Stat 179 (V read, not run; U for the table side)

FUN_0057d9b0: only when the target is a monster whose data record exists: stat 179 is a list (FUN_006261d0, up to 0x80 entries of (type word, value dword)); each entry whose type matches the target's MonType word (data + 0x1c) under FUN_0057a830 (a bit test in the MonType group table: type is a member of the target's type) adds its value to the percent. Percent, not flat. The sign is kept. The stub for a run exists in outline (hook FUN_006261d0 and FUN_0057a830) but needs register-aware hooks the harness lacks; not run (U). Tag: V read.

## Q5 Stat 120 (V read, not run)

FUN_0057b7d0 (the damage build), attacker a player (or a unit FUN_006259b0 calls type 2) striking a monster: stat 120 (the attacker's) is added to the target's defense stat 31 (floor 0, written back with FUN_00627260) before the damage record is built. So it is not an AR term: the to-hit of that swing is already rolled by then (S sk-melee-trace.md Q2 lists it in the same place). Whether the changed defense is restored afterwards, and what it could affect (damage reduction does not read the defense stat; AI and a later to-hit might) was not traced (U).

## Deviations d2d vs game.exe

| ID | Rule on both sides | Tag | Differential |
|----|--------------------|-----|--------------|
| D1 | Stat 115: game.exe zeroes a non-boss monster's defense; d2d has no input and ignores it | V run (game side) / V read (d2d, S D3) | yes for the game side now (`sk_gearar.py`); the d2d side would be a test around `hit_chance` with the defense forced to 0: buildable from `sk_melee2_test.cpp`, not built |
| D2 | Stat 116: game.exe cuts defense by min(p,100) percent, halving against bosses and players; d2d ignores | V run / V read | same as D1 |
| D3 | Stats 123 / 124: flat AR vs demon / undead before the percent; d2d ignores | V run / V read | same |
| D4 | Stat 179: percent AR per MonType; d2d ignores | V read | buildable: hook FUN_006261d0, FUN_0057a830 and FUN_00451f80 (a monster-data record), 100,000 cases like sk_melee2 H |
| D5 | Stat 120: raises monster defense in the damage build; d2d ignores | V read | the effect on later logic is unknown (U); no differential until it is |

## Bot findings

- B1 (I): the bot computes to-hit/AR only from the character's stats and has no gear-AR term either; the files to check are `internal/combat/melee.go` and its tests (not read in this card; location U). Described, not changed: if the bot ever scores gear by "attack rating", stats 115 / 116 / 123 / 124 / 179 should count only against the matching targets, and 115 / 116 not at all against bosses.
- B2: nothing in this card shows the bot wearing items with these stats; no finding beyond B1.

## PR candidate

No PR candidate: the differential is game-side only here; a d2d deviation is "proven" when the d2d side computes a different chance for a case with a live term, which needs the d2d inputs (a stats API that carries 115 / 116 / 123 / 124 / 179) to exist first. Adding them is a feature, not a one-line fix.

## Drill recipe (lab, fork verbs; docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md)

Confirm later (U): (a) which monsters FUN_0063ee90 and the mask-10 / mask-2 bits mean (dump monstats Id 271, 338, 359, 560, 561 and the flag word of a champion, a unique, a minion in the lab); (b) stat 179 sign and match: give a character an item with 179 for MonType X and attack an X and a non-X with a fixed seed, compare the roll/hit; (c) stat 120's lifetime: read the target's defense (stat 31) before and after one swing. Fork verbs: set/read unit stats and spawn a given monster class, per docs/forkdbg.md. Not run.

## Matrix-cell updates (proposed)

docs/research/sorc-skill-matrix.md and the melee row of the skill matrix: "AR from gear: game.exe applies 115 / 116 / 123 / 124 / 179 (players only); d2d none; gear only; V run for 115 / 116 / 123 / 124 (0 differ, 100,960 cases)." docs/coverage/ledger.tsv: no row added (skills.monster row 0 is already taken by sk-melee2-trace.md; a merger may fold this doc into that note).

## Limits not covered

Stat 179 and stat 120 not run; the boss-ish bit meanings, and the class ids 0x10f / 0x152 / 0x167 / 0x230 / 0x231, not identified; no d2d-side harness for these terms; no census of gear carrying them; the stubs of FUN_0063e940 / FUN_0063e990 / FUN_0063e9f0 stand in for the monster tables; monster attackers skip FUN_0057d8a0 (V read) and were not run.

## Shared-file lines for the merger

See openspec/changes/skgearar/merge-lines.txt.
