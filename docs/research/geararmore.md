# Attack rating from gear, second pass: stats 179 and 120 run, all six against d2d and the bot (GEARARMORE)

> status: current (2026-11-02, card geararmore, second pass of SKGEARAR). Offline: no game, no d2d slot, nothing pushed. Tags: **V run** = game.exe's own function run under unicorn on the worker host against a written reading; **V read** = read in the disassembly; **D** = 1.14d tables (patched stack); **S** = an existing trace or doc; **I** inference; **U** unverified. No game code is pasted: function tags and short formulas only. The first pass (stats 115, 116, 123, 124; 100,960 cases, 0 differ) lives in its own trace, branch `agent/skgearar`; its rules are cited here, not repeated.

## Summary

1. Stat 179 (attack_vs_montype): a percent AR term per target MonType, summed into the same percent as mastery and stat 119, **player attackers only** (V run, 100,032 cases, 0 differ). A monster attacker never reaches it.
2. Stat 120 (item_damagetargetac): not an AR term. FUN_0057b7d0 adds the attacker's stat 120 to the target monster's defense stat 31, floored at 0, and writes it back with no restore inside that function (V run, 100,648 cases, 0 differ). The 1.14d tables only carry negative values (-10 to -100), so it lowers the monster's defense for every later swing (D + V run; the lifetime beyond the function is U).
3. Stat 179 appears in none of the item tables I counted (D): no gear in the shipped data can carry it. The five other stats are on a few dozen affixes (Q4).
4. d2d: no input for any of the six (S sk-melee2-trace.md D3). The bot: the fork's `dump stats` "gear" sums do not even carry 119 (the to-hit percent the bot's model reads), see B1.
5. No PR candidate.

## Q1 Stat 179 in FUN_0057d9b0 (V run, 100,032 cases)

- Path (V read, 0057da19..0057dad5): the attacker's unit type is tested first. Type 0 (player) goes the main path; any other type jumps to a short path (AR = stat 19 + bonus argument + 5 x stat 2, percent = stat 119 only) that skips the mastery, the bonus-in-percent and stat 179. So 179 is player-only (V run: attacker types 0 and 1 are both in the grid).
- Main path: percent = mastery to-hit (only with a weapon) + stat 119 + the bonus argument; then, **only if the target is a monster (unit type 1) whose record exists (FUN_00451f80 by class id) and the record's MonType word at +0x1c is > 0 as a signed 16-bit**, FUN_006261d0 returns the stat-179 list (up to 0x80 entries of a 16-bit type and a 32-bit value) and every entry for which FUN_0057a830(entry type, target MonType) is non-zero adds its value to percent. FUN_0057a830 is a bit test in the MonType group table (row = target type, column = entry type; both must be > 0 and below the table's count at +0xc40 of the game-data struct); the 179 list counts a type once per entry (entries add, they do not dedupe).
- Then AR' = AR + MulDiv(AR, percent, 100) (FUN_00483360) and the usual chance (S sk-melee2-trace.md Q1).
- So the term is **percent, signed (a negative value lowers AR), summed with mastery and stat 119, applied after the flat 123 / 124 terms** (order: Q3 of the first pass).
- Run: `tools/emu/run-on-worker.sh --ttl 1500 gar_run.py a 100000 7` (seed 7; 32 boundary cases: MonType in {0, 1, 2, count-1, count, 0x7fff, 0x8000, 0xffff} x target type x attacker type, entry types {0, 1, count-1, count, 65535} with a +40 value; then 100,000 random: 0 to 128 entries, values {0, 20, 50, 100, -30, 250, -150}, random group table, random AR, defense, levels, mastery with and without a weapon). The chance is captured at FUN_00554010's call. Result: **0 differ from the reading in 100,032**; the 179 list changed the chance in 4,797 of them. Real code: FUN_0057d9b0, FUN_00483360, FUN_00451f80, FUN_0057a830 (on a synthetic game-data struct written at the global pointer). Stubbed: stat getters, FUN_00622560 / FUN_006223f0 (AR and defense of the units), FUN_0057d8a0 (run in the first pass), the mastery lookup, FUN_006261d0 (the list itself). The real stat list reader and the group table contents of the game's data are therefore S/U (Q5).

