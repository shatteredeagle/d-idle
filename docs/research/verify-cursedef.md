# Independent verification: SKCURSEVAL, SKDEFENDERS, SKSHIELD (first-pass traces)

> status: card verifycursedef, 2026-11-02, Opus verifier, offline (emulator on the worker host, no game, nothing pushed or posted). Time-boxed (60 min): SKCURSEVAL got an own run on both sides; SKDEFENDERS and SKSHIELD got an own READ of the key game.exe functions and of d2d, no own run (NOT RUN where said).

Tags: **V run** (own harness, counts and seeds given), **V read** (own disassembly read with upstream's `disasm.py`, addresses given; d2d source read at the commit named), **S/U** opinion or not checked. Candidates: docs/research/skcurseval-trace.md, skdefenders-trace.md, skshield-trace.md (origin/main ab755d74).

## Verdicts in short

| doc | verdict | main caveat |
|---|---|---|
| SKCURSEVAL | **CONFIRMED WITH CAVEATS** | the oracle IS game.exe's code (not re-derived arithmetic), but every curse cell depends only on the skill level: the clvl / mastery / other-curse grid dimensions are inert, so 46,136 / 66,136 cases are ~1,089 distinct evaluations; "curse value" means the calc cells only, not the effect |
| SKDEFENDERS | **HOLDS WITH LIMITS** (not re-run) | the 222,464 "0 mismatches" is game.exe vs the author's own reading (self-consistency), not vs d2d; "penalty and cap equal d2d" rests on a python line copy of d2d's panel, not on real d2d |
| SKSHIELD | **HOLDS WITH LIMITS** (not re-run) | the 382 scenarios are game.exe vs the author's model of its own reading; no d2d run; no ledger row to earn |

## 1. SKCURSEVAL

### Is the oracle real game.exe code? Yes (V read + V run)

- `FUN_00646ca0(unit, calc offset, skill, lvl)` (stdcall, `ret 0x10`) bounds-checks the offset against the calc buffer (`[0x744304]+0x40/+0x44`) and calls the VM `FUN_006c0bc0` with operand callback `FUN_00646be0` (-> `FUN_00646460(ctx, ...)`) and the function table at 0x745774 (count 7 at 0x7457ac). Own dump of that table (`vcd_tbl.py`): 8-byte entries min 0x6436e0, max 0x6436f0, rand 0x643700, skill 0x646c00, miss 0x643740, stat 0x643770, sklvl 0x646c60 (arg counts 2,2,2,2,2,2,3), the same order as d2d's `CalcFunction` enum. V read.
- The author's `sk_curseval.py` calls `nec_calc.run` -> `e.call(0x646ca0, ...)` natively; the d2d side `test_sk_rest_calc` (nec_test.cpp) calls the real inline `d2d::rules::eval_calc` (components/rules/skills.hpp:761 in the sorcrest checkout). **Unlike the SKIAS first pass, both sides run real code.** V read.

### Own harness (V run)

`tools/emu/vcd_calc.py` (game.exe) and `tools/emu/vcd_test.cpp` (d2d, built by `vcd_run.sh` against the sorcrest checkout's headers and build, the tree the author used). Deliberately different from the author's hooks: base level != level with items (`FUN_006442a0` flag 0 vs non-0), EVERY stat nonzero (seeded H(seed, stat)), stat() selectors told apart (`FUN_00625480` total, `FUN_006253b0` .base +1000, `FUN_00625500` .mod +2000, `FUN_00622560` stat 19), a skill the case does not list has no record (`FUN_00643810` -> 0), levels 0 and 100, 127, 128, 200, 255 included, other skills drawn from all 357 rows. Every stat and level read is logged.

| run | seed | cases (grid 11 rows x 105 levels + random) | cells differ |
|---|---|---|---|
| `run-on-worker.sh vcd_run.sh 500 424242` | 424242 | 1,655 | **0** |
| `run-on-worker.sh vcd_run.sh 6000 99` | 99 | 7,155 | **0** |

**Read log: empty for all 11 rows** in both runs: no curse calc ever calls the stat getter or the skill-level getter. d2d's compiled code (`vcd_test --dump`) shows why: every cell is built only from operands ln12 / ln34 / ln56 / dm-pairs / par5, constants, unary minus and min() (e.g. Lower Resist aura `4/5 21/0` = -dm56, Conviction `4/2 7/150 1/0 21/0` = -min(ln34, 150), Defense Curse `4/4 21/0`). V run.

### Claims

| claim (skcurseval-trace.md) | verdict | note |
|---|---|---|
| A1 0 of 46,136 / 66,136 differ | **VERIFIED** (own 0 of 8,810, also at levels 0 and 100..255) | but see the next row |
| case count as coverage | **inflated** | the only live input is (row, lvl): the clvl x masteries x other-curse grid multiplies identical evaluations (~24 per (row, lvl)); ~1,089 distinct inputs for levels 1..99 |
| A3 no synergy, only the skill's own level | **VERIFIED** (stronger: no stat or level read at all, V run log) | |
| "no deviation" not by construction? | **partly by construction** | the author's stat hook returns 0 for all but clvl / masteries and d2d's env.stat also returns 0 there; stat(x.base) / stat(x.mod) would go to the UNHOOKED `FUN_006253b0` / `FUN_00625500` on a zeroed unit -> 0 on both sides. Harmless here only because no curse cell uses stat() (own log); the same oracle must not be reused for rows that do |
| A2 value table | VERIFIED for the cells (0 diff) | the table itself not re-printed |
| A4 / Q2 / Q3 / Q5 (curse do FUN_005c37c0, target callback, length divisor) | NOT VERIFIED (not read in the time box) | these are the parts that make it a "curse value" claim; the run compares calc cells, not the applied effect |
| ledger `skills.monster 309,310 TESTED-0-DIFF 4752` | **earned for the calc aspect, number inflated** | 4,752 = 2 rows x 99 x 4 x 2 x 3 grid cases; distinct inputs are 198. Blood Mana (310) has NO aurastat calc at all (only range and len): its effect is not in a calc cell, so "TESTED-0-DIFF" says nothing about Blood Mana's behaviour. Recommend: note "calc cells only; 198 distinct inputs" |

### d2d-side note found on the way (V read, not a curse deviation)

game.exe's stat() (`FUN_00643770`, fastcall ecx = stat, edx = selector) dispatches: stat 0x13 -> `FUN_00622560(unit)`; selector 1 -> `FUN_006253b0`, 2 -> `FUN_00625500`, else `FUN_00625480`. d2d's `kCalcStat` calls `env.stat(left)` and ignores the selector and the stat-19 special. A deviation candidate for any calc using `stat('x'.base)` or `.mod`; not reached by the curse rows. U which rows use it.

## 2. SKDEFENDERS (own read; no own run)

- `FUN_0057be00` (own read, 0x57be00..0x57bf05): total = defender stat [desc+4] (`FUN_00625480`, ctx+0xc); attacker pierce [desc+0xc] (ctx+8) subtracted when total < 100 OR ctx+0x14 == 0; for ctx+0x14 == 0 and stat not 0x24 / 0x25: expansion (game +0x70 != 0) adds the DifficultyLevels penalty pointer, classic byte +0x6d 1 -> -20, 2 -> -50; total <= -100 -> -100 for every defender; total > 0 and not plain: cap = min(75 + max stat, 95), no max stat: 50 for 0x24 else 75; physical with attacker state 0x2f and `FUN_0063e990(defender)` -> 0. **Q4 VERIFIED (V read)**, magic (0x25) takes no penalty: VERIFIED.
- d2d at upstream main e36922ea (own read): `Character::panel` character.hpp:250-258 `clamp(sum + penalty, -100, min(75 + max, 95))`, `Fight::merc_fighter` fight.cpp:1927-1932 same, `monster_blow` combat.hpp:403-436 `phys x (100 - dr) / 100 - flat`, elements `resisted(roll, res) - mdr`, magic `roll - mdr`. **D1 (flat after resistance) and D2 (no magic resistance) VERIFIED as d2d code (V read)**; the counts 2,659 / 5,279 NOT RUN by me.
- "penalty and cap equal d2d" (ledger F04 note, A8): **HOLDS WITH LIMITS**: the d2d `effres` column is a python line copy (the doc says so in Honest limits); the real-build cross-check covers only the monster_blow columns. The clamp form is equivalent to game.exe's branches for the four elements (own read), so I expect it holds, but it is not a real-d2d result. S.
- Ledger F04 note "222,464 cases rule check on game.exe 0 mismatches": **earned as a self-consistency check** (game.exe vs the author's formula), **not a d2d comparison**; the note should say so. The row stays TESTED-DEVIATION on the strength of sk-resist-trace.md D1, fine.
- Q1, Q2, Q3, Q5..Q10, D3..D8, B1..B6: NOT VERIFIED (time box).

## 3. SKSHIELD (own read; no own run)

- `FUN_004c0d20` attach (own read, 0x4c0e00..0x4c0eb7): unusable (`FUN_006280a0(item, 0x4000)`) -> no attach; player (`[unit] == 0`) of a dual-wield class (`FUN_006235a0`): active 1, set to 0 only for a weapon (`FUN_00629bb0(item, 0x2d)`) that is not the primary (`FUN_0063bef0`); every other player and non-player: active 1; broken (`0x100`) skipped unless the stack argument is set, on BOTH paths (0x4c0e60 and 0x4c0e8e). **Summary 2 VERIFIED (V read)**, including D2's game side.
- Summary 1, 3, 4, 5 and the 382-scenario run: NOT VERIFIED by me. The 382 scenarios compare game.exe with the author's `model()` of its own reading: a self-consistency check, not a d2d differential. D1 (d2d drops non-weapon ED) rests on a d2d read (combat.hpp:105, :111-112), not a run.
- No ledger row for SKSHIELD (grep of docs/coverage/ledger.tsv): nothing to earn.

## Recommended actions

1. (medium) Ledger `skills.monster 309,310`: add "calc cells only, 198 distinct inputs; Blood Mana has no aurastat calc" or downgrade 310 to READ-ONLY.
2. (medium) skcurseval-trace.md A1: say the case count is grid multiplicity over inert inputs; drop or mark Q2..Q5 as V read only (no run of the curse do).
3. (medium) Ledger F04 note: "rule check" = game.exe vs the author's reading; d2d effres from a line copy.
4. (low) `nec_calc.py` / `skill_damage.py` oracles: hook `FUN_006253b0`, `FUN_00625500`, `FUN_00622560` too before using them on rows with stat(); add a d2d deviation candidate for the ignored stat() selector (U whether any row uses it).
5. (low) A follow-up verifier with time for an own run of `FUN_0057c6c0` (D1, D2 counts) and of the shield item maker (Summary 3).

Commands: `tools/emu/run-on-worker.sh vcd_tbl.py`; `tools/emu/run-on-worker.sh --ttl 900 vcd_run.sh N SEED` (builds `$D2_SCRATCH/vcd/vcd_test` once, writes cases / game / d2d tsv, prints DIFF and the read log).
