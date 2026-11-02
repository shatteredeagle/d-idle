# Curse value calcs: game.exe 1.14d vs d2d vs the bot (SKCURSEVAL, board item 16)

> status: card skcurseval, 2026-11-02, first pass, offline (no game, no d2d slot, nothing pushed or posted). Follows `sk-resist-trace.md` section 10 ("the curse value calcs were not differentialled") and Q7.

Tags: **V read** the game.exe 1.14d disassembly read by this agent (addresses given, no code pasted); **V run** game.exe's own code run under the emulator on the worker host and compared with d2d's real `eval_calc`; **D** the patched 1.14d tables (Skills.txt, States.txt, DifficultyLevels.txt) as game.exe's MPQ stack serves them; **S** a secondary source (named); **I** inferred; **U** unverified.

## Answer in five lines

1. **d2d's calc evaluator gives game.exe's value for every cell of the curse rows at every skill level 1..99 (V run): 0 of 46,136 (seed 1) and 0 of 66,136 (seed 7) cases differ**, md5 of the two output files equal in each run. 11 rows: Amplify Damage 66, Dim Vision 71, Weaken 72, Iron Maiden 76, Terror 77, Life Tap 82, Decrepify 87, Lower Resist 91 (curse do, srvdofunc 30), Conviction 123 (aura, srvdofunc 66), Defense Curse 309, Blood Mana 310. Cells: calc1..4, aurarange, auralen, perdelay, aurastatcalc1..6, passivecalc1..5, petmax, sumsk1..5, delay.
2. **The values** (V run, table in section 2): Lower Resist `-31` at level 1, `-47` at 5, `-55` at 10, `-62` at 20, flat `-70` from level 60 (the dm formula saturates, I); Conviction resists `-(30 + 5 (L-1))` capped by `min(ln34, 150)` at **-150 from level 25** (level 24 is -145), plus `skill_armor_percent` -49 at 1 rising to -100 at 60; Amplify Damage `damageresist` **-100** and Decrepify **-50** at every level (only length and radius grow); Weaken `damagepercent` -33 at every level; Defense Curse `-(50 + 5 (L-1))` armor %.
3. **There is no synergy in any curse row** (V run): the same cells print whatever the other 10 curse rows' levels (0 / 20 / 60) and the masteries (0 / 50) are, 0 of about 8,700 grid groups vary. The only input is the skill's own level (no caster level, no masteries).
4. **The level fed to the calcs is the level the do function is given** (V read, FUN_005c37c0: range, length and the six aurastat calcs are all FUN_00646ca0(caster, calc, skill, level argument)), so a curse value changes only with the casting skill level, never with the target. The 1/5 rule on immunes (SKRESIST D1) is applied afterwards, per target (FUN_005c35c0 -> FUN_005c3540), and stays the one proven curse deviation.
5. **No new d2d deviation, no PR candidate.** One bot finding (B1): the bot picks Lower Resist against magic-resistant targets; no 1.14d curse touches magic resistance.

## 1. Questions answered (Q1..)

