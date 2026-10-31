# Monster damage to the player: game.exe 1.14d traced vs d2d (agent DECOMPMON, 2026-10-30)

> status: research, current. Question: do monsters hurt the player the same in d2d as in the original (our death gates: deaths per 1000 kills; Soaky's deaths to Fallen, Carvers, Rakanishu and a lightning-enchanted Carver in Stony Field)? Tags: V = read in game.exe (`disasm.py`) or run in game.exe's own code under unicorn by this agent (`tools/emu/monster_hit.py`); V-up = upstream's own oracle re-run by UPSTREAMGAPS (item 2); D = 1.14d patched tables read with `mpq-cat` (docs/process/mpq-data-reading.md, `Id<TAB>hcIdx` seen); S = upstream `docs/research/re/combat.md`, `monsters.md`, `monster-ai.md` (psi29a/d2decomped main 32768ddb) or d2d code read; U = unverified; I = inference. Classic 1.14d, Normal, one player. Offline only: no game instance, nothing pushed or posted.

## Verdict in four lines

1. **The melee numbers are the same.** Hit chance (FUN_0057d9b0), the physical roll (FUN_0057b420) and the application to life (FUN_0057c6c0) were run in game.exe under unicorn and equal d2d's `hit_chance` / `monster_blow` / `Fight` rules: 100,000 + 100,000 random cases, 0 differ. d2d rolls the damage in whole points (inclusive) where game.exe rolls 256ths (top exclusive): same mean to 1/512 hp, a wider spread in d2d (it can land exactly the max). A running player is hit by every swing in both.
2. **One real deviation: the get-hit (flinch) rule.** game.exe's FUN_0057cb00 is tiered and keyed by hit class (a player hit by a Fallen, Carver, Shaman or Quill Rat, hit class 3: from 1/16 of max life 37.5 %, from 1/8 75 %, from 1/4 always); d2d flinches at a twelfth, always, for a player, and at an eighth for a monster (both marked `ponytail` upstream). Oracle: 672 boundary + 100,000 random cases equal the tiered reading; 18,502 of the random cases differ from d2d's twelfth. Reported as a finding (PROOF incomplete, no PR branch): `upstream-pr-candidates-monster.md`.
3. **Lightning Enchanted is the same shape**: 8 bolts of missile 195 at level mlvl / 2 on every hit taken, at most once per 10 frames, no to-hit roll; a Stony Field unique Carver (mlvl 7) throws 5 lightning per bolt. Fire Enchanted and Cold Enchanted act only on death (none of the four Act 1 superuniques has them; a random unique can).
4. **Our deaths are explained by the shared rules, not by a d2d gap**: the bot runs through packs (every swing lands) and Carvers at mlvl 4 deal 2 to 4 per swing once a second each; a pack of six takes a clvl 6 Sorceress's 60 life in about 3.5 s. Rakanishu adds 3 to 5 lightning per hit and 8 x 5 lightning bolts whenever he is hit.

## 1. The melee path in game.exe (addresses, V unless marked)

| Address | What it does | Tag |
|---|---|---|
| `0057ec10` (ECX game, EDX attacker, stack defender, to-hit bonus, extra reach) | the swing: reach test `00622c40` (3 for a monster, 1 for a player, + the extra); a player defender in mode 3 (RN) is hit without a roll (`0057ec71`); else `0057d9b0`; then the defender's block/dodge rolls `0057dfb0` (any success clears the hit bit); result bit 4 (`0057ed38`) = the hit landed and the defender has no state 0x36 | V |
| `0057d9b0` (ECX attacker, EDX defender, stack bonus, missile flag) | monster AR = stat 19 + stat 2 x 5 + bonus (`0057db93`), x (100 + stat 119) / 100 by `00483360`; defense = `006223f0` + stat 33 (melee) or 32 (missile); negative side moved across; chance = AR x 100 / (AR + DEF) (100 when both 0), x 2 x alvl / (alvl + dlvl), 5..95 (`0057db22`, `0057db2b`); hit when the attacker's seed (+0x20) % 100 < chance (`0057db35`..) | V (oracle `hit`: 100,000 cases = d2d `hit_chance`) |
| `005a5470` | the monster swing wrapper: `0057ec10(unit, defender, bonus, 0)`; callers at `0056e6b3`, `0056e708` (skill hits) pass the skill's to-hit (U which value for the plain attack: Skills.txt row 0 `Attack` has no ToHit / LevToHit, D) | V (call sites) |
| `0057b420` (ECX attacker, EDX use-stats, stack weapon, min, max, skill ED %, flat, SrcDam) | no weapon (`0057b48a`): min = stat 21 (at least 1), max = stat 22 (at least 2), both `<< 8`; + stat 111 `<< 8`; max at least min + 0x100; p = stat 25 + ED (+ stat 0 as % when EDX set), at least -90; min' = min + MulDiv(min, stat 18 + p, 100), max' likewise with stat 17; + flat; roll = min' + `0045c3e0`(seed, max' - min'): TOP EXCLUSIVE; x SrcDam / 128 when not 0x80 | V (oracle `dmg`: 100,000 cases) |
| `0057c6c0` (apply a damage record) | record int 0x13 (+0x4c) = total in 256ths; life stat 6 -= it; result < 0x100 -> 0 = dead (`0057c865`..`0057c874`); +8 is first clamped to the life left (`0057c7f8`); then mana/stamina, stun, cold, freeze, poison, burn records | V |
| `0057cee0` (the "takes damage" event) | player branch at `0057d364`: result bit 4, no state 0x15 (stun) and `0057cb00(unit, record, record[0x60] hit class)` == 0 -> mode 4 (GH) through `005809d0`; a monster's branch at `0057d0b2`..`0057d0c8` calls the same `0057cb00` | V |
| `0057cb00` (unit, record, hit class) -> 1 no get-hit, 0 get-hit | state 1 -> none; record[0x28] == record[0x4c] -> none; damage [0x4c] < 0x100 -> none; max = `00625d10` (max life); divisor by hit class from the table at `0057cbf8` (classes 1..12: 16, 8, 16, 32, 64, 8, 16, 32, 16, 8, 8, 16; others 16); damage < max / div -> none; < max / (div / 2): `00472210`(seed, 2) == 0 -> none; < max / (div / 4): rand(4) == 0 -> none; a monster whose MonStats flag test `0046c140(row, 3)` holds -> none; else get-hit | V (oracle `gh`: 672 boundary + 100,000 cases) |
| `00573cb0` -> `006538a0` | monster stats at spawn: MonLvl row at the monster level x MonStats % / 100 (MulDiv): AC, TH, HP, DM (A1MinD / A1MaxD / A1TH / A2.. / S1..), El1..3 as DM %; life rolled min + rand(max - min + 1) `<< 8`; stats 21 / 22 / 19 are whole points | S (monsters.md "Monster stats") + V-up (upstream `monstats.py`, 2202 / 2202 rows, re-run in item 2) |
| Unique mods in the hit | `005a0e80` champion: +dmg% (stat 25) / +tohit% (119) from MonUMod rows 11 / 10 x DifficultyLevels ChampionDamageBonus (90, D) / 100; `005a17e0` strong: rows 15 / 13 (unique) or 14 / 12 (minion); every unique: row 13 "+tohit%" 100 x 90 / 100; 9 / 17 / 18 enchanted: element min / max = MonLvl DM x rows 28..33 (66 / 100 %, D) for a unique | S (monsters.md "Champions and uniques") |
| Lightning Enchanted (mod 17) | `005a37d0` / `005a29a0` via `005a2ba0` on the "takes damage" event (bit 4 of the hooks table 0x73c0b8): in GH two frames on, else at once; at most once per 10 frames (monster data +0x18 flag 0x100); 8 missiles 195 `lightunique`, 4 ways x 2, at level mlvl / 2 (at least 1) | S (monsters.md "Boss mods in the fight") |
| missile 195 `lightunique` | ToHit blank (no roll: `005adf10` runs `0057d9b0` only when the row's ToHit is set, item 1), EType ltng, EMin 1 + 2 a level, EMax 1 + 2 a level, Range 98, Vel 12, CollideKill 1, HitClass 64 | D |
| Fire Enchanted (9) | on death, 4 frames on: `0057e090` hits everyone within difficulty + 4 subtiles for v = max life x MonsterCEDamagePercent (50, D) / 100 x 3/4 (Normal), rolled 60 %..v, half physical half fire | S |
| Cold Enchanted (18) | on death, 4 frames on: a nova of missile 194 (6 + 4 a level cold, ELen 75) at level mlvl / 2 | S + D |
| Attack cadence | an attack mode's end goes to NU through `005a7c20` and the think is scheduled aidel frames on (`005a73e0`; MonStats aidel 15 for every Act 1 type, D); A1 frames per direction and action frame from AnimData (D): Fallen / Carver `FAA1HTH` 10 frames, hit at 7; Shaman `FSA1HTH` 17 / 8; Zombie `ZMA1HTH` 16 / 9; Quill Rat `SIA1HTH` 16 / 11; Beast `YEA1HTH` 12 / 5; Skeleton `SKA11HS` 16 at speed 224 / 10; all at rate 256 (one frame a tick) but the skeletons | S + D |
| Hit classes (MonStats2 HitClass, D) | Fallen, Carver, Shaman, Quill Rat, Skeleton, Archer 3; Beast 1; Zombie 12 | D |

Not in the melee path: no "256ths truncation" of the kind item 1 found for spells: a monster's swing is rolled in 256ths and taken off a 256ths life. Crushing blow / open wounds / deadly strike come from MonProp or items, none on Act 1 Normal monsters (D, MonStats MonProp blank for the rows above; S for the hooks `005bffc0`).

## 2. d2d, item by item

| Item | d2d (upstream main 32768ddb; the lab fork has the same files, U not diffed) | Verdict |
|---|---|---|
| Hit chance | `rules::hit_chance` (combat.hpp:28); `monster_blow` skips the roll for a running player and a non-missile; block a third while running; dodge / avoid / evade after | SAME (V, oracle) |
| Damage roll | `rng.range(a1_min, a1_max)` inclusive whole points, then DR % and flat; `Foe::damage` whole points; `Fight` does `life -= damage << 8`, dead at life <= 0 | SAME mean (1/512 hp), wider spread; dead at 0 vs under 1.0 (U material: a hit landing on 0 < life < 1 is a corner) |
| Spawn stats | `rules::monster_stats`: MonLvl x % / 100 (pct), `to_hit` from L-TH x A1TH | SAME (V-up 2202 / 2202 at the row's own level; the area level at spawn in the running game is item 2's open point, U) |
| Unique bonuses | `uniques.hpp` `boss_stats` (+dmg%, +tohit%, enchanted elements from the MonLvl DM x the rows), `kBossBonus` = the difficulty's 90 % | S (read; not oracle-checked here) |
| Apply order | block -> dodge -> physical less DR -> each element less resist less MDR; poison / cold lengths | SAME shape (S) |
| Get-hit (player) | fight.cpp:4903 `damage x 12 >= max life`, always | DIFFERS (V): tiered by hit class in game.exe; 18,502 / 100,000 random cases differ; finding only, see the candidates doc |
| Get-hit (monster) | `hurt()`: `damage x 8 >= max_hp` (upstream combat.md "Approximations") | DIFFERS the same way (S for the d2d side; the game.exe call at `0057d0c8` is the same function with the monster's flag test) |
| LE bolts | `Fight::boss_events` (fight.cpp:1716): `hit_points < last_hp` and 10 frames since the last, 8 `lightunique` 4 ways x 2 at `max(level / 2, 1)`, `to_hit = 1 << 20` (always hits), damage `row_damage(...) >> 8` whole points (5 at level 3: exact) | SAME (S); two small ones: a hit that rounds to 0 whole points does not count as a drop (game.exe: any damage > 0 in 256ths triggers), and in GH d2d fires at once, not 2 frames on (upstream's own note) |
| FE / CE death blasts | `fire_blast`, 64 `coldunique` ring 4 frames on | SAME shape (S); not in Act 1's superuniques |
| Attack cadence | A1 for its AnimData length (`npc_timing`), the hit at the action frame (ai.cpp:1879), then NU and `next_act = now + aidel x 40 ms` (ai.cpp:2578) | SAME rule (S); the AI module is upstream's best-verified (177 functions, riskmap section 3) |
| Monster missiles (spike1, skbowarrow1) | `monster_blow(.., missile.src, true, rng, true)`, src.to_hit = A2TH / S1TH (ai.cpp:1904); game.exe rolls `0057d9b0` for them too (ToHit 1 in Missiles.txt, D) | SAME shape (S) |
| Shaman fire | skill 160 `ShamanFire`, missile `shafire1`: fire 1 + 10 a level .. 4 + 10 a level (D), skill level 1 (Sk2lvl, D) = 1 to 4 fire a bolt, ToHit blank | d2d path not read (U) |

## 3. Numbers for the death-risk model (D + V; the chance column is the shared formula)

Monster level = the area's level in Normal (S); unique + 3 (`005a0e40`, V in xp-award-trace).

| Monster (area, mlvl) | AR = L-TH x TH % | Swing (DM x A1 %) | Mean per landed swing | Chance vs a Sorceress DEF 20 / 40, clvl 3 / 6 (standing) | Swing period |
|---|---|---|---|---|---|
| Fallen (Blood Moor, 1) | 8 x 101 % = 8 | 1 to 2 | 1.5 | 14 % / 9 % at clvl 3; 8 % / 5 % at clvl 6 | 10 + 15 frames = 1.0 s |
| Fallen (Cold Plains, 2) | 12 | 1 to 3 | 2 | 30 % / 18 %; 18 % / 11 % | 1.0 s |
| Carver (Stony Field, 4) | 26 x 91 % = 23 | 2 to 4 | 3 | 61 % / 36 %; 43 % / 29 % | 1.0 s |
| Carver (Dark Wood, 5) | 35 x 91 % = 31 | 2 to 4 | 3 | 70 % / 43 %; 57 % / 37 % | 1.0 s |
| Rakanishu (Stony Field, 7; mods 17 LE, 6 fast; +90 % to-hit as a unique) | 56 x 91 % x 1.9 = 96 | 2 to 5 + 3 to 5 lightning | 3.5 + 4 = 7.5 | 95 % / 86 %; 91 % / 75 % | 1.0 s (fast = velocity only, S) |
| LE bolt (unique Carver / Rakanishu, mlvl 7, level 3) | no roll | 5 lightning each, 8 per trigger | up to 40 at point blank | every hit he takes, at most each 0.4 s | |
| Fallen Shaman fire (level 1) | no roll | 1 to 4 fire | 2.5 | | S1 cast |

Running (mode RN): every swing lands, block a third (both backends, V / V-read). Player life: Sorceress 40 at clvl 1, +1 a level, +2 a vitality point (CharStats D: hpadd 30, vit 10, LifePerLevel 4, LifePerVitality 8 quarter points), so 45 to 95 at clvl 6 depending on vitality points (I).

Expected damage per second (I, from the rows above): a Carver pack (6) on a running clvl 6 Sorceress: 6 x 3 x 1.0 = 18 hp/s, 60 life gone in 3.3 s; standing with DEF 40: 6 x 3 x 0.29 = 5.2 hp/s. Rakanishu's camp (him + 8 Carvers) on a running target: 7.5 + 24 = 31 hp/s before the bolts; one Charged Bolt volley into him at melee range answers with 8 x 5 lightning. A Fallen pack (5, Blood Moor) on a runner: 7.5 hp/s; standing at clvl 6: 0.4 hp/s. The Stony Field deaths in `docs/g1-log.md` (Rakanishu 11:42, Fallen / Carver 12:53, Carver 12:55 and 14:12, potions unused) and the a1 / a11 Carver deaths in `docs/plan/death-data.md` fit "ran into a Carver pack and did not drink", with no rule that d2d would apply differently. The replicated soak should therefore show the same deaths per 1000 kills from the same bot behaviour, within the fidelity-gate tolerance (1.0 per 1000); what could still move it is listed below.

## 4. What is second order between the backends (I)

1. Get-hit: for a Carver's 4 on 60 life (6.7 %), game.exe flinches 37.5 % of the time (over 1/16), d2d never (under 1/12); for Rakanishu's 10 on 60, game.exe 75 %, d2d always. A flinch interrupts the cast (Sorceress GH: 8 frames, AnimData `SOGHHTH`, D) and lets the pack close: sign depends on the monster, not one-directional.
2. d2d's inclusive whole-point roll gives the exact max one time in (max - min + 1); game.exe never does. Tail only.
3. d2d lets a player live at 0 < life < 1 hp; game.exe kills at under 1.0. One hit in thousands.
4. The monster level at spawn in the running d2d (area level vs the row's level: item 2, U). Carver rows say Level 5 (D) but Stony Field is level 4: if d2d spawned Carvers at 5 they would hit 10 % more often and the same damage (DM 4 at both). Worth one `dump` on the lab binary.

## 5. Reproduce

```sh
# The worker host, read-only game.exe copy, upstream harness (never run as a game)
cd ~/d2d-lab/pr/monster/tools/emu && cp ~/code/d2lod-wt/decompmon/tools/emu/monster_hit.py .
D2_MPQ_DIR=~/d2d-lab/data /tmp/casttrace-venv/bin/python monster_hit.py --exe /tmp/casttrace/game.exe 100000
#  hit: 100000 cases, 0 differ / dmg: 100000 cases, 0 differ / gh: 672 boundary + 100000 random cases, 0 differ from the tiered port; 18502 differ from d2d's deterministic twelfth (about 12 s)
D2_MPQ_DIR=~/d2d-lab/data /tmp/casttrace-venv/bin/python disasm.py --exe /tmp/casttrace/game.exe 57d9b0   # and 57ec10 57b420 57c6c0 57cee0 57cb00
```

## 6. Open and unverified

- The to-hit bonus the plain monster attack passes to `0057ec10` (the wrapper `005a5470`'s callers were not read; Skills.txt `Attack` has none, so 0 is the expectation).
- The unique / champion stat bonuses and the enchanted element rows were read from upstream's doc and d2d's `boss_stats`, not run.
- Whether the hooks table entry for mod 17 fires on a sub-1-hp hit was not run (S says "takes damage" event; d2d needs a whole-point drop).
- The Shaman fire path in d2d; the merc as a target (not in scope).
- Nothing here was checked live on either backend: the model's hp/s are arithmetic on the traced rules and the D tables.