## Q2 Stat 120 in FUN_0057b7d0 (V run, 100,648 cases)

- Path (V read, 0057b805..0057b857): the unit that builds the damage is a player (type 0) or one FUN_006259b0 calls type 2, and the target is a monster (type 1). v = attacker stat 120; if v is not 0: new = target stat 31 + v (32-bit add), negative becomes 0, written with FUN_00627260. Then (same branch) the demon test FUN_0063e940 adds the attacker's stat 121 (0x79, when > 0) to the damage record's field +0xc (the demon damage term, out of scope here).
- The to-hit of this swing is already rolled (the damage build follows the hit), so stat 120 can only matter to LATER rolls on that monster, and to nothing in this function (damage reduction does not read stat 31; S first pass, I).
- No restore of stat 31 is inside FUN_0057b7d0 (V read of the whole function: one write, no second write of stat 31). Whether a caller restores it is not traced (U). If it does not, every hit lowers the monster's defense by |v| down to 0 (I).
- Run: `tools/emu/run-on-worker.sh --ttl 1500 gar_run.py b 100000 7` (seed 7; 648 boundary cases: attacker type 0 / 1, "type 2" on / off, target type 0 / 1 / 2, v in {0, -1, -50, -100, 5, -(2^31-1)}, defense in {0, 1, 49, 50, 51, 99, 100, 1000, 2^31-1}; then 100,000 random). Result: **0 differ from the reading**, with a write in 46,205. The first run exposed the 32-bit wrap (INT_MAX + 5 wraps negative, floors to 0): the reading is now the 32-bit add. Stubbed: every callee; the run is cut at the first call after the branch (FUN_0057b660), so nothing past the 120 branch ran (V run only for this branch).

## Q3 Whose gear carries these stats (D, `gar_census.sh`)

`tools/emu/run-on-worker.sh --ttl 600 gar_census.sh` counts table rows by property code (item_ignoretargetac = ignore-ac 115; item_fractionaltargetac = reduce-ac 116; item_damagetargetac = dmg-ac 120; item_demon_tohit = att-demon 123; item_undead_tohit = att-undead 124; attack_vs_montype = att-mon% 179). Rows with the code in any cell, patched 1.14d stack, 2026-11-02:

| table (rows) | 115 | 116 | 120 | 123 | 124 | 179 |
|---|---|---|---|---|---|---|
| uniqueitems (404) | 14 | 13 | 4 | 4 | 6 | 0 |
| setitems (129) | 1 | 0 | 0 | 1 | 0 | 0 |
| magicprefix (671) | 0 | 0 | 0 | 7 | 6 | 0 |
| magicsuffix (749) | 2 | 0 | 4 | 0 | 0 | 0 |
| runes (170) | 4 | 3 | 1 | 1 | 1 | 0 |
| gems (70) | 1 | 1 | 0 | 1 | 1 | 0 |
| automagic, cubemain | 0 | 0 | 0 | 0 | 0 | 0 |

(Rows are table rows, not items: a table row can be a base-type family; the two "of Piercing / Bashing / Puncturing" suffixes repeat per item class.) Values (D): 120 is -10 to -100, so it is a defense cut; 124 up to +100 (uniques) and 123 up to +250. **Stat 179 is on no row of these eight tables** (D); the property exists but nothing in the shipped tables grants it (mods that add it, if any, are outside the tables counted; U).

Gear-built characters: the share of characters with at least one of 115 / 116 / 123 / 124 / 120 on worn gear cannot be computed from tables alone (it needs the drop and gear distribution). Bounds only (D, I): the stats are on 3 to 4 percent of unique rows and ~2 percent of magic affixes (e.g. 13 of 671 prefixes for 123 / 124, 6 of 749 suffixes for 115 / 120); a character wearing none of those rows has an identical to-hit on d2d and game.exe. A census over real characters was not possible here (U; the drill below gives a way: `dump stats` "gear" once the fork carries these stat ids).

## Q4 Order against the other terms (V read + V run)

defense (target stat + melee-defense stat) -> 115 -> 116 (first pass) -> AR + 123 / 124 (flat, vs demon / undead) -> percent = mastery + stat 119 + bonus + sum of the 179 entries matching the target MonType -> AR' = AR + MulDiv(AR, percent, 100) -> sign fixes -> chance. Stat 120 is outside this chain (after the hit, Q2). Percent 0 and no gear gives exactly d2d's `hit_chance` input (S).