| id | question | answer | where | tag |
|---|---|---|---|---|
| Q1 | Does d2d's `eval_calc` equal game.exe's calc machine on the curse rows at every level 1..99, including the cap? | Yes: 0 differences. Grid 26,136 cases = 11 rows x 99 levels x clvl {1, 30, 60, 99} x masteries {0, 50} x other-curse levels {0, 20, 60}; plus random cases (levels 1..99, clvl 1..99, masteries 0..300 or 0, 0..8 other skill levels 0..60): N = 20,000 seed 1 (46,136 total), N = 40,000 seed 7 (66,136 total). Conviction's cap (`min(ln34,150)`) is inside the compared range: the cell is -145 at L24 and -150 at L25..99 on both sides | `tools/emu/sk_curseval.py`, FUN_00646ca0, d2d `eval_calc` | V run |
| Q2 | Where does game.exe evaluate the curse calcs and with what level? | The curse do FUN_005c37c0 (table 0x7322b0, srvdofunc 30) checks the skill row (record stride 0x23c at +0xb98, aurastat1 id at +0x54, auratargetstate at +0x82), then calls FUN_00646ca0 for the range calc (record +0x64), the length calc (+0x60) and up to six aurastat calcs (+0x68 + 4k), each as (caster unit, calc, skill id, the level argument of the do). The values go into a stack block with the skill ids and FUN_0056e780 is called with the callback FUN_005c35c0 (I: an area search calling it per unit, d2d's comments say the same). The calcs are evaluated once per cast, before the callback, not per target | 0x5c3862, 0x5c3877, 0x5c391f (the three calls), 0x5c3996..0x5c399e (callback 0x5c35c0 into FUN_0056e780) | V read |
| Q3 | What does the target callback do with the values? | FUN_005c35c0: the target must be a unit of type 1 (monster) and FUN_006259b0 must not return 1 (meaning not read: U); a state-dependent test FUN_0056e2f0 (kind 10 for state 23, 12 for state 27, 11 for state 56, else 0) gates the target; the FIRST aurastat value goes through FUN_005c3540 (the 1/5 rule, SKRESIST Q6) and a result of 0 means no curse; else the state is put with the length (FUN_0056e970) and every aurastat value is adjusted by FUN_005c3540 again and written into the state's stat list (FUN_00627150) | 0x5c35c0..0x5c371b | V read (the 1/5 rule itself: V run in SKRESIST) |
| Q4 | Which curse rows have synergy terms? | None. Cells of every curse row are identical across 0 / 20 / 60 levels of the other curse rows and masteries 0 / 50 | `sk_curseval_tbl.py` (grid check) | V run |
| Q5 | The length divisor on Dim Vision and Terror | FUN_005c3400(state): 1 when the state is 0x17 (dimvision, D) or 0x38 (terror, D), else 0. When 1, the evaluated length goes through FUN_005c37a0: divide (cdq / idiv, toward 0) by the difficulty row's dword at +0x1c (row index = a byte at +0x6d of the caster record passed in; row size 0x58 via FUN_00611d30). DifficultyLevels.txt AiCurseDivisor is 1 / 2 / 4 for Normal / Nightmare / Hell (D); that the +0x1c dword is that column and the +0x6d byte the game difficulty is I (matches d2d's own note, `gamedata.hpp:500`, and the column order) | 0x5c3400, 0x5c37a0, 0x611d30, call 0x5c388b | V read + D, I for the two field names |
| Q6 | Do monsters' curses on players use the same do? | Not read in this card: this callback accepts only monster targets (type 1), so a curse on a player takes another path | FUN_005c35c0 head (0x5c35d6) | V read for the filter, U for the player path |
| Q7 | The numbers (cells at caster clvl 30, no synergy) | Section 2 | `sk_curseval_tbl.py` | V run |

## 2. Values by skill level (V run, game.exe; d2d identical)

Columns: aurarange (subtiles), auralen (frames, 25 per second), aurastat values in slot order. Ids of the stats are from the Skills.txt row (D).

