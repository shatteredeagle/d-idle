# SEEDDIST: are d2d's outcome distributions equal to game.exe's, with one stream instead of per-unit seeds? (card seeddist, 2026-11-02)

> status: current (2026-11-02). Offline: game.exe 1.14d's own code under the emulator on the worker host, d2d's real compiled code from a scratch upstream build, no game, no d2d slot, no Go change, nothing pushed or posted. Question (docs/research/seed-study.md section 1.3 and 7, docs/research/tierb-design-20261102.md "statistical A:B first"): game.exe gives every unit its own seed `{step of game+0xd0, 666}`; d2d rolls monster life on one world stream and drops on a unit seed of its own game-seed model. Seed-for-seed equality is out of reach, so the test is whether the OUTCOME DISTRIBUTIONS agree for the Act 1 classes the early game meets.
>
> Tags: **V read** read in the 1.14d disassembly (`tools/emu/sd_dis.sh`, upstream `disasm.py`) or in the d2d source at file:line; **V run** game.exe's real functions run under the emulator and/or d2d's real functions run in a compiled test, the counts stated; **D** from the patched MPQ tables; **S** shown by an existing doc (named); **I** inferred; **U** unverified.

## Answer in eight lines

1. **For the Act 1 classes, d2d's outcome distributions equal game.exe's in every quantity tested.** Life roll, hit roll vs defence, damage roll mean and the first-drop outcome (class pick, quality, item count, NoDrop) show no deviation beyond chance in 330 p-values (16 below 0.05 where 16.5 are expected, none below 0.01; section 3.7), plus the one known shape deviation of the damage roll (D3). Classes: Fallen, Zombie, Skeleton, Quill Rat, Rogue (Dark Hunter), Gargantuan Beast (brute1), Blood Raven, Andariel (V run, section 2 for what was run).
2. **The rule functions are identical seed for seed.** The same 20,000 unit seed states per class into game.exe's `FUN_0055a6d0` and d2d's `roll_drops`: 360,000 first-drop cases (9 classes x 2 magic-find values x 20,000), 0 differ; the same states into the life roll: 160,000 cases, 0 differ in life (V run, section 3.1). Distribution equality therefore reduces to equality of the seed SUPPLY, which the samples test.
3. **The mechanism differs on purpose and the distributions do not care (D1).** game.exe rolls life on the unit seed (`0x573f83..0x573f8f` calling `FUN_0045c3e0` with ecx = unit + 0x20); d2d rolls it on the world stream `Rng{0x2545f491}` (ai.cpp:2744, fight.cpp:4890, world.hpp:295). Both are uniform on the row range (goodness of fit p between 0.17 and 0.94 on 100,000 units per class per side) and equal to each other (two-sample chi-square p 0.2 to 0.98).
4. **Cross-game structure (D2).** The d2d world stream starts at the same constant in every game: 20 d2d games with equal draw history give 20 of 20 identical 1000-monster life sequences (V run). In game.exe the sequence depends on the map seed. For a statistical A:B (many monsters per run) this changes nothing; for a seed-for-seed comparison it is the blocker already named (seed-study.md B2).
5. **One real mechanical deviation, carried and now quantified (D3):** monster physical damage is a continuous roll in 256ths with the top excluded in game.exe, whole points inclusive in d2d. Means agree within 0.05 hp (smallest p 0.044 over 8 classes), the variance of d2d's per-hit damage is 1.16 to 3.0 times game.exe's (small ranges worst), KS D up to 0.50.
6. **Real room population cross-check (V run):** the units game.exe's own room population makes (FUN_00573cb0 itself, Normal, plain monsters, 200 seeds, 6 levels) have life uniform on the row range as the model says, section 3.6.
7. **PR candidate: none.** No d2d deviation is proven that a fix could remove with before > 0 and after 0: D1/D2 are by design of the single stream and invisible in distribution; D3 is carried (F13) and needs a rule decision.
8. **Ledger:** formula key F16 (treasure class pick and item quality) earns TESTED-0-DIFF with 360,000 seed-for-seed cases (the 540,000 first drops per side of the distribution runs support it; section 7). Nothing else in the ledger moves.

## 1. Questions answered (Q1..)

