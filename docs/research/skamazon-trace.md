# Amazon skills: Jab, Strafe, Fend, Multiple Shot, the arrow and javelin skills, the passives, Decoy and Valkyrie; game.exe 1.14d vs d2d vs the bot (card skamazon, coverage gap skills.ama)

> status: card skamazon, 2026-11-02, offline (no game, no d2d slot, no Go change). First trace of the Amazon rows (docs/coverage/rule-coverage.md: skills.ama 0 of 30). Method of docs/research/nec-skill-trace.md and sk-missile-trace.md: game.exe's own code under upstream's emulator against the real d2d build (the same two scratch builds as those traces: upstream main 72132b99 and the SKMISS fix `23e2baec`; patched 1.14d tables).

Tags: **V run** run natively in game.exe under the emulator by this card and compared line by line with the real d2d build, or (where said) run on the game side only (scripts named; counts given); **V read** read in the disassembly (addresses given), NOT offered as proof of anything (model split of 2026-11-01); **D** from the patched 1.14d tables (`Skills.txt`, `Missiles.txt`, `ItemStatCost.txt` of the patch stack); **S** d2d source (file:line) or an earlier trace of ours / upstream's `docs/research/re/skills.md`; **I** inference; **U** not verified. Dead ends are in section 8.

## Summary