| row | slots | L1 | L5 | L10 | L20 | L25 | L30 | L60 | L99 |
|---|---|---|---|---|---|---|---|---|---|
| Amplify Damage 66 | damageresist | -100 | -100 | -100 | -100 | -100 | -100 | -100 | -100 |
| (range / len) | | 3 / 200 | 7 / 500 | 12 / 875 | 22 / 1625 | 27 / 2000 | 32 / 2375 | 62 / 4625 | 101 / 7550 |
| Weaken 72 | damagepercent | -33 | -33 | -33 | -33 | -33 | -33 | -33 | -33 |
| (range / len) | | 9 / 350 | 13 / 590 | 18 / 890 | 28 / 1490 | 33 / 1790 | 38 / 2090 | 68 / 3890 | 107 / 6230 |
| Decrepify 87 | velocity, damage, damageresist, attackrate | -50 x4 | -50 x4 | -50 x4 | -50 x4 | -50 x4 | -50 x4 | -50 x4 | -50 x4 |
| (range / len) | | 6 / 100 | 6 / 160 | 6 / 235 | 6 / 385 | 6 / 460 | 6 / 535 | 6 / 985 | 6 / 1570 |
| Lower Resist 91 | fire, light, cold, poison (equal) | -31 | -47 | -55 | -62 | -64 | -65 | -70 | -70 |
| (range / len) | | 7 / 500 | 11 / 700 | 16 / 950 | 26 / 1450 | 31 / 1700 | 36 / 1950 | 66 / 3450 | 105 / 5400 |
| Conviction 123 | skill_armor_percent / fire = cold = light | -49 / -30 | -70 / -50 | -80 / -75 | -90 / -125 | -92 / **-150** | -94 / -150 | -100 / -150 | -100 / -150 |
| (range / len / period) | | 20 / 0 / 50 | same | same | same | same | same | same | same |
| Defense Curse 309 | skill_armor_percent | -50 | -70 | -95 | -145 | -170 | -195 | -345 | -540 |
| Iron Maiden 76 | calc1 / calc2 = calc3 (percent) | 200 / 50 | 300 / 75 | 425 / 106 | 675 / 168 | 800 / 200 | 925 / 231 | 1675 / 418 | 2650 / 662 |
| Life Tap 82 | calc1 | 50 | 50 | 50 | 50 | 50 | 50 | 50 | 50 |
| Dim Vision 71, Terror 77, Blood Mana 310 | (no aurastat; length 175 / 200 / 200 at L1) | | | | | | | | |

Lower Resist's per-level step shrinks (-6 from L1 to L2, then smaller, flat -70 from L60): the `dm56` term is a saturating level function (I; the exact bracket constants are in Skills.txt, not decoded here). Conviction's resists follow `30 + 5 (L-1)` until the 150 cap (I from the values: -30, -35, ..., -145, -150). Full by-level table: `tools/emu/run-on-worker.sh sk_curseval_tbl.py` after a run (levels 1, 2, 5, 10, 15, 20, 24, 25, 26, 30, 40, 60, 99 per row).

Consequence on a monster whose base resistance is 100 or more (I, arithmetic with the SKRESIST 1/5 rule): Lower Resist L20 lowers it by 62 / 5 = 12 (truncated toward 0), so a base of 100 to 111 drops under 100 and becomes hittable, a base of 112 or more stays immune; Conviction L20 lowers fire / cold / lightning by 125 / 5 = 25 (bases 100..124 drop under 100). Whether a build reaches those skill levels (with +skills) is the build's matter, not this card.

## 3. Deviation list (d2d vs game.exe)