| id | question | answer | where | tag |
|---|---|---|---|---|
| Q1 | How does game.exe make a monster's unit seed? | `FUN_00552df0(ecx = unit, edx = game seed ptr)`: `FUN_00650e30` sets unit +0x20 = {1, 0x29a}; then ONE step of the game seed: `mul 0x6ac690c5`, `add high`, `adc`, stored back to the game seed ([esi], [esi+4]); unit +0x28 = the new low word; `FUN_00650e40(unit + 0x20, low)` stores {low, 0x29a = 666}. | 0x552e1e..0x552e50, 0x650e30, 0x650e40 | V read; V run (800,000+ unit makes in the emulator, every one asserted equal to a Python mirror of the step) |
| Q2 | What is `FUN_0045c3e0`? | roll below edx on the seed at ecx: edx <= 0 returns 0 with no step; a power of two takes the new low & (n - 1); else new low % n (`mul`, `add`, `adc`, store, `div`). d2d's `Rng::operator()` is the same (rules.hpp:154-170). | 0x45c3e0..0x45c40f | V read; V run (asserted per roll against the mirror in `sd_game.py`, and by the 160,000-case control) |
| Q3 | How is a plain monster's life rolled? | `edx = (max - min) + 1`, `ecx = esi + 0x20` (the unit seed), `call 0x45c3e0`, `+ min`, `+ MulDiv(life, player-count %, 100)`, capped at 0x7fffff, `<< 8` into stats 7 and 6. | 0x573f83..0x573fc8 (FUN_00573cb0) | V read; V run (the real unit population, section 3.6) |
| Q4 | Where does d2d draw life? | `make_monster(..., Rng& rng, ...)` -> `monster_stats(..., rng, ...)` -> `rng.range(min, max)`; `spawn_monsters` is handed `Fight::rng`, a reference to the world `Rng rng{ 0x2545f491u }`. The unit seed (`Spawn.seed`, a step of `Spawning.game` per unit, monsters.hpp:121) is used for the look, the variant, the MonProp row, the AI's draws and the drop, not for life. | ai.cpp:2744, 2728; fight.cpp:4890; world.hpp:295; ai.cpp:138-141, 158; loot.hpp:112; monsters.hpp:361-372 | V read (the assignment of Spawn.seed for populated rooms is not located: I; the camp NPCs' `it->seed = Rng{ spawning.game.next() }` is gamedata.cpp:311) |
| Q5 | What does the game stream do before the first unit? | `FUN_00530930` steps the game seed four times (regions, objects, sunitproxy, quests); d2d's `start_spawning` does the same four steps (gamedata.cpp:194-198). Other units (items, missiles, the merc, preset objects) step it in game.exe between monsters; d2d steps only for what it makes. | re/monsters.md 122-166 (S); gamedata.cpp:187-203 | S + V read |
| Q6 | Monster to-hit and physical roll | to-hit `FUN_0057d9b0`: chance = AR x 100 / (AR + def) x 2 alvl / (alvl + dlvl), clamped 5..95, hit when the attacker's seed `% 100` is below it; physical `FUN_0057b420`: `min << 8 + rand((max - min) << 8)`, top excluded. d2d: `hit_chance` (combat.hpp:28) and `monster_blow` (combat.hpp:403-436): `rng(100) >= chance` misses, damage `rng.range(a1_min, a1_max)` whole points inclusive. | monsterhit-trace.md section 0, monster-attack-trace.md Q1 | S for the addresses; V run for this card's counts (section 3.2, 3.3) |
| Q7 | First drop | `FUN_0055a6d0` off the dropper's seed as the monster's life, AI and combat left it (re/drops.md "The dropper's seed at death"); d2d `Loot::drop` -> `tc_upgrade` + `roll_drops` on `monster.seed` (loot.hpp:100-112). At Normal no class move-on (`lvl` 0, game FUN_0055afa0 only in expansion past normal). | re/drops.md 33-130 (S), loot.hpp:94-118 | S; V run (sections 3.1, 3.4) |
| Q8 | Do the outcome distributions agree? | yes for every quantity and class tested; numbers below | sections 3.1 to 3.6 | V run |

## 2. What was run

All on the worker host, 2026-11-02, through `tools/emu/run-on-worker.sh`. Game side `tools/emu/sd_game.py` (upstream emulator `drlg.boot()`, game.exe 1.14d with the patched tables): real `FUN_00552df0` (unit seed), `FUN_0045c370` (step), `FUN_0045c3e0` (roll), `FUN_006538a0` (stat row), `FUN_0057d9b0` (to-hit), `FUN_0057b420` (physical roll), `FUN_0055a6d0` / `FUN_00654e00` (drop and class) with upstream's `drops.py` oracle hooks; the getters hooked as `monster_hit.py` hooks them. d2d side `tools/emu/sd_d2d_test.cpp`, built by `sd_build.sh` against the scratch upstream build used by the earlier traces (libs of main 7d5b3a5c, source tree sorcrest-d2d): real `make_monster`, `monster_look` / `monster_variant` / `monster_props`, `monster_blow`, `tc_upgrade`, `roll_drops` on the real loaded tables. Statistics `sd_stats.py` (pure Python; no scipy on the worker host: chi-square and Kolmogorov p-values coded in the script from the incomplete gamma and the Kolmogorov series).

