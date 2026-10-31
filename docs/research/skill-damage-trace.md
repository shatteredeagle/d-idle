# Skill damage traced in game.exe 1.14d: where d2d differs for the Sorceress's attack skills

> status: research, 2026-10-31, agent DECOMPSKILL (card decompskill; riskmap gap rank 4 `skill_damage`). Offline only: no game instance, no d2d slot; nothing posted, pushed or filed on GitHub. Tags: V = read in game.exe (`tools/emu/disasm.py`) or run in the unicorn oracle by this agent (command given); D = read from the patched 1.14d tables (`mpq-cat`); S = upstream's `docs/research/re/*.md`; U = unverified. Does NOT redo PR #135 (the 256ths on a monster's life) nor PR #128 (cast ticks).

**Answer.** The formula that turns a skill level into element damage is game.exe's own in d2d, except for two things, both proven by a differential of game.exe's functions against d2d's `elem_damage` (116,000 cases: 4,549 wrong before, 0 after): (1) the MIN damage's synergy gate: game.exe adds the EDmgSymPerCalc share to the min only when the base min is over one point (or EMinLev1 is set), so every "1-x" lightning skill (Lightning, Chain Lightning, Charged Bolt Sentry, ...) keeps min 1 whatever its synergies; d2d raised it. (2) The synergy and mastery shares are MulDiv (divide-first above 0x100000 256ths): a few 256ths on hits over 4,096 points. For the skills our bot casts at clvl 1-20 (Charged Bolt, Fire Bolt, Ice Bolt, Frost Nova, Static Field) nothing differs in the formula; Lightning / Chain Lightning (the lightning plan from clvl 12) had a min 1.0 -> up to 2.6 points too high on d2d. Found on the way, not proven by a differential: Static Field's three rules (below). Candidate branch `fix/skill-damage` (3 commits) in `~/d2d-lab/pr/skilldmg` on the worker host; PR files: docs/research/pr-skill-damage-{proof,draft,evidence-pack}.md.

## What was already proven (not redone)

Per-level ranges and bolt counts in the tooltip (`skill_tips.py` vs `test_game --skill-tips`, FUN_004edea0, 22,800 lines, docs/research/upstream-gaps/item1-charged-bolt-hit.md); the missile's roll `min + rand(max - min)` in 256ths with the mastery stat (FUN_005a89a0 / FUN_005a8910, `missile_roll.py`); the resistance, pierce, life in 256ths and death (PR #135, `life_apply.py`); the cast length (PR #128). What was NOT covered: the unit path (synergies from the unit's skill levels, masteries, the stat the missile carries), the level brackets above 20, Static Field, cold length.

## The formula as game.exe computes it (V unless marked)