| id | rule: game.exe | rule: d2d | tag | differential |
|---|---|---|---|---|
| D0 | curse calc cells (value, length, range, period, other slots) at levels 1..99 | `eval_calc` on the same Skills.txt | **no deviation**, V run 0 of 46,136 and 0 of 66,136 cases | exists, this card (`sk_curseval_run.sh`) |
| D1 | (inherited from SKRESIST D1) a negative curse value on a monster whose base resistance >= 100 is divided by 5 at apply time | full value (`Fight::apply_target_stats`, fight.cpp:3152) | V run on the real build in sk-resist-trace.md (88,101 of 147,628) | exists (`sk_resist.py` + `test_resist`) |
| D2 | FUN_005c37c0 passes the evaluated length (after the divisor) on without a clamp that I saw before FUN_0056e780; whether FUN_0056e970 or the state timer has a minimum is not read | d2d floors the length at 25 frames (`std::max(length, 25)`, fight.cpp:1222 player casts, :4761 monster on player) | U | unreachable with 1.14d data if the shortest length divided is still above 25 frames (Dim Vision L1 at Hell 175 / 4 = 43, Terror L1 200 / 4 = 50; I from the section 2 lengths), so not worth a harness |
| D3 | Dim Vision / Terror length divisor: the AiCurseDivisor of the difficulty row, applied by FUN_005c37a0 when the state is 23 or 56 | the same rule for `kCurse` with auratarget dimvision / terror, plus Confuse and Attract (fight.cpp:1218..1221), difficulty from `game_difficulty` | V read (state set, divide), D (divisors 1 / 2 / 4); Confuse and Attract's own divide (FUN_005c3bcc) not read: U | buildable: call FUN_005c37a0 under the emulator with a difficulty table loaded like the game's and compare to d2d's `curse_divisor`; oracle = the table value (weak: both read the same table), so low value |
| D4 | the curse's effect on the target (state, stat list, AI swap) | monster fields (`monster.curse`, `apply_target_stats`, `set_kind`) | not compared (outside this card) | would need a full target run |

Only D1 is a proven d2d deviation; it is not mine.

## 4. Bot findings (described, not changed)

| id | finding | file:line | tag |
|---|---|---|---|
| B1 | `curseFor` picks Lower Resist when the target's magic resistance is >= 50 and physical below 50 (the comment says Teeth and Bone Spear are magic). Lower Resist's slots are fire, lightning, cold and poison resistance (V run, section 2; no 1.14d curse lowers magic resistance, SKRESIST Q7), so the cast never helps a magic-resistant target and costs mana. It fits elemental skills against fire / cold / lightning resistant packs | internal/combat/necro.go:358 (comment :350) | V run (slot list), I (the effect on play) |
| B2 | On a physical immune (Hell, 20 rows, SKRESIST) Amplify Damage's -100 is divided by 5 (-20, V run in SKRESIST), the pick does not model it | internal/combat/necro.go:358..367 | I |
| B3 | Curse handling is by state name only; the bot does not use the value table, so it cannot know that Lower Resist saturates at -70 and Conviction at -150 | internal/d2/missiles.go:55..95 (state flags), internal/combat/necro.go:44 | I |

## 5. PR candidates

**No PR candidate.** The d2d calc evaluator equals game.exe on every cell compared (0 mismatches before, nothing to fix); the one proven curse deviation (D1, 1/5 rule) belongs to SKRESIST and has its own follow-up (`pr-curseimmune-draft.md`).

## 6. Dead ends and limits