- **Every calc cell of the 30 Amazon rows equals d2d (V run, 0 differ of 27,800 cases).** `calc1..4`, `petmax`, `sumsk1..5`, `delay`, the aura range / length / period, `aurastatcalc1..6` and `passivecalc1..5` of Skills 6..35 (the level-based part of every skill, including Multiple Shot's and Strafe's arrow counts, Fend's hit cap, the five passives, Decoy and Valkyrie): levels 1..40 alone and with every neighbour at 20 (clvl 30 and 99, masteries 0 and 50), plus random cases (5,400 at seed 1, 22,400 at seed 7).
- **The elemental arrow and javelin skills carry the SKMISS roll deviation unchanged (V run, 118,000 cases: before the fix 13,353 of 64,000 (seed 1) and 10,228 of 54,000 (seed 7) differ in the monster's life after one hit, 0 differ after `23e2baec`).** Fire, Exploding, Immolation, Cold, Ice, Freezing Arrow, Lightning Bolt and Lightning Fury: the element's min / max is equal on all 118,000 (game.exe FUN_00644d50 / e40 against d2d `missile_damage`); only the roll differs. The existing SKMISS candidate already fixes it; no new PR.
- **A real d2d gap: the Pierce passive (and the item's piercing attack) do nothing in d2d (D2; V run on the game side, S on the d2d side, no d2d run).** game.exe's FUN_0059f940 (run when a missile is made) turns the shooter's `skill_pierce` (stat 166, written by the Pierce passive) plus `item_pierce` (stat 156) into the number of pierces the missile may make (0..4, stored as the missile's stat 328); the collision code (0x5ada60) spends one per unit. d2d reads only stat 328 of the shooter as a percentage (fight.cpp:2971), and nothing in d2d writes it from a skill: the Pierce passive never acts. The game-side function was run on 6,776 cases (section 2, Q6). A d2d-side harness does not exist, so this is a finding, not a PR candidate.
- **Bot: five findings, one of them a wrong class (B1..B5).** `classOptions` gives the Paladin Charged Strike, an Amazon skill (D: charclass ama), and omits Lightning Bolt, the javelin plan's main attack (internal/combat/rules.go:238, :244). The Pierce points in the multishot plan are worth something only in game.exe, not in d2d (B3).
- **Deviation count:** D1 magic-roll family (V run, differential yes, fix exists), D2 Pierce (V run game side + S), D3 Strafe timing (S), D4 merc Jab hits as one (S), D5 Magic Arrow conversion unread (U). **No new PR candidate.**
- **Coverage ledger:** `skills.ama` 30 rows (22 TESTED-0-DIFF on the calc aspect, 8 TESTED-DEVIATION on the roll) and the eight arrow missiles (section 7).

## 1. How it was run

| Piece | What | Tag |
|---|---|---|
| `tools/emu/ama_calc.py` + `nec_test.cpp --calc` | game.exe's FUN_00646ca0 on every calc pointer of Skills 6..35 (record offsets as nec-skill-trace section 1: calc1..4 +0x138.., petmax +0xc0, sumsk +0xd0 gated by the ids +0xc4, delay +0x190, aurarange +0x64, auralen +0x60, perdelay +0x128, aurastat +0x68 gated by +0x54, passive +0xa4 gated by +0x98) vs d2d `eval_calc` on the loaded tables; stat getter hooked as in skill_damage.py (stat 12 = clvl, 0x149..0x14c masteries, else 0) | V run |
| `tools/emu/ama_missile_hit.py` + `sk_missile_hit_test.cpp` | `sk_missile_hit.py` (card SKMISS) with the Amazon skill table: Fire / Exploding / Immolation Arrow (7, 16, 27), Cold / Ice / Freezing Arrow (11, 21, 31), Lightning Bolt (20), Lightning Fury (35); FUN_005a89a0 rolls on the missile seed, FUN_0057c6c0 applies; d2d side `missile_damage` + `missile_blow` + `hurt`. Only the ELEMENT part of the hit; the weapon's physical part (SrcDam) is not run | V run |
| `tools/emu/ama_pierce.py` | game.exe FUN_0059f940 (V read first, section 2 Q6) called on a fake shooter and a fake missile unit, stat getters FUN_00625480 / 006253b0 / 00625500 and setter FUN_00627260 hooked to one dict, FUN_0046ace0 to a row with all bits set; FUN_00650e40 and the rolls run for real | V run (game side only) |
| build and run | `tools/emu/ama_build.sh` (hand build of `nec_test.cpp` and `sk_missile_hit_test.cpp` against the skmana scratch build = before, and against the skmiss scratch build = after the fix), `ama_run.sh calc N SEED`, `ama_run.sh hit N SEED`, `ama_mm.sh SEED` (min / max columns only), `ama_pierce_run.sh`, `ama_asm.sh` (full linear disassembly into the scratch dir, grep and `at ADDR`) | V run |

Runs: `ama_run.sh calc 3000 1` = 5,400 cases, DIFF 0; `ama_run.sh calc 20000 7` = 22,400 cases, DIFF 0. `ama_run.sh hit 20000 1` = 64,000 cases: before the fix 13,353 lines differ, after 0; `ama_run.sh hit 10000 7` = 54,000 cases: 10,228 differ, 0 after. `ama_mm.sh 1` and `ama_mm.sh 7`: the min / max columns differ on 0 lines in both. The harness's own counter "d2d's roll wrong on 63,981" counts every case where a copy of d2d's roll (modulus max - min + 1) draws a different number than game.exe's, even when the life afterwards is equal (a small life or a full resistance hides it); the honest figure for "different result" is the life column above. Cases per skill at seed 1: 7,950..8,047 each of the eight skills (seed 7: 6,684..6,821). `ama_pierce_run.sh`: 6,776 game-side cases.

## 2. Questions and answers

**Q1. The start and do functions of the 30 rows (D, from Skills.txt; names S upstream's re/skills.md).** `srvstfunc / srvdofunc / missile / element`:

| Skill (id) | st / do | srvmissile | EType | note |
|---|---|---|---|---|
| Magic Arrow (6) | none / none | magicarrow (27) | none (the missile row converts: EType mag, DmgCalc1 `dl12`, base 1, per level 1) | SrcDam 128, flat 1..1 |
| Fire Arrow (7) | 4 / none | firearrow (12) | fire 1..4 | SrcDam 128 |
| Inner Sight (8) | none / 6 | - | - | aura state `innersight`: armorclass, radius, length |
| Critical Strike (9), Dodge (13), Avoid (18), Penetrate (23), Evade (29), Pierce (33) | passive | - | - | passive stat by calc (Q5) |
| Jab (10) | 5 / 7 | - | none | SrcDam 128, sequence skill (hits per sequence from the weapon) |
| Cold Arrow (11) | 4 / none | coldarrow (40) | cold 6..8, length 100 | SrcDam 128 |
| Multiple Shot (12) | 4 / 8 | srvmissilea `multipleshotarrow` (213), srvmissileb `multipleshotbolt` (214) | none | SrcDam 96; do 8 = the fan |
| Power Strike (14) | 6 / 2 | - | lightning 1..16 | melee, SrcDam 128 |
| Poison Javelin (15) | 4 / none | poisonjav (38) | poison 32..48, length 200 | SrcDam 128 |
| Exploding Arrow (16) | 4 / none | explodingarrow (41) | fire 2..6 | SrcDam 128 |
| Slow Missiles (17) | none / 6 | - | - | aura state, `skill_handofathena` 33 (constant) |
| Impale (19) | 7 / 2 | - | none | melee, SrcDam 128, tohit 100 |
| Lightning Bolt (20) | 4 / none | lightningjavelin (205) | lightning 1..40 | SrcDam 96 |
| Ice Arrow (21) | 4 / none | icearrow (28) | cold 6..10, length 50 | SrcDam 128 |
| Guided Arrow (22) | 4 / 10 | srvmissilea `guidedarrow` | none | SrcDam 128; do 10 = the seeking missile |
| Charged Strike (24) | 6 / 11 | srvmissilea `chargedstrikebolt` | lightning 1..30 | melee, spawns the bolts (calc1) |
| Plague Javelin (25) | 4 / none | plaguejavelin (43) | poison 10..16, length 75 | SrcDam 128 |
| Strafe (26) | 8 / 12 | srvmissilea `strafearrow` (682), srvmissileb `strafebolt` (683) | none | SrcDam 96; st 8 counts targets, do 12 shoots |
| Immolation Arrow (27) | 4 / none | immolationarrow (85) | fire 12..23 | SrcDam 128 |
| Decoy / Dopplezon (28) | none / 15 | - | - | summon `dopplezon`, petmax 1 |
| Fend (30) | 9 / 13 | - | none | SrcDam 128, tohit 40; do 13 = the repeated hit |
| Freezing Arrow (31) | 4 / none | freezingarrow (87) | cold 40..50, length 50 | SrcDam 128 |
| Valkyrie (32) | none / 16 | - | - | summon `valkyrie`, petmax 1 |
| Lightning Strike (34) | 10 / 14 | srvmissilea `lightningstrike` | lightning 1..25 | melee |
| Lightning Fury (35) | 4 / none | lightningfury (206) | lightning 1..40 | SrcDam 128 |

**Q2. What do the level-based cells say? (V run equal to d2d, clvl 30, no synergy; the harness's own numbers).**

| Skill | cell | L1 | L10 | L20 | L40 |
|---|---|---|---|---|---|
| Multiple Shot (12) | calc1 arrows `min(24, ln12)` / calc3 centre | 2 / 2 | 11 / 2 | 21 / 2 | 24 / - |
| Strafe (26) | calc1 max arrows / calc2 damage % / calc3 min arrows / aurarange | 4 / 5 / 2 / 35 | 10 / 50 / 4 / 35 | 10 / 100 / 7 / 35 | - |
| Fend (30) | calc1 max hits / calc2 ED % | 12 / 70 | 12 / 160 | 12 / 260 | - |
| Jab (10) | calc1 (ED %) | -15 | 12 | 42 | - |
| Impale (19) | calc1 ED % / calc2 break chance % / calc3 durability | 300 / 46 / 1 | 525 / 30 / 1 | 775 / 25 / 1 | - |
| Guided Arrow (22) | calc1 (damage %) | 0 | 45 | 95 | - |
| Charged Strike (24) | calc1 bolts | 3 | 5 | 7 | - |
| Lightning Fury (35) | calc1 bolts / aurarange | 2 / 15 | 11 / 15 | 21 / 15 | - |
| Lightning Strike (34) | calc1 / calc2 / aurarange | 20 / 2 / 20 | 20 / 11 / 20 | 20 / 21 / 20 | - |
| Decoy (28) | calc1 / calc2 frames / calc3 life % / petmax / resists (aura 1..4) | 10 / 250 / 50 / 1 / 4 | 100 / 1375 / 50 / 1 / 40 | 200 / 2625 / 50 / 1 / 80 | - |
| Valkyrie (32) | calc2 / delay / strength, dexterity (aura) / resists (aura, Decoy level 0) / armour % / to-hit (passive) | 25 / 150 / 25, 12 / 2 / 0 / 40 | 52 / 150 / 250, 120 / 20 / 90 / 400 | 82 / 150 / 500, 240 / 40 / 190 / 800 | - |
| Inner Sight (8) | armorclass (aura1) / length / range | -40 / 200 / 20 | -305 / 1100 / 20 | -815 / 2100 / 20 | - |
| Slow Missiles (17) | slow % (aura1) / length / range | 33 / 300 / 20 | 33 / 1650 / 20 | 33 / 3150 / 20 | - |

(Slow Missiles' radius is constant 20 and its slow constant 33 percent in the patched table, so points past the first only lengthen it.) Fend's hit count is capped at calc1 = 12 (Q4); Strafe's arrow count by level and the 35 radius are Q3. Inner Sight's armorclass cell reads the token `-edmn` in the patched table; the value is equal on both sides under the same stat hook, what the token means is **U**.

**Q3. Strafe and Multiple Shot: how many arrows, and when? (V run for the counts; S, V read upstream for the code).** Multiple Shot (do 8, FUN_005db410): `calc1` missiles (2 at L1, 11 at L10, 21 at L20, capped 24) at points a step apart across the line to the target, centred on it, `calc3` of them in the middle, damage the weapon's SrcDam 96 (75 percent of 128) each (S upstream re/skills.md:652..661). Strafe: start 8 (FUN_005dacd0) counts the monsters within `aurarange` (35) and clamps that count to `[min(calc3, calc1), calc1]` (2..4 at L1, 4..10 at L10, 7..10 at L20), do 12 (FUN_005dba40) shoots one arrow at the next target per call, `calc2` its damage percent (S upstream skills.md:781..783). I did **not** re-read these two functions (U for the code; V run for the numbers). d2d: `Fight::fire` / `player_shots` shoot Strafe's queued arrows on a 3-tick timer (fight.hpp:841, "ponytail: Strafe's arrows go out on a timer, not a repeated attack animation") (D3).

**Q4. Jab, Fend, Impale, Power Strike, Charged Strike, Lightning Strike: the melee ones? (V run for the cells; S for the code).** Fend (st 9 FUN_005dae30): hit count = min(enemies in reach (FUN_0056bc80), `calc1` = 12), then do 13 as Zeal with ED `calc2` (S upstream skills.md:418..420); d2d mirrors it (fight.cpp:681..685: `kicks_left = min(calc1, in_reach) - 1`). Jab: do 7 (FUN_005db2d0) builds toht, ED `calc1`, the element on each hit; the sequence (one hit per event frame, a spear's Jab is 21 frames and three hits, upstream skills.md:395) is by weapon class. d2d's player swing builds ED `calc1` and SrcDam the same way (fight.cpp:820..830); its merc Jab hits as one (fight.cpp:2111, D4). Impale (st 7, FUN_005dab40): ED `calc1` (300, 525, 775 percent), the weapon loses `calc3` durability with chance `calc2` percent (46, 30, 25) on a hit (upstream skills.md:421..426): the percent cells equal d2d; the durability rule in d2d was not read (U). Power Strike, Charged Strike and Lightning Strike are the Paladin-style elemental melee skills of the Amazon: their cells are equal, their do functions (2, 11, 14) and the bolts they spawn were not read here (U).

**Q5. The passives and Penetrate / Pierce (V run for the values; S for the use).** Passive stats by skill level, clvl 30 (V run equal):

| Skill | passive stat | L1 | L10 | L20 | L40 |
|---|---|---|---|---|---|
| Critical Strike (9) | passive_critical_strike (337) | 16 | 56 | 68 | 76 |
| Dodge (13) | passive_dodge (338) | 18 | 47 | 56 | 62 |
| Avoid (18) | passive_avoid (339) | 24 | 55 | 65 | 72 |
| Evade (29) | passive_evade (340) | 18 | 47 | 56 | 62 |
| Penetrate (23) | item_tohit_percent (119) | 35 | 125 | 225 | 415 |
| Pierce (33) | skill_pierce (166) | 23 | 71 | 85 | 95 |

Dodge swings, Avoid missiles, Evade while moving: game.exe's defender code FUN_0057dd60 reads the three stats (S monsterhit-trace Q4, V read there) and d2d's `take` does the same (combat.hpp:411..412: `moving ? evade : missile ? avoid : dodge`, one roll `rng(100) < value`); the values are the table's. Critical Strike: d2d puts stat 337 into `fighter.critical` (combat.hpp:143); how game.exe uses it was not read here (U). Penetrate adds attack rating percent (stat 119): d2d applies `item_tohit_percent` on pets (fight.cpp:3321) and the player's attack-rating sum; the player path was not compared (U). Pierce: Q6.

**Q6. What does the Pierce passive do in game.exe, and in d2d? (V run game side, V read, S)** game.exe: FUN_0059f940 (called when a missile is made; EAX = the shooter, argument = the missile unit):
1. 0x59f95c..0x59f985: the missile unit type is 3, the shooter type 0 or 1, and the missile row's byte +4 has the pierce bit (the mask byte at 0x6ce270; D: 60 rows of Missiles.txt have `Pierce` 1, among them arrow, javelin, firearrow, magicarrow, icearrow, coldarrow, explodingarrow, immolationarrow, freezingarrow, poisonjav, plaguejavelin, lightningjavelin, lightningfury, multipleshotarrow, multipleshotbolt, strafearrow, strafebolt, pilum, throwaxe; the Guided Arrow, Charged Strike and Lightning Strike rows were not checked).
2. 0x59f98e..0x59f9a8: `chance = stat 0xa6 (166, skill_pierce) + stat 0x9c (156, item_pierce)` of the shooter; zero ends the function with nothing set.
3. 0x59f9ac..0x59f9c7: the shooter's stat 0x148 (328, pierce_idx) is read and FUN_00650e40 turns it into a two-word seed.
4. 0x59f9d1..0x59f9ff: up to **four** rolls of the 0x6ac690c5 generator step, `roll % 100 < chance`, stopping at the first failure; the count of passes is kept.
5. 0x59fa05..0x59fa1a: FUN_00627260 sets the missile's stat 328 to that count.
6. Collision code 0x5ada60..0x5adae1 (V read): when the row has the pierce bit and the missile's stat 328 is not zero, it is decremented and the function answers 2 (fly on, not killed).

V run (`ama_pierce.py`, 6,776 cases: chance 0..110 x item 0 / 20 x eight shooter stat-328 values, plus 5,000 random): chance 0 gives nothing set; with the shooter's stat 328 = 0 the count is **0 for every chance up to 66 and 4 from 67** (so the first roll of that seed is 66); other seeds give other thresholds (examples: stat 328 = 1 gives 3 pierces at chance 60..80 and 4 at 99; 12345 gives none at 80 and 4 at 99). So game.exe's Pierce is a count of 0..4 per missile, decided when it is made. (The harness treats FUN_006253b0 as a stat getter, as drops.py does; that is an assumption of the run, not a claim.) Whether the shooter's stat 328 stays constant between shots (which would make the count the same for every arrow of a character: the table's 69 percent at Pierce level 9 would be the first level that pierces) is **U**: 16 sites in the image set a stat 0x148 (list in section 8), none was read.

d2d (S): `missile_info.collide_kill && !(missile_info.pierce && rng(100) < psum[kPierceIndex])` (fight.cpp:2971): the shooter's stat 328 is a percentage, one roll per collision, no cap. The passive stat is stored by name (gamedata_load.cpp:532: `passive_stat[i] = found->second`), so Pierce writes stat 166; nothing in d2d reads 166 or 156 (no `skill_pierce` / `item_pierce` in components/), and 328 is written by no skill, so for a character with only the passive, **d2d never pierces** (I from S; not run). The Penetrate / Pierce plan points are therefore worth nothing in a d2d soak.

**Q7. Decoy (Dopplezon) and Valkyrie? (V run cells; S the effect)** Decoy (do 15, FUN_005dc000): the owner's look, life = owner's max life x `calc3` (50 percent, constant), the owner's level, gone after `calc2` frames (250, 1375, 2625 at L1, 10, 20) (S upstream skills.md:848); d2d `Fight::summon_one` case kDecoy (fight.cpp:3344..3348): life `maxlife x calc3 / 100`, `until = calc2 x 40 ms`, stands idle: same rule by reading (V not run). Decoy's resist aura (four resists `min(lvl x 4, 85)`) and the unused-looking `calc1 = lvl x 10` are table cells; d2d does not read calc1 for Decoy (U whether game.exe does). Valkyrie (do 16, FUN_005dc1e0): petmax 1, strength / dexterity `lvl x 25 / 12`, four resists `min((lvl + Decoy level) x 2, 85)`, armour percent `10 x (lvl - 1)`, to-hit `40 x lvl`, sumskills Dodge / Avoid by the Amazon's own base levels (calc `skill('Dodge'.blvl)`; V run: 0 when the neighbour is 0, equal to d2d with neighbours at 20), upstream skills.md:861 lists "Valkyrie's Dodge / Avoid / Evade" among the unbuilt parts. Not run: the Valkyrie's base life and damage (MonStats `valkyrie`, `pettype`), as for the Necromancer pets (nec-skill-trace dead end 1).

**Q8. Slow Missiles and Inner Sight? (V run cells; U effect)** Both are do 6 rows (not read) that carry a target state: Slow Missiles `slowmissiles` with the stat `skill_handofathena` 33 within radius 20, for 300 + 150 x (L - 1) frames (D, V run cells); Inner Sight `innersight` with a negative armorclass. d2d reads neither skill by name; whether it applies the aura through the generic aura path (`monster_states`, states.cpp:259) was not traced (U). The stat id of `skill_handofathena` and what game.exe does with it (monster missile speed?) was not read (U).

## 3. d2d side (S, read at upstream main 72132b99)

Fight: `start_swing` pays and sets `kicks_left` from `calc1` (Fend capped by `in_reach()`) (fight.cpp:662..700); `skill_swing` builds the melee hit (ED `calc1`, SrcDam, knockback, conversion; fight.cpp:820..840); `cast_missile` / `fire` shoot the row after its do (fight.hpp:796..870): do 8 a fan, do 10 a seeking one, do 12 Strafe's queue (timer); `missile_blow` applies resist and pierce (combat.hpp:247..259) with the top-inclusive roll (the SKMISS finding); passives through `player_fighter` into `fighter.critical / dodge / avoid / evade` (combat.hpp:143, 151); pierce chance `psum[328]` (fight.cpp:2971). The Amazon's summons: Decoy (fight.cpp:3344), Valkyrie as a NecroPet (fight.cpp:3652, fight.hpp:945).

## 4. Deviations d2d vs game.exe

| id | rule in game.exe | rule in d2d | tag | differential |
|---|---|---|---|---|
| D1 | missile hit roll `min + rand(max - min)`, top exclusive (FUN_005a89a0), element part of Fire / Exploding / Immolation / Cold / Ice / Freezing Arrow, Lightning Bolt, Lightning Fury | `Rng::range(min, max)` top inclusive, modulus + 1 (combat.hpp:420) | **V run** (118,000 cases: 13,353 + 10,228 differ before, 0 after `23e2baec`; min / max equal) | yes: `ama_run.sh hit`; the SKMISS fix |
| D2 | Pierce: stat 166 + 156 of the shooter = chance, 0..4 pierces decided when the missile is made, stat 328 on the missile counts them down (FUN_0059f940, collision 0x5ada60) | one roll per collision against the shooter's stat 328 as a percentage; the passive's stat 166 and item_pierce 156 are not read (fight.cpp:2971) | V run (game side, 6,776 cases) + S (d2d side read only) | buildable: a `Fight` harness (`pets_merc_harness.cpp` shows one) with an Amazon whose Pierce is level 1..20, a row of monsters on a line, N arrows; oracle: units struck per arrow from the game side (`ama_pierce.py` for the count, then the collision function 0x5ada60 per unit) once the shooter's stat 328 question (section 8) is settled |
| D3 | Strafe shoots one arrow per action frame of a repeated attack animation, count clamp by monsters in 35 (FUN_005dacd0 / FUN_005dba40; S upstream) | queued arrows on a fixed 3-tick timer (fight.hpp:841, fight.cpp via `player_shots`) | S (d2d's own "ponytail" comment + upstream notes); the game side not run here | buildable: run FUN_005dba40 with fake units as in sorcrest2 (V run precedent) and read the frame gaps of the shots; d2d side `Fight` harness, `player_shots`; oracle: the tick of each arrow for 1..10 targets |
| D4 | a Jab sequence with a spear: three hits in 21 frames (upstream skills.md:395) | the merc's Jab "hits as one" (fight.cpp:2111); the player's goes through the sequence frames (start_sequence, fight.cpp:702..714) | S (d2d comment); only mercs affected | buildable as D3 on the merc's side; low value (the Act 2 desert mercenary attacks with Jab, internal/town/merc.go:100) |
| D5 | Magic Arrow converts a share of the physical damage to magic through the missile row's DmgCalc1 (`dl12`, base 1, per level 1) | not read | U | buildable: the missile calc (not the skill calc) under the same oracle as `ama_calc.py`, hooked to the missile row; d2d side `missile_damage` with the conversion |

Not deviations but open (U): Critical Strike's use in game.exe, Penetrate's attack-rating path, Decoy's calc1, Valkyrie's base stats, Slow Missiles' and Inner Sight's application, Poison / Plague Javelin's per-tick poison (the element min / max are skillfx.damage), Lightning Strike's and Charged Strike's spawned bolts, Impale's durability rule.

## 5. Bot findings (described, not changed)

| id | finding | where |
|---|---|---|
| B1 | `classOptions` gives the Paladin "Charged Strike" (lightning). Charged Strike is an Amazon skill (Skills.txt row 24, charclass ama, D); the Paladin has Charge. The immune workaround would offer a Paladin a skill it cannot have. | internal/combat/rules.go:238 |
| B2 | The Amazon list leaves out Lightning Bolt (the javelin plan's main attack from clvl 14, `amazon_javelin_normal30.json` `main_attack[2]`), Fire / Cold / Ice / Immolation Arrow, Guided Arrow, Magic Arrow and Jab; it lists Exploding Arrow only under fire and Freezing Arrow only under cold although both have SrcDam 128, so against a fire or cold immune the physical part still lands (D, the table's SrcDam; game.exe and d2d both apply the physical weapon part). A fire immune target then gets no Exploding Arrow from `ImmuneWorkaround` although it still does about its weapon share. Lightning Fury is listed with SrcDam 128 too. The model "one damage type per skill" understates every row with SrcDam (I; the magnitude depends on the weapon). | internal/combat/rules.go:244..245 |
| B3 | The multishot plan spends points on Pierce (rows at lines 450, 463, 476) with the tag "the guide's 'until 69 percent' is not recomputed" (U in its own tags). Table: Pierce is 69 percent at skill level 9 and 71 at level 10 (V run, Q5); game.exe's pierce is a 0..4 count decided at the shot with threshold 67 under the stat-328-is-zero reading (V run game side, Q6). The "until 69" in the plan therefore matches level 9 by the table, but the soak (d2d) credits nothing for any Pierce level (D2). | internal/gear/builds/amazon_multishot_endgame90.json:450, :463, :476; tags line 5 |
| B4 | The multishot plan notes "Magic Arrow for physical immunes" (main_attack note). Magic Arrow's missile converts only a share of the physical damage to magic (missile row DmgCalc1 `dl12`, base 1, per level 1; D, not evaluated, D5), so against a physical immune it is not a full-damage replacement (I). | internal/gear/builds/amazon_multishot_endgame90.json:61 |
| B5 | `internal/combat/friendly.go:43` says the decoy (monster 356 `dopplezon`) cannot be named by the AI column; fine, but Decoy's lifetime is calc2 frames (250 at L1, 2,625 at L20; Q7), so a census that counts the decoy as a permanent ally over-counts after that (D, V run cells). The Valkyrie's table cells carry no lifetime (V run: only the delay 150); whether she is permanent in game.exe was not read (U). | internal/combat/friendly.go:43 |

## 6. PR candidates

**No PR candidate.** D1 is the SKMISS candidate (`fix/missile-roll-top`, `23e2baec`, local); this card adds its Amazon evidence (118,000 cases, before 13,353 and 10,228 differ, after 0; harness `ama_missile_hit.py`) for the pack's "other skills" line; no branch was made or changed. D2 is a real d2d gap with the game side run, but there is no d2d-side harness yet, so under the 2026-11-01 rule it is a bot-side finding and an open question. D3 to D5 have no run.

## 7. Matrix cell updates (proposed text; no Amazon matrix doc exists, so a table is proposed here)

| Row | Cell | New value |
|---|---|---|
| all 30 Amazon skills | calc cells (counts, lengths, radii, percents, stat shares) | V run equal to d2d, 27,800 cases (`ama_run.sh calc`) |
| Fire / Exploding / Immolation / Cold / Ice / Freezing Arrow, Lightning Bolt, Lightning Fury | element min / max | V run equal (skillfx.damage 116,000; 118,000 here) |
| same eight | roll and life after the hit | D1 V run, fixed by the SKMISS candidate |
| Poison Javelin, Plague Javelin, Power Strike, Charged Strike, Lightning Strike | missile / melee hit roll | U (not in the d2d test's element switch; the D1 rule applies by read) |
| Pierce | missile pass-through | D2: game.exe 0..4 count from stats 166 + 156 (V run game side), d2d ignores both (S) |
| Penetrate, Critical Strike | effect | values V run equal; effect U |
| Dodge, Avoid, Evade | values by level | V run equal; the dodge rule S (monsterhit-trace) |
| Multiple Shot, Strafe | arrow counts by level | V run (cells), S (code); timing D3 |
| Jab, Fend, Impale | ED, hit cap, durability | cells V run equal; Fend cap S, Jab sequence S, Impale durability U |
| Decoy, Valkyrie | life share, duration, aura stats | cells V run equal; effect S; pet base stats U |
| Slow Missiles, Inner Sight | length, radius, stat | cells V run equal; application U |
| Guided Arrow | damage %, seek | calc1 V run equal; flight S |

Ledger lines (docs/coverage/ledger.tsv): `skills.ama 6,8-10,12-15,17-19,22-26,28-30,32-34 TESTED-0-DIFF 27,800 cases (calc aspect)`; `skills.ama 7,11,16,20,21,27,31,35 TESTED-DEVIATION 118,000 cases (D1)`; `missiles 12,28,40,41,85,87,205,206 TESTED-DEVIATION 118,000 cases (D1)`.

## 8. Dead ends and limits

1. **The calc differential does not cover** stats the getter returns 0 for (any calc reading an item stat). The Amazon rows read skill levels, masteries and clvl; the `-edmn` token of Inner Sight is not a stat read (the value is non-zero under the zero-stat hook), but what it is stays **U**.
2. **Pierce's seed.** FUN_00650e40 turns the shooter's stat 328 into the roll seed; whether stat 328 of a player ever changes is not known. 16 sites in the image push 0x148 and call the stat setter (0x45d082, 0x45e785, 0x45f491, 0x481219, 0x4cd36b, 0x4cd529, 0x549d50, 0x549dd0, 0x549e50, 0x54a010, 0x54a090, 0x54a110, 0x5720f1, 0x5980b0, 0x59fa1a, 0x5adadc); the last two are this function and the collision code, the rest are not read (0x148 may be an offset there). Until read, "the count is the same for every arrow" is **I**.
3. **Strafe, Multiple Shot, Jab, Guided Arrow and the melee skills' code** is upstream's reading (S), not mine; I did not re-read FUN_005dacd0, 005dba40, 005db410, 005db2d0, 005db6d0.
4. **Poison Javelin / Plague Javelin** hit and per-tick poison: not run (the d2d test's element switch covers fire, lightning, cold). Needs the poison length rule of `life_apply.py` (cold / poison / burn side effects exist in the harness).
5. **The SrcDam (weapon) part of every arrow** is not part of the hit run; to-hit (`ToHit` column: 10, 20, 30, 40, 100) and the weapon's physical roll are the Sorceress-free part that sk-melee-trace covers for melee only.
6. **Local interpreter calls were not available to this card** for scripts outside the worker host, so the table dumps were read with cut / awk one-liners; no claim depends on that.
7. A harness bug seen in earlier traces (case-sensitive skill lookup, Valkyrie's `sumskill` names) did not recur: the d2d side reused `nec_test.cpp`, which already has the case-insensitive lookup.

## 9. Drill recipe (lab, fork verbs; for later, not run here)

Fork verbs (docs/forkdbg.md): `debug skill`, `debug points`, `debug spawn`, `debug mon set <id> life .. maxlife .. res <type> ..`, `debug cast <skill> unit <id>|<x> <y>`, `debug missiles`, `debug states`, `step`, `dump stats`. A throwaway Amazon on one slot, `--no-save`, lockstep, from docs/forkcheats.md.

| Drill | Confirms | How |
|---|---|---|
| A1 Pierce | D2, B3 | `debug points 33 1/9/10/20`, bow and arrows, ten monsters on a line (`debug spawn`), `debug cast` Multiple Shot or a plain attack at the first; count the monsters struck per arrow (`debug missiles` / life lost). Game: 0 pierces at levels 1..8 and 4 pierces at 9+ if the stat-328-is-zero reading holds (every arrow the same); d2d (this build): none at any level |
| A2 Pierce seed | the I in dead end 2 | `dump stats` of the character before and after twenty shots: does stat 328 change? |
| A3 Strafe timing | D3 | five monsters in 35, `debug points 26 1/10/20`, `debug cast 26`, `debug missiles` per `step`: the ticks between arrows, and the arrow count against `clamp(monsters, calc3, calc1)` |
| A4 Magic roll | D1 | as sk-missile-trace M4 with Fire Arrow level 1 (element 1..4 in 256ths: 256 to 1024), one target with life 1000, read the life lost over 200 casts |
| A5 Magic Arrow | D5, B4 | `debug mon set` physical resist 100, Magic Arrow at levels 1, 10, 20: life lost per hit |
| A6 Decoy | Q7 | `debug points 28 1/10/20`, cast, `monsters` list: the decoy's life (50 percent of max life) and its end at 250 / 1375 / 2625 frames |
| A7 Valkyrie | Q7 | `debug points 32 1/10`, cast, `dump stats` of the pet (strength, dexterity, resists, armour) against the table cells |
| A8 Slow Missiles | Q8 | cast on a shooter, `debug states`: state and length 300 / 1650 / 3150 frames; its missile's velocity |

## Shared-file lines for the merger

See `openspec/changes/skamazon/merge-lines.txt`.