## Q5 What stays unproven

- The real FUN_006261d0 list reader (it walks the item stat list under the unit's stat-list flag 0x80000000): stubbed (S, small).
- The contents of the MonType group table (which MonTypes belong to which): read from game data, not extracted here (U; a `monstats` + `montype` table join would give it, D later).
- Lifetime of the stat-31 change (Q2) and the sign convention in 1.14d item text: values are negative in the tables (D); a drill (below) shows the lifetime.
- Monster attackers: skip the whole block (V read, V run for 179).

## Deviations d2d vs game.exe

| ID | Rule on both sides | Tag | Differential |
|----|--------------------|-----|--------------|
| D1 | Stat 179: game.exe adds the matching entries' values to the AR percent (player vs monster); d2d has no input | V run (game side) / S (d2d side, D3 of sk-melee2) | game side: `gar_run.py a`; d2d side: a test around `player_blow`'s hit with a percent input, from `sk_melee2_test.cpp`, not built; moot until some data grants 179 (Q3 shows none) |
| D2 | Stat 120: game.exe cuts the monster's defense per hit, floor 0, no restore in the function; d2d has no such write | V run (function) / U (lifetime) | needs the caller chain traced and a d2d `Fight` run over several swings; not built |
| D3..D5 | 115 / 116 / 123 / 124: first pass | V run (first pass) | game side only; d2d side not built |

## Bot findings (described, not changed)

- B1 (V read): `internal/d2dsrc/gearstats.go` (lines 5-11 and 34) feeds the melee model only stats 19, 21, 22 and 25. `internal/combat/melee.go` lines 12-18 and 39-40 read stat 119 (to-hit percent) and 325, but the fork's `gear` sums cover stats 16..25 only (docs/forkdbg.md), so **on d2d the model's `ARPct` never sees gear stat 119**, and 115 / 116 / 120 / 123 / 124 / 179 are absent too. For a gear-built character the bot's hit chance is therefore low by the item to-hit percent (stat 119 is not in this card's six, but it is the same path) and ignores the monster-specific terms. Fix would be on the fork's dump (carry these ids) plus a target-aware term in `combat`; neither is done here.
- B2 (I): even with the ids fed, the model has no target-kind input (demon, undead, MonType, boss), so 123 / 124 / 179 / 115 / 116 would need the target's flags; `melee.go`'s `Foe` (line 73) is where that would go. Not read further.

## PR candidate

No PR candidate: the differential here is game-side only; a d2d deviation is proven only with a d2d-side run (mismatches before > 0, after the fix 0), which needs the d2d inputs to exist first, and for 179 no data grants the stat.

## Drill recipe (lab, fork verbs; docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md) - not run

1. 120 lifetime: spawn a plain monster, read its stat 31, hit it once with a weapon carrying a dmg-ac affix (e.g. a unique with -50 from `give` if the fork supports it), read stat 31 again, hit again and watch the to-hit chance (U, D2).
2. 179: no table grants it; confirm by dumping a charm and a unique list with `dump stats` "gear" after extending the id list; expect no 179.
3. B1: `dump stats` on a character with an item carrying 119 (to-hit percent) and 124; check that `gear` shows neither today.

## Matrix-cell updates (proposed text)

docs/research/sorc-skill-matrix.md and the melee row: "AR from gear: stat 179 player-only percent per MonType, no item grants it in 1.14d tables (D); stat 120 lowers the monster's stat 31 per hit (V run); 115 / 116 / 123 / 124 per the first pass; none fed on d2d; the bot lacks stat 119 gear (B1)." docs/coverage/ledger.tsv: no row added: the only fitting row (skills.monster 0) is already claimed by sk-melee2-trace.md, and the F-keys of `formulas` do not name these rules.

## Limits not covered

Monster-attacker path for stats 115 / 116 / 120 (not run); the stat list reader; the MonType group table contents; caller restore of stat 31; no census of real characters; counts are table rows (D), not drop odds. Case counts: A 100,032 (seed 7), B 100,648 (seed 7).

## Shared-file lines for the merger

See openspec/changes/geararmore/merge-lines.txt.
