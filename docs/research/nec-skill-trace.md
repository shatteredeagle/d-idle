# Necromancer skills: Raise Skeleton / Skeletal Mage, Corpse Explosion, Teeth, Bone Spear / Spirit, the curses, Poison Dagger / Nova; game.exe 1.14d vs d2d vs the bot (card necskill, coverage gap skills.nec)

> status: card necskill, 2026-11-01, offline (no game, no d2d slot, no Go change). First trace of the Necromancer rows (docs/coverage/rule-coverage.md: skills.nec 0 of 30). Reuses the method of SKMISS / SKMANA / PALSKILL (docs/research/sk-missile-trace.md, sk-mana-trace.md, pal-skill-trace.md): game.exe's own code under upstream's emulator against the real d2d build (upstream main 72132b99, patched 1.14d tables).

Tags: **V run** run natively in game.exe under the emulator by this card and compared line by line with the real d2d build (scripts named; counts given); **V read** read in the disassembly (`tools/emu/nec_dis.sh`, upstream's `disasm.py`; addresses given), NOT offered as proof of anything (model split of 2026-11-01); **D** from the patched 1.14d tables (`Skills.txt`, `Missiles.txt`, `States.txt` of the patch stack); **S** d2d source (named file:line) or an earlier trace of ours; **I** inference; **U** not verified. Dead ends are listed in section 8.

## Summary

- **Every calc cell of the 30 Necromancer rows equals d2d (V run, 0 differ of 27,800 cases).** `calc1..4`, `petmax`, `sumsk1..5`, `delay`, the aura range / length / period, `aurastatcalc1..6` and `passivecalc1..5` of Skills 66..95, evaluated by game.exe's calc machine (FUN_00646ca0) and by d2d's `eval_calc`: levels 1..40 alone and with every neighbour at 20 (clvl 30 and 99, masteries 0 and 50) plus 20,000 random cases (seed 7). So the numbers the plan relies on (summon counts, curse lengths and radii, Corpse Explosion's percentages, Bone Armor's absorb, the golem and skeleton stat shares) are the same on both sides; whether the *effects* around them match is the rest of this doc.
- **Teeth, Bone Spear and Bone Spirit: the SKMISS roll deviation applies unchanged (V run, 36,600 cases: 10,509 differ on the real d2d build, 0 differ with the SKMISS fix `23e2baec`).** The magic missile's damage min / max is equal on all 36,600 (game.exe FUN_00644d50 / e40 vs d2d `missile_damage`); the life after one hit differs wherever d2d's `Rng::range(min, max)` (top inclusive, modulus max - min + 1) draws a different value than game.exe's `min + rand(max - min)`. The existing SKMISS candidate already fixes it; no new PR (section 6).
- **Corpse Explosion (V read, partly I).** game.exe (FUN_005c4df0, the do-55) takes the corpse's life from the monster's **base row average** ((min + max) / 2 of the row's life range at its level), rolls `calc1..calc2` percent (70..120, constant at every skill level; only the radius grows) on the **corpse's own seed** exclusive of the top, **scales the damage down by caster level / corpse level when the corpse is the higher level**, and puts half of it in fire (calc3 = 50). d2d uses the corpse's actual hit points, a top-inclusive roll, no level scaling. Three deviations, none with a differential today (D3 to D5); a differential can be built (section 5).
- **Summons (V read the do, V run the calcs; U for the base stats).** Raise Skeleton and Skeletal Mage (do 31, FUN_005c4b00) take a corpse, spawn the row through FUN_0056d940 with `petmax` (a calc) passed as the cap, then set level (FUN_005c49e0: min(owner level, owner level x 3 / 4)), the skill's passive and aura stats (FUN_005c4470) and the sumskills. Counts by level (V run): **petmax = level up to 3, then 2 + level / 3: 1, 2, 3, 3, 5 (L10), 8 (L20), 12 (L30)**. d2d implements the cap by dropping the oldest of the same `pettype`. d2d takes the pet's base life and damage from the raw MonStats columns (its source says so, summons.cpp:147): **U**, no run of the game side.
- **Curses (S, V run for the values).** Amplify Damage -100 % physical resist (length 200 ticks at L1, 875 at L10, 1625 at L20; radius 3 / 12 / 22), Decrepify -50 % on four stats, Lower Resist -31 / -55 / -62 on four resists at L1 / 10 / 20, Weaken -33 % damage, Iron Maiden returns 200 / 425 / 675 percent, Life Tap 50 percent: the values are V run equal to d2d. The 1/5 curse-pierce rule and the curse on an immune are docs/research/sk-resist-trace.md (D1 there, a PR candidate with its own proof); not repeated.
- **Deviations d2d vs game.exe: D1 magic roll (V run, differential yes, fix exists), D2 Teeth and Poison Nova NextHit (S + I, differential buildable), D3 to D6 Corpse Explosion (V read, differential buildable), D7 burn of the fire half (U).** **No new PR candidate:** D1 is covered by the SKMISS branch; D2 to D7 have no run. Bot: no Necromancer combat code exists (B1 to B4, plan level).
- **Coverage ledger:** `skills.nec` 30 rows (27 TESTED-0-DIFF on the calc aspect, 3 TESTED-DEVIATION on the roll) and the missiles 114, 192, 193 (section 7).

## 1. How it was run

| Piece | What | Tag |
|---|---|---|
| `tools/emu/nec_calc.py` + `nec_test.cpp --calc` | game.exe's FUN_00646ca0 on every calc pointer of Skills 66..95 (record offsets calc1..4 +0x138..0x144, petmax +0xc0, sumsk +0xd0 gated by sumskill ids +0xc4, delay +0x190, aurarange +0x64, auralen +0x60, perdelay +0x128, aurastat +0x68 gated by ids +0x54, passive +0xa4 gated by ids +0x98) vs d2d `eval_calc` on the loaded tables; stat getter hooked as in skill_damage.py (stat 12 = clvl, 0x149..0x14c masteries, else 0), skill levels by id | V run |
| `tools/emu/nec_missile_hit.py` + `nec_missile_test.cpp --missile-hit` | sk_missile_hit.py with the magic element: stats 0x34 / 0x35, resist 0x25, record field +0x20 (as sk_resist.py's ELEM 1); FUN_005a89a0 rolls on the missile seed, FUN_0057c6c0 applies; d2d side `missile_damage` + `missile_blow` + `hurt` | V run |
| build and run | `tools/emu/nec_build.sh` (hand build against the skmana scratch build, 72132b99), `nec_run.sh N SEED`, `nec_run_missile.sh N SEED`; the "after" binary is the same test linked against the skmiss scratch build (fix `23e2baec`): `SRC=~/d2d-lab/pr/skmiss OUT=... nec_build.sh ~/d2d-lab/pr/skmiss-out/build` | V run |

Runs: `nec_run.sh 3000 1` = 5,400 cases, DIFF 0; `nec_run.sh 20000 7` = 22,400 cases, DIFF 0. `nec_run_missile.sh 20000 1` = 23,300 cases, d2d 6,794 differ (min / max column 0 differ, life column 6,794); `nec_run_missile.sh 10000 7` = 13,300 cases, 3,715 differ. After binary: 0 and 0. A first d2d-side run of the calc test showed 185 lines differing, all Fire Golem's `sumskill1 = holy fire`: the harness's skill lookup was case-sensitive, d2d's `Fight::skill_named` is not (summons.cpp:79); the harness was fixed to the same lookup (a harness bug, not a deviation). A second harness bug (an out-of-range `pierces[4]` for magic, undefined behaviour) crashed the d2d-side missile test until fixed.

## 2. Questions and answers

**Q1. Raise Skeleton: how many, with what stats, by level? (D + V run for the numbers)** Skills 70: do 31, summon `necroskeleton`, pettype skeleton, TargetCorpse, mana 6 + (L - 1) (SKMANA: 0 diff). By skill level L (V run, no synergy, clvl 30):

| L | petmax | life bonus (calc1, percent) | aura damagepercent | tohit (aura2) | armorclass (aura3) | passive maxhp (p1, 256ths) | passive normaldamage (p2) |
|---|---|---|---|---|---|---|---|
| 1 | 1 | 0 | 0 | 15 | 15 | 0 | 0 |
| 3 | 3 | 0 | 0 | 45 | 45 | 0 | 0 |
| 5 | 3 | 100 | 14 | 75 | 75 | 0 | 0 |
| 10 | 5 | 350 | 49 | 150 | 150 | 0 | 2 (Skeleton Mastery level 0 + edmn) |
| 20 | 8 | 850 | 119 | 300 | 300 | 0 | 16 |
| 30 | 12 | 1350 | 189 | 450 | 450 | 0 | 46 |

(`aura damagepercent` is `(L - 3) x 7` from L4; `calc1` is `50 x (L - 3)`; at Skeleton Mastery level M the passive adds `M x 8 x 256` max life and `M x 2 + edmn` damage, V run equal to d2d on the calc, section 1.) The base life and damage of the skeleton itself (MonStats `necroskeleton`) are not measured on the game side: **U**.

**Q2. Skeletal Mage?** Skills 80: do 31, summon `necromage`, pettype skeletonmage, sumskill1 `NecromageMissile`; petmax as Raise Skeleton; calc1 7 per level above 3 (7, 49, 119 at 4, 10, 20); armorclass aura `(L + Skeleton Mastery) x 10`; the mage's bolt level `sumsk1` = 1 (L4), 4 (L10), 9 (L20) (V run). Damage of the bolt: not run (the missile row `necromagemissile` and its element come from the sumskill; U).

**Q3. Corpse Explosion: what does it do? (V read, FUN_005c4df0 + callback FUN_005c4d10)** Walk of the code:
1. 0x5c4e02: target unit = FUN_00553540; must be a corpse (FUN_00645680: unit type 1, mode 12, not already used FUN_0063a770); state 0x68 is set on it (FUN_00639db0), so it cannot be exploded twice.
2. 0x5c4e6c..0x5c4ed6: base life `ebx` = stat 7 (max life). For a monster (type 1) it is **replaced**: FUN_006538a0(monster data, flag, byte +0x6d, level stat 12) fills a 0x38-byte range record and `ebx = (rec[0] + rec[1]) << 7` = **(min + max) / 2 of the life range, in 256ths** (the base-row average at that level and difficulty, not the corpse's actual life).
3. 0x5c4ef4..0x5c4f68: `range = calc ln34 (+0x64)`, `calc1` (+0x138) and `calc2` (+0x13c) evaluated with the skill (FUN_00646ca0, level); `lo = MulDiv(ebx, calc1, 100)`, `hi = MulDiv(ebx, calc2, 100)`; the damage = `lo + rand(hi - lo)` through FUN_0045c3e0 on `unit + 0x20`, the **corpse's own seed**, top exclusive.
4. 0x5c4f77..0x5c4f9a: if the corpse's level (stat 12) is non-zero and above the caster's, `damage = MulDiv(damage, caster level, corpse level)`.
5. 0x5c4fc7..0x5c503a: `calc3` (+0x140, 50 here) is clamped to 0..100; with EType (fire) set the fire part is `MulDiv(damage, calc3, 100)` with the element length of FUN_00644f20 (burn), filled into the 0x70-byte damage record by FUN_0056c8e0, and the rest `MulDiv(damage, 100 - calc3, 100)` stays physical.
6. 0x5c5089: FUN_0056b7e0 area apply, search radius `(range + 1) / 2`, callback FUN_005c4d10: per unit it copies the record, **zeroes the physical field (record +8) when FUN_006492a0 puts the unit beyond the squared bound `(range / 2)^2`**, then FUN_0057c6c0 (the same apply as a missile hit: resistances, life) and FUN_0057cee0.
Table (D, V run): `calc1 = 70`, `calc2 = 120`, `calc3 = 50` at every level 1..30; `aurarange` (ln34) 8 at L1, 17 at L10, 27 at L20, 37 at L30; mana 15 + (L - 1) (FUN_0056c160 / SKMANA). **So damage does not grow with the skill level: only the radius does** (and the count of corpses it can use at once per cast stays one). V run for the table, V read for the rest.

**Q4. Teeth, Bone Spear, Bone Spirit? (D, V run, V read)** Teeth (67): do 8 spreads `calc1 = min(ln12, 24)` teeth (V run: 2, 6, 11, 21, 24 at L1, 5, 10, 20, 30), missile `teeth` 114 (Vel 16, Range 50, **NextHit 1, NextDelay 4**, LastCollide, CollideKill). Bone Spear (84): missile 192 (Vel 24, Range 40, LastCollide 1, no CollideKill: it passes through), Bone Spirit (93, do 10): missile 193 (Vel 12, Range 128, hit func 10, do 7, CollideKill). All three are magic (EType 4): the damage min / max is equal (36,600 cases) and the only deviation found is the roll (D1). Resistance: magic, no pierce stat (FUN_0057bf80, S sk-resist-trace Q5).

**Q5. The curses? (V run values; effects S)** Skills 66 (Amplify Damage), 72 (Weaken), 76 (Iron Maiden), 82 (Life Tap), 87 (Decrepify), 91 (Lower Resist): all do 30 (curse), radius `ln12` (subtiles), length `ln34` ticks:

| Curse | L1 | L10 | L20 | stat(s) |
|---|---|---|---|---|
| Amplify Damage (66) | r 3, len 200 | r 12, 875 | r 22, 1625 | damageresist -100 |
| Weaken (72) | r 9, 350 | r 18, 890 | r 28, 1490 | damagepercent -33 |
| Iron Maiden (76) | r 7, 300, calc1 200 | 840, 425 | 1440, 675 | calc1 = ln56 percent of the damage it deals (calc2 / calc3 = a quarter) |
| Life Tap (82) | r 4, 400, calc1 50 | r 13, 940 | r 23, 1540 | calc1 = 50 percent of the melee damage dealt to it heals the attacker |
| Decrepify (87) | r 6, 100 | 235 | 385 | velocity, damagepercent, damageresist, attackrate each -50 |
| Lower Resist (91) | r 7, 500, -31 | r 16, 950, -55 | r 26, 1450, -62 | fire, lightning, cold, poison resist |

d2d side (S): do 30 in `Fight::spot` (strike.cpp:785..808: the auratargetstate on every monster within the radius for `auralen` frames, one curse at a time per monster slot); stats applied through `apply_target_stats` (missiles.cpp:813: resists 36..45, armorclass) and `monster_states` (states.cpp:259: damagepercent 25, velocity 67, attackrate 68 / 69, Iron Maiden's reflect, Terror); Life Tap `strike.cpp:456`. d2d shortens Dim Vision, Terror, Confuse and Attract by the AiCurseDivisor; not the others. The 1/5 pierce on an immune and the base-resistance rules: sk-resist-trace.md (D1 there). Iron Maiden in d2d is reflected only on a cursed monster's melee hit on the player (ai.cpp:2701); game.exe's rule (an event on the cursed unit's melee damage, V not read here) is **U**.

**Q6. Poison Dagger, Poison Nova? (D, S)** Damage with synergy and length: skillfx.damage is already TESTED (116,000 cases, docs/research/skill-damage-trace.md, includes both). Poison Nova is the do-22 nova with NextHit 1 / NextDelay 4 rows (missile 118, Vel 12, Range 30): docs/research/sk-nova-trace.md D1 and PR #158 (Nova, Frost Nova compared only; docs/research/pr-nexthit-v2-addendum.md says the other NextHit rows are not). Poison Dagger (73) is a melee do 32 with `srvstfunc 16`: not read here (U). The poison hit path (per-tick poison from the roll and length) was not run with the Necromancer rows: U.

**Q7. Bone Wall, Bone Prison? (D, S)** Bone Wall (78): do 60, summon `bonewall`, makers `bonewallmaker` (missile 207, do 13, Range 7); `calc1 = 25 x (L - 1) + (Bone Armor + Bone Prison) x 10` (225 at L10, 475 at L20), `calc2 = 8`. d2d lays `max(calc2, 2) / 2` walls each way a subtile apart at once and holds them 24 s ("ponytail: the published time", summons.cpp:218): the maker's own flight and the wall's life (calc1 as hit points?) not read: **U**. Bone Prison (88): do 62, twelve in a ring (d2d: summons.cpp). Not run.

## 3. d2d side (S, read at upstream main 72132b99)

`Fight::cast_summon / summon / summon_one` (components/game/fight/summons.cpp:52..258): `corpse_near` picks the nearest unused DD corpse within 2 (the bot's cast point), marks it used; the cap loop drops the oldest same-`pettype` pet while the count is at `petmax` (:128..130); pet level `min(clvl, clvl x 3 / 4 + bonus)`, bonus 0 for the Necromancer (:133); life, damage, to-hit, armour from the MonStats columns "straight" ("FUN_005b2f20's pet branch isn't traced", :147); passive and aura stats of the skill added with the owner's calcs (:171..191: maxhp >> 8, normaldamage, damagepercent, tohit, armorclass, thorns, resists); calc1 percent on life (:192); sumskills as pet missile / aura (:196..199); Revive keeps the corpse's own monster at calc2 frames (:206); Bone Wall / Prison hold 24 s (:218). Corpse Explosion: `Fight::corpse_blast` (strike.cpp:634): corpse HP x `Rng::range(calc1, calc2)` / 100, radius `aurarange / 2`, half physical, half fire, both through `resisted`, a sure hit; no level scaling, no burn.

## 4. Deviations d2d vs game.exe

| id | rule in game.exe | rule in d2d | tag | differential |
|---|---|---|---|---|
| D1 | magic missile roll `min + rand(max - min)`, top exclusive (FUN_005a89a0) | `Rng::range(min, max)`: top inclusive, modulus + 1 (combat.hpp:420) | **V run** (36,600 cases: 10,509 differ, 0 after `23e2baec`) | yes: `nec_run_missile.sh`; the SKMISS fix |
| D2 | Teeth and Poison Nova rows have NextHit 1 / NextDelay 4: the first missile to hit puts unit state 0x56 `justhit` and the others pass the unit for 4 frames (sk-nova-trace D1, PR #158 proved for Nova / Frost Nova) | per-missile `hit_at` list: every tooth strikes the monster once, again after NextDelay (missiles.cpp:587..596, before #158) | S (#158 proof) + I (the Teeth fan was not compared) | buildable: `test_verify_nexthit.cpp` / `verify_nexthit.py` with a Teeth fan (do 8: count `calc1`, spread by the fan table) instead of the nova ring; oracle FUN_005adcc0 chain on the monster's justhit state; inputs: monster distance 1..8, tooth count 2..24 |
| D3 | CE base life = (min + max) / 2 of the row's life range (FUN_006538a0, 0x5c4ec8..0x5c4ed6) | the corpse's actual `stats.hit_points` (strike.cpp:638) | V read | buildable: run FUN_005c4df0 on a unit made by upstream's `monsters.py` spawn (umod_spawn.py shows how the life range is read), FUN_00553540 / FUN_00645680 hooked, FUN_0056b7e0 replaced by a direct call of the callback; d2d side: a `Fight` harness (`pets_merc_harness.cpp` shows one) calling `corpse_blast`; inputs: monster ids x levels x difficulty x champion / unique flags; oracle: damage dealt to one target at distance 1 |
| D4 | damage x caster level / corpse level when the corpse level is higher (0x5c4f77..0x5c4f9a) | none | V read | same harness as D3 (corpse level above and below the caster's) |
| D5 | roll on the corpse's seed, top exclusive | `range(low, high)` top inclusive on the Fight's rng | V read (the primitive is V run in D1) | same harness; SKMISS-style fix in `corpse_blast` |
| D6 | physical half zeroed beyond `(range / 2)^2`, search to `(range + 1) / 2` (callback FUN_005c4d10) | one radius `range / 2` for both halves | V read (partial: the callback's argument layout was not fully mapped) | same harness, targets at distance 1..20 subtiles |
| D7 | fire half carries the element length (FUN_00644f20): a burn | `blow.damage` only, no burn | U (the apply of the record's fire length was not read) | same harness, read the target's burn state |

Not a deviation but open: the pet base stats (section 3) and Skeleton Mastery's share (FUN_005d6b60, "not traced" in upstream's skills.md): the passive stats are applied in d2d and equal by calc; the unread part is the game's own base life / damage of the pet row. Closing it needs a spawn of `necroskeleton` through FUN_0056d940 (flags 0x42) with an owner unit, then FUN_005c49e0 / FUN_005c4470; not attempted (dead end 1).

## 5. Bot findings (described, not changed)

| id | finding | where |
|---|---|---|
| B1 | The Necromancer plans have no combat code behind them: there is no cast, curse, summon census or corpse reader (grep of internal/ and cmd/ finds only plan JSON, gear, merc policy). The only summon-related code is friendly recognition by the monstats AI column (`internal/bot/catalog.go:35`, `internal/bot/yield.go:14`). The plan rows and hotkeys (`internal/plandata/build-necromancer_leveling_normal30.json`, `necromancer_summoner_endgame90.json`) depend on behaviour the bot cannot yet perform. | docs/classes/necromancer.md section 8 (the same list) |
| B2 | The summoner plan's main attack from clvl 6 to 90 is Corpse Explosion (`necromancer_summoner_endgame90.json`, `main_attack[1]`). Per Q3 its damage per corpse does **not** rise with the skill level (70 to 120 percent of the corpse's base life, half fire), it needs a corpse, it is cut by level when the corpse is above the character, and the fire half is nil on a fire immune. Points put into it past the radius add no damage. | `main_attack`, `rows` |
| B3 | The leveling plan keeps Teeth until clvl 17 then Bone Spear (`necromancer_leveling_normal30.json`, `main_attack[0..1]`). Teeth's tooth count is `skill level + 1` (up to 24) but NextHit (D2) means a close monster is struck by fewer teeth on game.exe than on d2d: a d2d soak over-credits Teeth's single-target damage (I; magnitude not measured). | `main_attack` |
| B4 | The summoner plan holds Raise Skeleton at one point per clvl from 2 to 6 and later; the count cap is `2 + L / 3` and the life bonus starts at L4 (Q1), so the first three points only add skeletons (1, 2, 3) and tohit / AC; the first life and damage bonus point is the fourth. If the soak's upkeep rule is "recast to the cap", the cap is the `petmax` column above, not a constant. | `rows` clvl 2..6 |

## 6. PR candidates

**No new PR candidate.** D1 is the SKMISS candidate (`fix/missile-roll-top`, `23e2baec`, local, see docs/research/sk-missile-trace.md and pr-skill-damage-evidence-pack.md for the pack): this card adds its Necromancer evidence (36,600 cases, before 10,509 differ, after 0, harness `nec_missile_hit.py`, the same differential extended to the magic element) for the pack's "other skills" line; no branch was made or changed. D2 is the posted #158 generalised to more rows (needs its own run). D3 to D7 have no differential yet, so under the 2026-11-01 rule they are bot-side findings and open questions, not PR text.

## 7. Matrix cell updates (proposed text; docs/research/sorc-skill-matrix.md is Sorceress only, so a Necromancer table is proposed here)

| Row | Cell | New value |
|---|---|---|
| all 30 Necromancer skills | calc cells (counts, lengths, radii, percents, stat shares) | V run equal to d2d, 27,800 cases (`nec_run.sh`) |
| Teeth, Bone Spear, Bone Spirit | damage min / max | V run equal (skillfx.damage, 116,000; and 36,600 here) |
| Teeth, Bone Spear, Bone Spirit | roll and life after the hit | D1 V run, fixed by the SKMISS candidate |
| Teeth, Poison Nova | hits per monster | D2 S + I |
| Raise Skeleton, Skeletal Mage | count by level | V run (calc), V read (cap enforcement) |
| Raise Skeleton, Skeletal Mage | the pet's base life, damage, to-hit | U (d2d reads MonStats columns; game side not run) |
| Corpse Explosion | base, roll, level cap, split, radius | D3..D7 V read, no run |
| Amplify Damage, Weaken, Iron Maiden, Life Tap, Decrepify, Lower Resist | values by level | V run equal; effects S |
| Poison Dagger | melee hit and poison | U |
| Bone Wall, Bone Prison | makers, hold time | U |

Ledger lines (docs/coverage/ledger.tsv): `skills.nec 66,68-72,74-83,85-92,94,95 TESTED-0-DIFF 27,800 cases (calc aspect)`; `skills.nec 67,84,93 TESTED-DEVIATION 36,600 cases (D1)`; `missiles 114,192,193 TESTED-DEVIATION 36,600 cases (D1)`. Note the 27,800 run covers rows 67, 84, 93 as well; they take the stronger status for their own line.

## 8. Dead ends and limits

1. **Pet base stats not measured on game.exe.** A spawn of `necroskeleton` through FUN_0056d940 needs a full owner and room (the pets harness `pets_merc.py` models only the target search). Not attempted in this card; the next step is `umod_spawn.py`-style room population with a player unit, then FUN_005c49e0 and FUN_005c4470, and the d2d side a `Fight` harness for `summon_one`. Until then the pet's life and damage are U on both sides' agreement.
2. **FUN_005c2ff0** (called from the do 31 before the spawn: it builds a 0x70-byte record with flags 0x1000, sets a field to `stat 7 / 8` of the caster and applies it with FUN_0057c6c0) was read but not understood (it only acts when the caster's unit type is 0 and its class id +4 is 3). Not used in any claim above; it may be a cost or a guard. U.
3. **The CE area callback** FUN_005c4d10: the argument layout of its context (record pointer, position, squared bound) was inferred from the push order in FUN_005c4df0, not confirmed by a run; D6 stays V read, partial.
4. **Poison**: Poison Dagger / Poison Nova hit and per-tick damage (the poison length and per-tick roll) were not run. The existing skillfx.damage covers min / max / length only.
5. **Bone Wall / Prison**: makers and hold time are d2d's published constants (24 s), not game.exe reads.
6. **The calc differential does not cover** stats the getter returns 0 for (any calc reading an item stat); the Necromancer rows read only skill levels, masteries and clvl, so the limit does not bite here (V: the run covers every cell, none read another stat).
7. Merc and party interactions (revive of a merc corpse, party CE) not examined.

## 9. Drill recipe (lab, fork verbs; for later, not run here)

Fork verbs (docs/forkdbg.md): `debug skill`, `debug points`, `debug spawn`, `debug mon set <id> life .. maxlife .. res <type> ..`, `debug cast <skill> unit <id>|<x> <y>`, `debug missiles`, `debug states`, `step`, `dump stats`. A throwaway Necromancer on one slot, `--no-save`, lockstep, from docs/forkcheats.md.

| Drill | Confirms | How |
|---|---|---|
| N1 Raise Skeleton count | Q1 cap by level | `debug points 70 1/4/10/20`; kill monsters near, cast on each corpse, `monsters` list friendly units: 1, 3, 5, 8; the cap's replacement of the oldest |
| N2 Skeleton base stats | the U in Q1 and section 3 | read `dump stats` of the pet after the cast (life, damage); compare with game.exe only through a later emulator run (dead end 1) |
| N3 CE base life (D3) | D3, D4 | `debug mon set` the corpse monster's life to 1 and 1000 before it dies (so its actual life differs from the row average), explode: the damage must not follow the actual life on game.exe, follows it on d2d; repeat with a corpse above the clvl (a higher-level monster) for D4 |
| N4 CE radius (D6) | D6 | monsters at 5, 8, 10 subtiles of the corpse at skill level 1 (range 8): which are hurt by the physical half (read life lost against a physical-immune target to isolate the fire half) |
| N5 Teeth hits (D2) | D2 | one monster at distance 1, 3, 6, `debug mon set` life large, `debug cast 67`, life lost per cast at level 1 (2 teeth), 10 (11), 20 (21); on game.exe it should not scale with the tooth count |
| N6 Magic roll (D1) | D1 | as sk-missile-trace M4 with Teeth level 1 (4 to 8 in 256ths) |
| N7 Curse durations | Q5 | cast each curse at level 1 on a monster, `debug states`: length 200 / 350 / 300 / 400 / 100 / 500 ticks |

## Shared-file lines for the merger

See `openspec/changes/necskill/merge-lines.txt`.