- **Classes (MonStats Id, D):** fallen1 (row 19, level 1, life 1..4, A1 1..2, TC `Act 1 H2H A`), zombie1 (5, level 1, life 7..12, A1 1..3), skeleton1 (0, level 2, life 7..11, A1 1..3), quillrat1 (63, level 1, life 1..5, A1 1..2, TC `Quill 1`), corruptrogue1 (43, "Dark Hunter", level 2, life 5..9, A1 1..3), brute1 (28, "Gargantuan Beast", level 2, life 11..19, A1 2..3), bloodraven (267, level 10, life 113 fixed, A1 4..6, TC `Blood Raven`), andariel (156, level 12, life 1024 fixed, A1 6..19, TC `Andariel`, quest TC4 `Andarielq`, TCQuestId 6). The brief's "Rogue" is taken as the Act 1 Normal archer row corruptrogue1 (U: if the Rogue Scout rogue2, TC empty, was meant, it never drops).
- **Seed supply (both sides, own seeds per class, V run for the functions, I for the gap model):** games with a map seed, 4 creation steps, then one step per unit made; even games a gap of 1 (consecutive units), odd games 1..5 extra steps between units (items, missiles, preset objects). Game side: the unit seed from the real `FUN_00552df0`, then the look roll (`rand(3)`, a region set pick, I: the true set count is 1..3 per type) and the life roll on it. d2d side: `Rng{ game.next() }`, then the real `monster_look` (3 sets), `monster_variant`, `monster_props`; life from the world stream.
- **Case counts:** life: 100,000 game units + 100,000 d2d units (one stream) + 20 x 1000 d2d units on disjoint stretches + 20 x 1000 on equal starts, per class (8 classes); hit: 4 defence settings x 20,000 to 22,000 swings per class per side (8 classes), 100 swings per unit on its own seed; damage: the 20,883..20,971 hits of the first setting per class per side; drops: 3 configurations x 20,000 monsters per class per side (mf 0, mf 300, mf 0 with 0..40 extra draws on the seed before the drop) for 8 classes plus Andariel's TC4 (27 files per side, 540,000 monsters per side); control: 20,000 seed states per class x 2 magic-find values (9 classes, 360,000 drop cases) and x life (8 classes, 160,000).

## 3. Results (V run)

Tests: goodness of fit (GOF) = chi-square against the rule's exact distribution (df = cells - 1); "game ~ d2d" = two-sample chi-square of homogeneity (cells merged until the pooled count is 10 or more) and, for life, the two-sample Kolmogorov-Smirnov D (asymptotic p, conservative for discrete data); proportions: z test against the rule's chance and two-sample z, Wilson 95 percent interval in the script output; lag-1 = chi-square of independence on consecutive pairs of one unit (life: consecutive units of a game). Every p-value of the card is collected (section 3.7).

### 3.1 Life roll (rule: min + rand(max - min + 1)) and the seed-for-seed control

100,000 units per class per side. game = unit seeds as `FUN_00552df0` makes them, life through `FUN_0045c3e0`; d2d A = one world stream `{0x2545f491}` through the real `make_monster`; d2d B = 20 x 1000 on disjoint stretches of that stream (20,000 per class).

| class | life | game GOF p | d2d A GOF p | d2d B GOF p | game ~ A chi2 (df) p | KS D, p | game lag-1 p (all / gap-1 games) | A lag-1 p | game mean / var | A mean / var | rule mean / var |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Fallen | 1..4 | 0.912 | 0.436 | 0.183 | 0.96 (3) 0.81 | 0.0016, 0.999 | 0.921 / 0.979 | 0.726 | 2.499 / 1.251 | 2.494 / 1.252 | 2.500 / 1.250 |
| Zombie | 7..12 | 0.793 | 0.435 | 0.193 | 3.83 (5) 0.574 | 0.0023, 0.956 | 0.855 / 0.361 | 0.373 | 9.497 / 2.912 | 9.492 / 2.918 | 9.500 / 2.917 |
| Skeleton | 7..11 | 0.94 | 0.7 | 0.22 | 1.52 (4) 0.824 | 0.0018, 0.996 | 0.154 / 0.083 | 0.94 | 9.003 / 2.001 | 8.999 / 1.993 | 9.000 / 2.000 |
| Quill Rat | 1..5 | 0.835 | 0.738 | 0.982 | 0.48 (4) 0.975 | 0.0008, 1 | 0.708 / 0.559 | 0.0147 | 2.998 / 1.994 | 2.999 / 1.995 | 3.000 / 2.000 |
| Rogue (Dark Hunter) | 5..9 | 0.264 | 0.709 | 0.319 | 5.98 (4) 0.2 | 0.0027, 0.843 | 0.22 / 0.167 | 0.62 | 7.002 / 1.997 | 7.000 / 2.005 | 7.000 / 2.000 |
| Gargantuan Beast | 11..19 | 0.173 | 0.907 | 0.831 | 10.87 (8) 0.209 | 0.0055, 0.096 | 0.784 / 0.312 | 0.564 | 15.010 / 6.695 | 14.990 / 6.652 | 15.000 / 6.667 |
| Blood Raven | 113 | 100,000 of 100,000 equal | 100,000 of 100,000 equal | 20,000 of 20,000 equal | - | - | - | - | - | - | - |
| Andariel | 1024 | 100,000 of 100,000 equal | 100,000 of 100,000 equal | 20,000 of 20,000 equal | - | - | - | - | - | - | - |