| Address | What it does | Tag |
|---|---|---|
| `FUN_0059f900` -> `FUN_0064b860` | a missile is made with its damage: FUN_0064b860 calls FUN_00647bc0 / 00647d00 (physical), FUN_00644d50 / FUN_00644e40 (element min / max, flag 1 = with mastery) and FUN_00644f20 (length) at the missile's level; `FUN_0064ac60` then sets the missile's own stats with FUN_00627260: 0x15 / 0x16, 0x19, 0x30..0x3b (fire, magic, lightning, cold + length, poison + length), 0x3c..0x42, 0x67 / 0x68 / 0x6a, 0x78..0x7a, 0x8d, 0xb4, 0xc, 0x13b..0x13d, 0x146. No mastery stat (0x149..0x14c) goes onto the missile, so the roll's mastery read (FUN_005a8910's third argument, on the missile) is 0: the mastery is applied once, at creation | V (listings 64b860, 64ac60, 59f900, 5a89a0) |
| `FUN_00644b70` (level, EMinLev1..5) | the brackets: `(lvl-1) x Lev1` to 8; `7 Lev1 + (lvl-8) Lev2` to 16; `+ 8 Lev2 + (lvl-16) Lev3` to 22; `+ 6 Lev3 + (lvl-22) Lev4` to 28; `+ 6 Lev4 + (lvl-28) Lev5` | V, = d2d `level_bonus` |
| `FUN_00644d50` (unit, skill, lvl, flag) MIN | `esi = (EMin + brackets) << HitShift`; **if EDmgSymPerCalc is set AND (esi > 0x100 OR EMinLev1 != 0)** (0x644db3..0x644dc2): `esi += MulDiv(esi, calc, 100)` (FUN_006c0bc0 with the operand table FUN_00646460; the result 0 adds nothing); with flag 1 and a unit: `esi += FUN_00644c90(etype, unit, esi)` | V |
| `FUN_00644e40` MAX | the same without the gate (0x644e9e..0x644eac: calc index, tables loaded, index in range) | V |
| `FUN_00644c90` (etype, unit, dmg) | byte table at 0x644d38: fire 1 -> stat 0x149, lightning 2 -> 0x14a, cold 4 and freeze 12 -> 0x14b, poison 5 -> 0x14c, magic 3 and 6..11 -> none; `MulDiv(dmg, stat, 100)`, 0 when the stat is 0 | V |
| `FUN_00644f20` length | `ELen + ELevLen1 x (lvl-1)` to 8, `7 ELevLen1 + ELevLen2 x (lvl-8)` to 16, then ELevLen3; `+ MulDiv(len, ELenSymPerCalc, 100)` | V, = d2d `elem_length` |
| `FUN_00646c00` skill(id, operand) | the unit's record for `id` (FUN_00643810: the unit's skill list, +0x3c == 0, preferring +0x34 == -1 then the higher +0x28), its level with items (FUN_006442a0 flag 1: +0x28 base, + FUN_00644180's item bonus when +0x34 == -1, capped by FUN_00611830), then FUN_00646460(unit, operand, id, level); operand 41 `blvl` reads FUN_006442a0 flag 0 (base points) | V |
| `FUN_00646460` | the operand table (0x646abc, 0x49 codes): 40 `ulvl` = stat 0xc, 41 `blvl` as above, 38 / 39 / 49 / 50 / 52 / 53 call FUN_00644d50 / 00644e40 (flag 0 / 1), 51 FUN_00644f20, 55..58 calc1..4 | V, = d2d `calc_operand` |
| `FUN_00483360` MulDiv | up to 0x100000 x up to 0x10000: 32-bit product / c; above 0x100000 (c <= a >> 4): `a / c x b` | V (PR #135 proof) |
| `FUN_005c96a0` Static Field (ctx, target) | life = stat 6 >> 8, nothing under 1; with a floor % (params +8: DifficultyLevels' Static Field min, D: 0 / 33 / 50): nothing when life <= max life x floor / 100; dmg = MulDiv(life, calc1 %, 100), at most life - 1, then << 8, at least params +0; **a negative lightning resistance is neutralised**: dmg = MulDiv(dmg, 100, 100 - res) when res < 0 (0x5c9763..0x5c9787), so the later FUN_0057bf80 brings it back to calc1 % of life; the record (FUN_0056c8e0, lightning) goes through FUN_0057c6c0 (flag 1) and FUN_0057cee0 | V (read, not run) |

## d2d's version (file:line at upstream/main f8aab0e8) and the deviations

| Piece | d2d | Differs? |
|---|---|---|
| brackets | `components/rules/skills.hpp:501` `level_bonus` | no (116,000 cases, levels 1..60) |
| min / max | `skills.hpp:544` `elem_damage`: synergy added unconditionally, `damage * calc / 100` | **yes: the min gate** (4,509 of 116,000 lines; 13 skills: Power Strike, Lightning Bolt, Charged Strike, Lightning Strike, Lightning Fury, Lightning, Chain Lightning, Holy Shock, Shock Field, Charged Bolt Sentry, Claws of Thunder, Lightning Sentry, Death Sentry: EMin 1 << 8 or 2 << 7 = 0x100, EMinLev1 0, D) |
| MulDiv | `damage * calc / 100`, `int64 * stat / 100` | **yes above 0x100000** (344 lines; 20 skills at levels 23+ with big synergies / masteries: e.g. Fire Claws, Meteor, Blizzard; at most 0x100 (one point) off) |
| mastery | `elem_damage(.., mastery = true)` with `env.stat(329 + etype)` on the caster, once; `missile_blow` (combat.hpp:340) rolls `range(elo, ehi)` with no second mastery | no (mapping fire / lightning / cold / poison = game.exe's byte table; applied once as game.exe) |
| length | `skills.hpp:556` `elem_length` | no (0 of 116,000) |
| synergy levels | `calc_operand` 41 -> `env.base_level`; `kCalcSkill` -> `env.level` for other operands | no (same split as FUN_00646c00 / handler 41; cases give one level per skill: base = with items, a limit) |
| Static Field | `components/game/fight.cpp:1172`: `resisted(hit_points * pct / 100, res[3])`, whole points, `land()` | **yes, three rules, not in the PR (read only, no differential)**: no difficulty floor (DifficultyLevels), no "at most life - 1", negative lightning resistance raises d2d's damage (game.exe caps it at calc1 %); plus whole points through `resisted` where game.exe keeps 256ths through FUN_0057bf80 (the #135 family) |
| Charged Bolt count | `fight.cpp:2730` `calc1` bolts, `skills.txt` calc1 `min(24, ln12)` (D) | no (tooltip differential, item1) |
| mana cost | `skills.hpp:516` `mana_cost` (FUN_0056c160) | not re-traced here (S; the gap-casts note checked the csv against the formula) |

The differential: `tools/emu/skill_damage.py` (ours, in the branch) runs the real FUN_00644d50 / 00644e40 / 00644f20 (flag 1) on the loaded 1.14d tables, hooking only the stat getter (FUN_00625480: stat 0xc = clvl, 0x149..0x14c = the case's masteries), the unit's skill record lookup (FUN_00643810) and its level (FUN_006442a0); `test_game --skill-damage` runs d2d's `elem_damage` / `elem_length` with the same `CalcEnv`. Cases: every skill with an EType at levels 1..40 x {alone; neighbours (id +-15) at 20; the same with masteries 100; clvl 99 with masteries 37} = 16,000, plus 100,000 random (levels 1..60, clvl 1..99, masteries 0..600, 0..6 other skills at 0..40). Before: 4,549 lines differ; after the gate alone: 344; after the gate and MulDiv: 0 (md5 equal). Upstream's `test_combat` ok, `test_game` rc 0 (27 OK), `pr_guard` 0 FAIL 0 WARN.

```sh
# the worker host, ~/d2d-lab/pr/skilldmg (commits 20441196 tooling, d72ee7f3 gate + doc note, c6be9247 MulDiv)
cd tools/emu && source /tmp/casttrace-venv/bin/activate && export D2_MPQ_DIR=<dir with bin/game.exe and the MPQs> D2_PATCH_INSTALLER=<LODPatch_114d.exe>
python skill_damage.py gen 100000 1 > cases.tsv && python skill_damage.py game cases.tsv > g.tsv      # about 1 min
../../build/tests/test_game --skill-damage cases.tsv | grep -E '^[0-9]+ ' | diff g.tsv -                # empty
```

## Effect on the bot (I, from the formula; no live measurement)

- Blizzard plan clvl 1-25 (Charged Bolt, Fire Bolt, Ice Bolt; Frost Nova on packs; Static Field on bosses): the formula is identical on both backends, so d2d's casts per kill for these skills were right before this work (the #135 fraction was the only engine gap). The remaining difference, if any, is the bolt hit rate and pacing (docs/research/gap-casts-per-kill.md).
- Lightning plan (Lightning from clvl 12, Chain Lightning 18, with Charged Bolt points as synergy): d2d's Lightning min was `1 + 8 % x (CB + CL + Nova points)` points instead of 1; at CB 10 that is 1.8 vs 1.0 on a 1-40 roll: about 2 % more average damage on d2d. Small, but in the direction of d2d killing faster than the original.
- Static Field on d2d: no 33 % / 50 % floor in Nightmare / Hell (the bot's `staticCasts` floor by difficulty already stops casting there) and a monster with negative lightning resistance loses more than 25 % per cast (Normal Act 1: none of the bot's targets have one, D: monstats). Not material at clvl 1-20.

## Not covered (U) and leads

- The cold length / chill application on the hit (FUN_0057c6c0's cold record vs d2d `blow.chill_ticks = resisted(elen, res)`), the freeze duration (FUN_004e6c70) and the slow: the length formula matched (0 of 116,000), its application was not traced (hooked out of PR #135's oracle too).
- Static Field: the three rules above are V by reading, not by a differential; a `static_field.py` oracle of FUN_005c96a0 (hooks: stat getter, max life, FUN_0057c6c0 as PR #135's) is the next candidate; the bot's own `staticCasts` rule is unaffected.
- Item bonuses to skill levels (+skills): one level per skill in the cases (base = with items); game.exe's `blvl` (flag 0) vs `lvl` (flag 1) split is read (FUN_006442a0), d2d's `base_level` / `level` split matches by reading, not by a case where they differ.
- The physical part (FUN_00647bc0, `skill_phys`) and the weapon share (SrcDam): not in this trace (the staff melee fallback is the plain attack: monster-attack-trace.md's `0057b420` family on the player's side, not traced for the player).
- Mana cost: formula only (S); FUN_0056c160 not re-read.