- The oracle's stat getter returns 0 for every stat but clvl and the four masteries (as `nec_calc.py`). The curse rows read only their own skill level, so the limit does not bite: V run, no cell varies with the other 10 rows' levels or the masteries. A +skills item bonus changes the LEVEL passed in, which this run covers by scanning all levels 1..99, but how d2d chooses that level for a curse cast (`Fight`'s `lvl`, `effect->level`) and what the do receives in game.exe from the dispatcher (skill base level + bonuses) were not compared (U; section 7 drill).
- d2d side: the test mode of the existing necskill differential (`nec_test.cpp` built as `test_sk_rest_calc` in a scratch build of upstream main); not a rebuild for this card (S, same harness as SORCREST / NECSKILL). The game side was verified with the same program text as nec_calc.py (FUN_00646ca0 via the emulator).
- Not run: the callback FUN_005c35c0 as a whole (target filters, AI swap, state put), the 1/5 rule again (done in SKRESIST), Confuse (do 61, FUN_005c3f20) and Attract (59), Slow Missiles / Inner Sight (do 6), monster curses on players, the length-25 floor (D2).
- Conviction (aura, do 66, FUN_005cf3a0) evaluates its calcs in the aura callback, not in the curse do read above; its calc machine is the same FUN_00646ca0 (V run here, and S pal-skill-trace.md), the call site's level argument was not re-read in this card (U).
- The 1.14d curse "dm56" semantic (damage bracket function) was inferred from values, not decoded.
- Dead end: reading `skill_armor_percent` as d2d's stat 171: not checked against ItemStatCost here; the d2d switch (fight.cpp:3161) uses 171 for a percent of armour class and 31 for flat; Conviction and Defense Curse list `skill_armor_percent`, so which stat number that is must match (U).

## 7. Drill recipe for the fork verbs (U to confirm later; nothing run)

Verbs: `debug skill`, `debug spawn`, `debug difficulty`, `debug goto`, `step N`, `monsters` (resistances per monster), `cmd skill` (docs/forkdbg.md, docs/forkcheats.md).
1. Level read-back: give a Necromancer clvl 30 Lower Resist at 1, 10, 20 (`debug skill`), spawn a Hell monster with a known fire resistance below 100, cast at it, `monsters`: the fire resistance must fall by -31 / -55 / -62 (U for the live level passed in, +skills).
2. Conviction cap: Paladin with Conviction 25 and 24 (with the right items or `debug skill`): fire resistance change on a Normal monster -150 / -145 (cap line).
3. Immune: same on a Hell monster with fire 110: expected 110 - 31 / 5 = 104 (SKRESIST D1; d2d shows 79).
4. Length: Dim Vision L1 on Normal / Nightmare / Hell, count frames until the state ends: 175, 87, 43 (D3; Hell 175 / 4).
5. Level source: cast Amplify Damage with an item +skills (oskill or all-skills) and read the state's level in the monsters verb to see whether d2d uses base or total level (U).

## 8. Matrix-cell updates (proposed text for sorc-skill-matrix / necro rows)

- sk-resist-trace.md Q7 / section 10, "the curse value calcs (dm56, ln34) on game.exe vs d2d (not run)": replace by "V run: equal at levels 1..99, 46,136 and 66,136 cases (docs/research/skcurseval-trace.md)".
- Matrix line "Fire Bolt / Fire Ball etc. (a) resist, immunity": append "curse values: Lower Resist -31..-70, Conviction -30..-150 (V run, skcurseval-trace.md)".
- Ledger: `skills.monster 309,310 TESTED-0-DIFF` (calc aspect, 4,752 grid cases).

## 9. Reproduce

```
tools/emu/run-on-worker.sh --ttl 3000 sk_curseval_run.sh 20000 1 > $D2_SCRATCH/out1.txt 2>&1   # 46,136 cases, DIFF 0, md5 0a9b37a1... both
tools/emu/run-on-worker.sh --ttl 3000 sk_curseval_run.sh 40000 7 > $D2_SCRATCH/out7.txt 2>&1   # 66,136 cases, DIFF 0, md5 06b5f673... both
tools/emu/run-on-worker.sh sk_curseval_tbl.py > $D2_SCRATCH/tbl.txt 2>&1                          # values table and the synergy check (8,991 groups, 0 vary)
```
`sk_curseval_run.sh` generates the cases (`sk_curseval.py gen N SEED`, random.Random(seed)), runs game.exe's FUN_00646ca0 (`sk_curseval.py game`, case format of `nec_calc.py`), runs d2d's `test_sk_rest_calc --calc` and diffs; the output prints the three line counts and `DIFF n`. The second run overwrites the first's output directory (the table script reads the latest).

## Shared-file lines for the merger

In `openspec/changes/skcurseval/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG). Decomp-queue row (proposed): `- NN Curse value calcs (SKCURSEVAL, 2026-11-02): Lower Resist, Conviction (cap -150 at L25), Amplify, Decrepify, Weaken, Defense Curse cells at levels 1..99 equal d2d, 0 of 46,136 / 66,136 cases; curse do evaluates calcs once per cast at the do's level; Dim Vision / Terror length / AiCurseDivisor read; no PR; bot B1 Lower Resist vs magic`. Glossary: no new term.