- **Seed-for-seed control (the same unit seed states into both real codes):** life 160,000 cases (8 classes x 20,000 states, states with high word 666 and with a random high word), 0 differ in the life value for all 8 classes; the seed after the roll is equal in all 120,000 cases of the 6 rows with a range and DIFFERS in 20,000 of 20,000 for each of Blood Raven and Andariel (min = max: game.exe still steps, d2d's `range` does not; D4). First drop: 9 classes x 2 magic-find values x 20,000 = 360,000 cases (class, quality, count and the seed after all in the compared line), 0 differ.
- **Cross-game structure (D2, V run):** 20 d2d games, each a fresh world stream with equal draw history, 1000 monsters each: 20 of 20 sequences identical to game 0 for every class (the "C" runs). The B stretches above differ because they start at different offsets.
- **Reading:** both sides are uniform on the row range (smallest GOF p 0.173 over 18 tests), equal to each other (smallest two-sample p 0.2 for game ~ A, 0.10 for game ~ B), unbiased in mean and variance (the means sit within 0.02 of the rule's in every class), with no serial dependence beyond chance (smallest lag-1 p 0.0147 over 18 tests: a minimum that small turns up in about 23 percent of null runs).

### 3.2 Hit roll vs defence (rule: chance = AR x 100 / (AR + def) x 2 alvl / (alvl + dlvl), clamped 5..95; hit when the unit seed rolls below it)

Defender at defence 0 / level 1, defence 20 / level 2, defence 120 / level 12 and defence 400 / level 30 (the last repeats the 120 row where the chance sits at the 5 percent floor: same seeds). Per cell: swings 20,000 (22,000 for the first setting) per class per side, `FUN_0057d9b0` (real) against the real `monster_blow`. Cell text: rule % / game % / d2d % (p of game ~ d2d, two-sample z).

| class | def 0, lvl 1 | def 20, lvl 2 | def 120, lvl 12 | def 400, lvl 30 |
|---|---|---|---|---|
| Fallen | 95 / 95.15 / 94.95 (0.356) | 18 / 18.43 / 17.77 (0.084) | 5 / 5.21 / 4.78 (0.048) | same as 120 |
| Zombie | 95 / 95.17 / 94.84 (0.11) | 18 / 18.02 / 17.94 (0.825) | 5 / 5.03 / 5.34 (0.162) | same as 120 |
| Skeleton | 95 / 94.92 / 95.02 (0.647) | 37 / 37.03 / 36.67 (0.456) | 5 / 4.82 / 4.89 (0.745) | same as 120 |
| Quill Rat | 95 / 94.93 / 94.95 (0.931) | 18 / 17.93 / 18.20 (0.499) | 5 / 5.00 / 5.25 (0.247) | same as 120 |
| Rogue (Dark Hunter) | 95 / 95.11 / 95.05 (0.791) | 37 / 37.48 / 36.57 (0.061) | 5 / 5.16 / 4.91 (0.262) | same as 120 |
| Gargantuan Beast | 95 / 95.32 / 95.17 (0.446) | 37 / 37.48 / 37.26 (0.649) | 5 / 5.16 / 4.92 (0.283) | same as 120 |
| Blood Raven | 95 / 95.00 / 95.02 (0.913) | 95 / 94.98 / 95.12 (0.504) | 59 / 59.13 / 59.25 (0.807) | 18 / 18.01 / 18.00 (0.979) |
| Andariel | 95 / 95.03 / 95.02 (0.983) | 95 / 95.03 / 94.97 (0.748) | 58 / 58.63 / 57.65 (0.047) | 16 / 15.94 / 16.04 (0.774) |

All 32 game z tests against the rule: smallest p 0.028 (Gargantuan Beast, first setting; 1 of 32 below 0.05, 1.6 expected); all 32 d2d z tests: smallest p 0.027 (Zombie, a repeated row: 2 of 32 below 0.05, one a duplicate); game ~ d2d: smallest 0.047 (3 of 32 below 0.05, one a duplicate of another; 1.6 expected, the repeated rows are not independent); lag-1 independence of consecutive swings: smallest p 0.0105 (Blood Raven, defence 400). Hit rates differ by at most 1.0 percentage point (Andariel, defence 120: 58.63 vs 57.65, p 0.047) and by no more than 0.9 elsewhere, in both directions (d2d lower in 16 of the 26 distinct cells, counting by hand from the table). No deviation. The to-hit formula itself is S: monsterhit-trace.md (100,000 cases, 0 differ against the port).

### 3.3 Physical damage of a hit (D3)

Hits of the first setting (95 percent chance): about 20,900 per class per side. Game: `FUN_0057b420` (real), 256ths; d2d: `taken.physical` of the real `monster_blow`, whole points.

| class | A1 min..max | game mean hp / var | game GOF p (1/16 hp bins, uniform on [min, max)) | d2d mean / var | d2d GOF p (uniform on min..max) | mean diff d2d - game (z p) | var ratio d2d / game | KS D (p) | range seen: game 256ths of its bounds / d2d hp |
|---|---|---|---|---|---|---|---|---|---|
| Fallen | 1..2 | 1.4975 / 0.0828 | 0.877 | 1.4979 / 0.2500 | 0.552 | +0.0005 (0.904) | 3.02 | 0.498 (0) | 256..511 of 256..512 / 1..2 |
| Zombie | 1..3 | 2.0053 / 0.3331 | 0.426 | 1.9972 / 0.6649 | 0.762 | -0.0081 (0.242) | 2.00 | 0.332 (0) | 256..767 of 256..768 / 1..3 |
| Skeleton | 1..3 | 1.9975 / 0.3341 | 0.788 | 2.0114 / 0.6656 | 0.123 | +0.0139 (0.044) | 1.99 | 0.339 (0) | 256..767 of 256..768 / 1..3 |
| Quill Rat | 1..2 | 1.4932 / 0.0835 | 0.033 | 1.4973 / 0.2500 | 0.442 | +0.0041 (0.305) | 2.99 | 0.499 (0) | 256..511 of 256..512 / 1..2 |
| Rogue (Dark Hunter) | 1..3 | 2.0004 / 0.3341 | 0.456 | 2.0029 / 0.6650 | 0.759 | +0.0025 (0.713) | 1.99 | 0.334 (0) | 256..767 of 256..768 / 1..3 |
| Gargantuan Beast | 2..3 | 2.4957 / 0.0829 | 0.745 | 2.5005 / 0.2500 | 0.885 | +0.0048 (0.229) | 3.01 | 0.501 (0) | 512..767 of 512..768 / 2..3 |
| Blood Raven | 4..6 | 4.9994 / 0.3356 | 0.529 | 5.0065 / 0.6647 | 0.434 | +0.0071 (0.304) | 1.98 | 0.336 (0) | 1024..1535 of 1024..1536 / 4..6 |
| Andariel | 6..19 | 12.5585 / 14.0875 | 0.095 | 12.5131 / 16.3700 | 0.337 | -0.0455 (0.234) | 1.16 | 0.071 (9e-47) | 1536..4863 of 1536..4864 / 6..19 |

Each side follows its own rule exactly (game uniform on [min, max) in 256ths: smallest GOF p 0.033 over 8; d2d uniform on min..max: smallest 0.123). The means agree within 0.05 hp (no difference significant after the 8 tests: smallest p 0.044). The shape differs as the rule says: game.exe never reaches the bound (maximum seen 767 of 768), d2d reaches it; d2d's variance is 2.0 to 3.0 times game.exe's for the 2-wide and 3-wide rows (continuous vs two or three whole values), 1.16 for Andariel; KS D 0.33 to 0.50 is the integer-vs-continuous gap, not a shift of the centre.

### 3.4 First drop (class pick, quality, item count; mf 0, mf 300, mf 0 with 0..40 extra draws on the seed)

20,000 monsters per class per configuration per side; the outcome is the whole list of the drop (item code and quality, gold with its multiplier; "-" = nothing), cells merged until pooled 10 or more. Game: `FUN_0055a6d0` (real); d2d: real `tc_upgrade` + `roll_drops` on the unit seed as `spawn_monsters` leaves it.

| class | config (mf, extra draws) | NoDrop % game / d2d | outcome chi2 (df) p | quality p | item-count p |
|---|---|---|---|---|---|
| Fallen | 0, 0 | 62.56 / 62.67 | 87.1 (88) 0.506 | 0.452 | 0.812 |
| Fallen | 300, 0 | 62.56 / 62.67 | 108.9 (98) 0.212 | 0.687 | 0.812 |
| Fallen | 0, 0..40 | 62.69 / 61.86 | 94.3 (83) 0.187 | 0.153 | 0.089 |
| Zombie | 0, 0 | 62.54 / 62.47 | 90.7 (85) 0.315 | 0.407 | 0.877 |
| Zombie | 300, 0 | 62.54 / 62.47 | 100.7 (103) 0.546 | 0.503 | 0.877 |
| Zombie | 0, 0..40 | 62.52 / 61.81 | 103.6 (84) 0.072 | 0.776 | 0.146 |
| Skeleton | 0, 0 | 62.57 / 62.35 | 82.1 (84) 0.538 | 0.293 | 0.657 |
| Skeleton | 300, 0 | 62.57 / 62.35 | 102.1 (103) 0.505 | 0.423 | 0.657 |
| Skeleton | 0, 0..40 | 62.32 / 63.10 | 66.5 (85) 0.931 | 0.568 | 0.107 |
| Quill Rat | 0, 0 | 76.41 / 76.70 | 18.8 (14) 0.174 | 1 | 0.486 |
| Quill Rat | 300, 0 | 76.41 / 76.70 | 18.8 (14) 0.174 | 1 | 0.486 |
| Quill Rat | 0, 0..40 | 76.79 / 76.63 | 18.0 (14) 0.209 | 1 | 0.705 |
| Rogue (Dark Hunter) | 0, 0 | 62.81 / 62.97 | 106.1 (84) 0.052 | 0.427 | 0.74 |
| Rogue (Dark Hunter) | 300, 0 | 62.81 / 62.97 | 104.3 (101) 0.392 | 0.027 | 0.74 |
| Rogue (Dark Hunter) | 0, 0..40 | 62.48 / 62.37 | 85.2 (85) 0.473 | 0.746 | 0.82 |
| Gargantuan Beast | 0, 0 | 62.41 / 62.31 | 85.9 (85) 0.452 | 0.954 | 0.828 |
| Gargantuan Beast | 300, 0 | 62.41 / 62.31 | 102.3 (104) 0.528 | 0.996 | 0.828 |
| Gargantuan Beast | 0, 0..40 | 62.35 / 62.59 | 68.6 (81) 0.836 | 0.523 | 0.627 |
| Blood Raven | 0, 0 | 0.30 / 0.35 | 270.8 (285) 0.717 | 0.43 | 0.135 |
| Blood Raven | 300, 0 | 0.30 / 0.35 | 271.5 (278) 0.599 | 0.347 | 0.106 |
| Blood Raven | 0, 0..40 | 0.26 / 0.28 | 225.0 (265) 0.964 | 0.058 | 0.398 |
| Andariel | 0, 0 | 0.02 / 0.02 | 8.1 (8) 0.423 | 0.288 | 0.735 |
| Andariel | 300, 0 | 0.02 / 0.02 | 8.1 (8) 0.423 | 0.034 | 0.202 |
| Andariel | 0, 0..40 | 0.01 / 0.01 | 7.2 (5) 0.208 | 0.94 | 0.628 |
| Andariel TC4 (quest class) | 0, 0 | 0.40 / 0.46 | 45.8 (50) 0.642 | 0.362 | 0.36 |
| Andariel TC4 (quest class) | 300, 0 | 0.40 / 0.46 | 51.2 (56) 0.658 | 0.933 | 0.71 |
| Andariel TC4 (quest class) | 0, 0..40 | 0.50 / 0.54 | 60.9 (55) 0.273 | 0.090 | 0.924 |

(The mf 0 and mf 300 rows share unit seeds, so mf 300 changes only the quality draws: the NoDrop columns repeat. Blood Raven's and Andariel's own class mostly drop gold piles, so their outcome strings are gold counts; the quest class of Andariel is the one with items.)

- **No deviation in the 27 outcome tests** (smallest p 0.052; of all 81 drop p-values of the configurations, 2 are below 0.05 where 4 are expected).
- **Independent per-monster check:** the first item of each drop, mf 0, no extra draws, 9 classes: 101,996 game and 101,938 d2d items over 123 codes, chi-square 139.0 (123) p 0.154; per class p from 0.026 (Rogue) to 0.90.
- **Pooled quality (all items, every configuration):** game 850,966 and d2d 849,915 items; quality chi-square 8.86 (6) p 0.182. Items by quality, game / d2d: low 13,143 / 13,238; normal 395,124 / 393,935; superior 2,241 / 2,388; magic 123,276 / 123,316; set 97,926 / 98,261; rare 176,091 / 176,039; unique 43,165 / 42,738 (the Act 1 bosses and the quest class dominate the non-normal items). The pooled item-code chi-square (282.7 on 123 df) is NOT a p-value to read: the items of one drop are correlated, mf 0 / mf 300 repeat the same seeds and Andariel / Andariel TC4 share seeds; its largest cells are potions and scrolls at 5 percent apart in counts of 8,000 (diagnostic only), while every independent test above is clean.

### 3.5 Seeds per class and what a "game" is

Game seed of game g and class row r (both sides, different constants): `map seed = k1 (g + 1) + c + r x 0x1000193`, 20 games per class for life and drops (1000 monsters each), 20 per class for swings (20 monsters each). No two classes share unit seeds after the fix; Andariel and Andariel TC4 share (same row).

### 3.6 Real room population of game.exe (in situ, V run)

Does the unit-seed model above (my `sd_game.py` generator) describe what the game really does? Upstream's room population (`monsters.py`, `objgroups.quest_stubbed`) runs for real with `FUN_00573cb0` itself making every unit, the stat setter hooked (`ms_spawn.py`, the earlier monsterstats card's driver, run by `sd_spawn.sh`): map seeds 1..200 on Blood Moor, Cold Plains, Stony Field, Dark Wood, Den of Evil and Cave 1 (Levels ids 2, 3, 4, 5, 8, 9), seeds 1..100 on Burial Grounds and Catacombs level 4 (ids 17, 37) for the two bosses, Normal, one player. The life stored in stat 7 (256ths, shifted) of every plain unit at the row's own level:

| class | units | GOF vs uniform p | mean / rule mean | seen | levels seen |
|---|---|---|---|---|---|
| Fallen | 46,473 | 0.299 | 2.491 / 2.500 | 1..4 | 1 and 4 (level-4 units are not counted: another range) |
| Zombie | 7,088 | 0.176 | 9.524 / 9.500 | 7..12 | 1 |
| Skeleton | 13,014 | 0.77 | 9.000 / 9.000 | 7..11 | 2 |
| Quill Rat | 4,191 | 0.143 | 2.981 / 3.000 | 1..5 | 1 |
| Rogue (Dark Hunter) | 9,546 | 0.033 | 7.021 / 7.000 | 5..9 | 2 |
| Gargantuan Beast | 2,837 | 0.127 | 15.039 / 15.000 | 11..19 | 2 |
| Blood Raven | 100 | 100 of 100 equal 113 | - | 113 | 10 |
| Andariel | 100 | 100 of 100 equal 1024 | - | 1024 | 12 |

The real units follow the same uniform law (smallest p 0.033 over 6 tests, chance level; every mean within 0.04 of the rule's). The in-situ sample is a check of the model, not of d2d: the d2d population of the same levels was compared on stats by the earlier card (monster-stats-trace.md R3, 85,213 units, life inside the range). The bosses here are the base row at the maker; their boss layer is a later call (U for this card).

### 3.7 Multiplicity

Every p-value of the card (life 48, hit 160 including the repeated rows, damage 24, drops 92 = 81 configuration tests + 9 per-class first-item tests + the pooled first-item and pooled quality tests, in-situ 6; `sd_stats.py` collects them): **330 p-values, 16 below 0.05 (16.5 expected if all null), 0 below 0.01 (3.3 expected), none below 0.001, none below the Bonferroni line 0.05/330.** The smallest: hit Blood Raven defence 400 lag-1 0.0105, life Quill Rat d2d lag-1 0.0147, drop Rogue first item 0.0257, drop Rogue mf 300 quality 0.027, hit Zombie d2d z 0.0274, hit Gargantuan Beast game z 0.0281, damage Quill Rat game GOF 0.0326. The repeated rows (defence 120 and 400 at the 5 percent floor; mf 0 and 300 on the same seeds; Andariel and its TC4) are counted as separate tests, which makes the multiplicity count slightly generous in n and the chance level slightly lumpy; reading it as 330 independent tests changes nothing.

## 4. Deviations (D1..): d2d vs game.exe

| id | game.exe | d2d | tag | differential |
|---|---|---|---|---|
| D1 | life roll on the unit seed, first or second draw (look first: `0x573f83` after the look at `FUN_005739d0`) | life on the world stream (ai.cpp:2744); the unit seed is not drawn for life, so d2d's drop seed is one draw behind game.exe's | V read both; V run: distributions equal (section 3.1) | built: `sd_*` (this card): control (same states, 160,000 cases, 0 differ) and distribution (100,000 per side per class) |
| D2 | the unit seed depends on the map seed and every unit made | world stream fixed start `0x2545f491` in every game (world.hpp:295); equal draw history gives equal lives | V read; V run: 20 of 20 games identical (section 3.1) | built (the C runs); by design of the single stream, no distribution effect |
| D3 | physical roll `min << 8 + rand((max - min) << 8)`, top excluded, 256ths | `rng.range(min, max)`, whole points, inclusive (combat.hpp:414) | V run: variance ratio 1.16..3.0, mean within 0.05 hp (section 3.3); S: monster-attack-trace.md | built: `sd_*` damage runs and `monster_hit.py` |
| D4 | a fixed-life row (min = max: Blood Raven, Andariel) still steps the unit seed (`rand(1)`: n = 1 is a power of two, the new low is stored) | `Rng::range(min, max)` with max == min draws nothing | V run: seed after differs in 20,000 of 20,000 cases for each of the two rows (control) | built: the control; no effect today because d2d does not roll life on the unit seed; matters if D1 is ever changed to unit-seed life |

## 5. Bot findings (B1..), described, not changed

| id | finding | file:line | tag |
|---|---|---|---|
| B1 | `MeanHP` = (scaled min + scaled max) / 2 equals the mean of game.exe's life roll: V run game means 2.499, 9.497, 9.003, 2.998, 7.002, 15.010 against rule means 2.5, 9.5, 9.0, 3.0, 7.0, 15.0 (100,000 units each); consistent on both backends (d2d means within 0.02 as well). No change needed | internal/data/db.go:420-423 | V run |
| B2 | `ShooterPeak` and every "largest hit" bound use the inclusive whole-point max (`A1Max`); game.exe's largest hit is strictly below max (observed below, section 3.3), d2d's reaches max: the bound is conservative by less than 1 hp on game.exe and exact on d2d | internal/combat/threat.go:148-160 | V run (observed ranges), I (consumer effect) |
| B3 | The player-side damage model `Blow.MeanDamage` / `Swing` is written for game.exe's rule (uniform on [a, b) in 256ths, mean `a + (b - a - 1) / 2`), the same rule this card ran for monsters; on d2d the monster side is whole-point inclusive (D3), so expected damage taken is +1/512 hp per hit apart (negligible) but variance per hit is up to 3 times | internal/resource/melee.go:105-129 | I (player side not run here) |
| B4 | No bot code assumes a seed-for-seed life or drop sequence on either backend; the A:B plan in tierb-design-20261102.md (statistical, interleaved arms) is supported by D1/D2: nothing in the outcome distributions needs per-unit seeds | docs only | I |

## 6. PR candidates

None. A candidate needs a d2d deviation proven by the existing differential (mismatches before > 0, after the fix 0). D1 and D2 change no distribution (nothing to fix); D3 is carried since the monster-attack trace and changes a design decision (continuous 256ths hit points on d2d), not a defect found here; D4 has no effect. Nothing built on a local branch.

## 7. Matrix-cell updates (proposed text, no shared file edited)

- `docs/coverage/ledger.tsv` (done in this card, row key F16): `formulas	F16	TESTED-0-DIFF	docs/research/seeddist-trace.md	360000	2026-11-02	treasure class pick and item quality: 360,000 seed-for-seed cases (9 classes x mf 0 / 300 x 20,000 unit seed states) game.exe FUN_0055a6d0 vs d2d roll_drops 0 differ; plus 540,000 first drops per side with the seed supply of each side, distributions equal (section 3.4)`. Other rows: the existing `monster.stats` line (all, 242,220 cases) is not touched; add to its note on the next edit: "life roll distribution game.exe unit seed vs d2d world stream equal (seeddist-trace.md)".
- `docs/research/seed-study.md` section 6, blocker B2 ("d2d's world RNG is a single fixed stream, not game.exe's per-unit seeds"): add: "for outcome DISTRIBUTIONS (life, hit, damage mean, first drop) B2 does not matter: seeddist-trace.md, 100,000 units per class per side, no deviation; it still blocks seed-for-seed lockstep". Section 8 card 6 (d2d unit-seed fidelity): lower to "optional, only for lockstep (P3c)".
- `docs/research/tierb-design-20261102.md` section 1: add a sentence under item 1: "the per-unit seed model of game.exe was tested against d2d's stream: outcome distributions equal (seeddist-trace.md), so the statistical A:B is not undermined by the RNG difference."
- `docs/research/monster-stats-trace.md` Q3 ("How is life rolled?"): add the tag V run for the roll itself: "equal on 100,000 units per class per side (seeddist-trace.md)".

## 8. Drill recipe (lab binary, docs/forkdbg.md, docs/forkcheats.md, one hidden slot, a throwaway Sorceress with `--no-save`; run later by d2-live, not here)

Confirms the d2d half of the V run on the real lab binary (U today):

1. **Life distribution, live (D1, D2):** `debug goto 2`, then `debug spawn fallen1 20` five times (each call after killing or leaving the previous ones is not needed: the monsters stay), read `monsters` after each: the lives of 100 fallen1 must lie in 1..4 with a flat histogram (chi-square against 25 per value, p > 0.01). Then restart the slot with `--no-save` and the same steps: the first `debug spawn fallen1 20` must list the same 20 lives as the first run (D2: equal draw history, fixed stream start); a spawn after a different number of earlier world-stream draws (e.g. open a vendor first) must list different lives.
2. **Hit roll:** `debug spawn zombie1 1`, `debug mon set <id> dmg 1 1` and a known defence on the player (`debug stat 31 <n>`, forkcheats.md); count the hits of 2,000 swings via the Meter totals: the hit rate must be the rule's value (section 3.2 table, 18 to 37 percent at defence 20, level 2).
3. **Damage roll (D3):** `debug mon set <id> dmg 1 2` on a fallen1 and read the life lost per hit on `dump stats` (256ths): d2d shows only 256 or 512 (whole points); game.exe shows 256..511. The live game.exe side needs the original-game lab instance and is not in this recipe.
4. **First drop:** kill 200 `debug spawn fallen1` in Act 1 with no magic find and tally `ground` after each: NoDrop about 62.6 percent, gold about 13 percent, hp1 about 2.6 percent (section 3.4, game.exe and d2d equal).

## 9. Dead ends and limits

- The first sample set drew every class from the same map seeds, so every class shared its unit seeds and several classes showed identical hit counts; that is a design error (not independent samples). Fixed: the class row is added to the map seed on both sides; all numbers in this doc are from the fixed runs.
- A first d2d stream-B design advanced game g by only 31 draws, so the 20 stretches overlapped and the pooled goodness of fit failed (Quill Rat p 3e-8): an artefact of overlapping data, not a deviation. Redone with disjoint stretches (advance 100,003 draws).
- The first control read the seed file with the wrong field count (life differed in about 75 percent of cases): a bug of this card's harness, fixed; the control above is the corrected run.
- Limits: (1) the gap model (1 or 1..5 extra steps) and the look set count (3) are assumptions (I); game.exe's real gaps depend on what else is made. (2) Bosses: only the MonStats base row was rolled; Blood Raven and Andariel get boss mods and a boss level in game.exe (`FUN_005a2120`, monster-mods-trace.md) and their drops come from the superunique row / quest class; the first-drop test for them uses the MonStats TreasureClass columns (`Blood Raven`, `Andariel`, `Andarielq`) at the row's level. (3) AI draws between spawn and death are modelled as 0 or 0..40 extra draws. (4) Magic-find enters only through the killer stat the test hooks (0x5585d0 for game.exe, the `magic_find` argument for d2d). (5) Players = 1, Normal difficulty only; Nightmare and Hell take the area level (monster-stats-trace.md D1) and were not run. (6) The d2d side is the scratch upstream build (main 7d5b3a5c libraries), not the lab fork binary; the lab binary needs the recipe above. (6b) The d2d file:line cites in this doc come from the lab checkout of upstream main, which is newer than the scratch build the runs used; the functions the runs call (`make_monster`, `monster_blow`, `tc_upgrade`, `roll_drops`) are the same ones, but that the two source states agree on them is U (the earlier traces ran the same scratch build). (7) Power: a chi-square cannot see a deviation smaller than about 1 percent of a category at n = 20,000 (a rare quality at 0.1 percent of drops is invisible); the pooled quality table in section 3.4 states how many non-normal items were seen. (8) The hit-roll runs for defence 120 / level 12 and 400 / level 30 hit the 5 percent floor for most classes, so those two rows repeat each other (same seeds).

## Shared-file lines for the merger

See `openspec/changes/seeddist/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG).
