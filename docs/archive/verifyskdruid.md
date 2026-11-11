# Independent verification of SKDRUID (Druid skills 221..250, calc aspect)

> status: archived 2026-11-11 (TASK-311, docs/process/coordinator-efficiency.md rule 8): finished one-off report, no inbound link when archived; stub left at docs/research/verifyskdruid.md.

> status: card verifyskdruid, 2026-11-02, offline (emulator only: no game, no d2d slot, no Go change). Verifier of docs/research/skdruid-trace.md.

Tags: **V run** game.exe's own code under the emulator, compared with d2d's real `eval_calc`, counts given; **V read** read in the disassembly by this verifier; **D** patched 1.14d `Skills.txt` (read through `tools/emu/mpq-table.sh skills`); **I** inference; **U** not verified.

## 0. What was checked

- **Passes.** Only one SKDRUID document exists: docs/research/skdruid-trace.md (commit 7554a751, card archived as 2026-11-02-skdruid). It calls itself "first pass" but already contains the emulator runs (tools/emu/skdruid_*). No second-pass document or `agent/*` branch for the Druid item was found on main or among the local branches; this report therefore verifies that one document. If a second pass appears later it is not covered here.
- **Order.** Own disassembly reading and own model first (section 1), own harness and seeds (section 2), the authors' scripts and claims last (section 3).

## 1. Own reading of the game side (V read)

| Function | What it does (own reading) |
|---|---|
| FUN_00646ca0(unit, calcOffset, skill, lvl), stdcall 0x10 | Returns 0 when the calc buffer `[[0x744304]+0x40]` is null or `calcOffset >= [[0x744304]+0x44]` (so 0xffffffff, "no calc", gives 0 by the game's own guard). Else builds ctx {unit, skill, lvl} on the stack and calls the expression machine FUN_006c0bc0(buffer+off, size-off, operand callback 0x646be0, function table 0x745774, count [0x7457ac], &ctx). |
| 0x646be0 (fastcall: ecx operand, edx ctx) | Null ctx gives 0; else FUN_00646460(ctx.unit, operand, ctx.skill, ctx.lvl). |
| FUN_00646460(unit, operand, skill, lvl) | Bounds-checks the skill id against `[[0x744304]+0xba0]`, record = `[+0xb98] + id*0x23c`; jump table of 0x49 operand codes at 0x646abc (several cases call unit helpers such as 0x4e6ca0, 0x4cc7c0, 0x4efcb0 with the unit in edx). |
| FUN_00646c00 (skill('X'.op)) | Resolves the unit's record of skill X by FUN_00643810; if found, level = FUN_006442a0(unit, rec, 1); then the operand through FUN_00646460. |
| FUN_006442a0(unit, rec, flag) | level = rec+0x28 (hard points); when `flag != 0` and rec+0x34 == -1 adds FUN_00644180(rec) (the item bonus); clamps to >= 0 and to FUN_00611830(0) (the cap). So flag 0 = base level (blvl), flag 1 = with items (lvl). |

Model: every Druid calc cell is a pure function of (own lvl, clvl through the stat getter stat 0xc, other skills' base or total level through FUN_006442a0, the row's params). Both passes' oracle hooks FUN_00625480 (stat getter), FUN_00643810 (record lookup) and FUN_006442a0 (level); everything else (expression machine, operand table, ln/dm/min, params) runs natively. That is game.exe code, not a re-derivation of d2d's formula.

Calc offsets of every Druid cell (`tools/emu/vsd_run.sh ptrs`, V run): no cell has offset 0; rows **224 Shape Shifting** and **238 Rabies** have no calc in any cell (all 0xffffffff), 239 Fire Claws only `calc1=0`; `perdelay` is empty on all 30 rows.

## 2. Own harness and runs (V run)

`tools/emu/vsd_calc.py` + `tools/emu/vsd_test.cpp` + `tools/emu/vsd_run.sh N SEED [MODE]` (built by hand against the scratch build, as the cookbook says). Differences from the authors' harness, on purpose:

1. The level hook honours the flag: a case gives every other skill a base level 0..45 and an item bonus 0..12, so `.blvl` and `.lvl` differ; the d2d side sets `CalcEnv::base_level` and `CalcEnv::level` separately. The first pass fed both the same number.
2. Every cell is passed to FUN_00646ca0 whatever its offset (the game's guard decides).
3. The stat getter logs every stat id the game asks for.
4. Own level grid: own lvl 0, 1, 2, 7, 8, 9, 16, 17, 22, 23, 28, 29, 40, 60 alone (clvl 1) and with all other Druid skills at base 10 + bonus 7 (clvl 75), then random cases (lvl 0..70, clvl 1..99, up to 10 other skills).

| Run | Cases | Result |
|---|---:|---|
| `vsd_run.sh 4000 101` | 4,840 | DIFF 0 |
| `vsd_run.sh 25000 202` | 25,840 | DIFF 0 |
| negative control `vsd_run.sh 4000 101 game-noflag` (game hook answers blvl with base + bonus, as the first pass did) | 4,840 | DIFF 78, all on rows 235 / 249 / 250 (e.g. Cyclone Armor L1: 46,592 instead of 31,744): the diff sees a blvl/lvl mix-up |
| authors' `skdruid_run.sh 3000 1` re-run unchanged | 5,400 | DIFF 0 (reproduced) |

Stats asked by the game over all cases: only stat 12 (clvl), by the 10 rows using `ulvl`. So the stat hook answering 0 for other stats hides nothing for these rows. Non-triviality: the game produced 17..99 distinct values per level-dependent cell (`vsd_check.py`), e.g. Raven calc2 = clvl - 2 + lvl (73 at clvl 75, lvl 0).

Values read off the game side (V run, seed 101 fixed cases; all equal d2d):

| Row | Cell | Game values |
|---|---|---|
| 221 Raven | petmax `min(lvl,par2)` | 1 at L1, 5 at L40 |
| 227 Spirit Wolf / 237 Fenris / 247 Grizzly | petmax | 5 / 3 / 1 at L40 (1 at L1) |
| 222, 226, 231, 236, 241, 246 | petmax | 1 |
| 235 Cyclone Armor | aurastat 1/2 | 10,240 at L1 alone; 31,744 at L1 with Twister/Tornado/Hurricane at **base** 10 (+7 items): the synergy uses blvl; 130,048 at L40 |
| 249 Armageddon / 250 Hurricane | `delay` 150, aurarange 8 / 9, auralen | 250 alone at every own level; 750 with Eruption / Cyclone Armor at base 10 (par7 50 per base point) |
| 223 Werewolf / 228 Werebear | auralen | 1000 alone; 10,000 with Shape Shifting at 17 (uses `.ln12`, the total level) |
| 233 Maul | aurastat1 (`damagepercent`) `lvl*par3` | 20 at L1, 800 at L40; aurastat2 (`stunlength`) 23 .. 95 |
| 242 Hunger | calc1 / calc2 / calc3 | -75 / 72 at L1, 192 at L40 |
| 248 Fury | calc1 / calc2 | 2 .. 5 / 100 at L1, 763 at L40 |
| 226 / 236 / 246 totems | passivecalc1/2 (poisonresist, maxpoisonresist) | 100 constant |

## 3. Claim by claim (first pass)

| # | Claim | Verdict |
|---|---|---|
| C1 | Every calc cell of the 30 rows equals d2d, 0 of 27,800, oracle = game.exe FUN_00646ca0 | **HOLDS WITH LIMITS.** The oracle does execute game.exe's calc machine (not arithmetic), and seed 1 reproduces. Limits the doc did not state: (a) its level hook ignored the flag, so blvl vs lvl was never tested; closed here (30,680 own cases, DIFF 0, negative control 78); (b) the clamp in FUN_006442a0 (negative, cap) is hooked out in both harnesses; (c) rows 224 and 238 have no calc, so "every calc cell equals" compares 0 with 0 there. |
| C2 | Summon caps Raven 1,2,3,5,5,5,5; Spirit Wolf the same; Fenris cap 3; Grizzly 1; plant / totem summons 1 | **VERIFIED** (petmax cells, V run). The spawn-path enforcement remains U, as the doc says. |
| C3 | "Firestorm area rows have no calc" | **WRONG.** Row 225 has `calc1=ln12` (3 at every level, par2 0) and `delay=15` (D, V run). |
| C4 | Armageddon / Hurricane "aura period 150 ticks, range 8 / 9, length 250 ticks at every level (levels do not change them)" | **WRONG in two parts.** 150 is the `delay` cell (+0x190), not the aura period: `perdelay` is empty (0) on both rows. Length is 250 only without synergy: `auralen = ln12 + skill('Eruption' / 'Cyclone Armor'.blvl)*50` (750 at base 10, V run; 1,250 at base 20 by the same formula, I, which is what the first pass's neighbour-at-20 cases must have printed). Range 8 / 9: VERIFIED. |
| C5 | Cyclone Armor absorb 10,240 at L1, 68,608 at L20 | **VERIFIED** for the skill alone (L1 run; L20 = 268*256 on the same formula, L40 run 130,048); the synergy from Twister / Tornado / Hurricane base levels (+7 % each) was not mentioned. |
| C6 | "Maul tohit 20 / 400" | **WRONG label**, right numbers: the cell is aurastat1 = `damagepercent` (lvl*20); tohit is not a Maul aura stat. |
| C7 | Feral Rage "life-steal pair" | **HOLDS WITH LIMITS**: aurastat2 is `lifedrainmindam` (4*lvl); aurastat1 is `velocitypercent` (dm34), aurastat3 equals aurastat2 by calc. |
| C8 | Hunger calc1 -75, 72 .. 176 on calc2 / calc3; Fury 2 .. 5 hits, 100 .. 423 | **VERIFIED** (176 and 423 are the L20 values; 192 / 763 at L40). |
| C9 | Oak Sage / Heart of Wolverine / Spirit of Barbs "aura values 100 and level-scaled passive" | **WRONG description**: the 100s are the totems' passivecalc1/2 (poison resist and max poison resist, constant), the level-scaled numbers are calc1/calc2 (life and level of the summon). The summoned auras live on separate rows (Oak Sage Aura etc.), not run. |
| C10 | "Poison Creeper is Vines / Plague Poppy in 1.14d" | **WRONG mapping (I, from the in-game names)**: row 222 Plague Poppy is Poison Creeper, 231 Cycle of Life is Carrion Vine, 241 Vines is Solar Creeper. |
| C11 | Deviations: none in the calc aspect; effects not covered (D1..D3 U) | **VERIFIED** for the calc aspect; D1..D3 stay U (not run by either pass). |
| C12 | B1 "the bot has no Druid combat code" | **WRONG**: internal/combat/druid.go exists (see B1, B2 below). |
| C13 | "Nothing here is V read" | Correct as a self-description. |

What the first pass got right: the harness design (game.exe's calc machine as oracle, d2d's real `eval_calc`, explicit non-zero check), the counts, the petmax numbers and the honest "effects not run". What it got wrong: several value descriptions in the summary (C3, C4, C6, C9, C10) and the bot finding (C12), and it did not state the blvl limit of its hook.

## 4. Deviations, bot findings, PR

- Deviations d2d vs game.exe: none in the calc aspect (V run, 30,680 own cases). No PR candidate: nothing to fix.
- **B1 (V read of our code, numbers V run):** internal/combat/druid.go:105-110 hard-codes the four caps 5 / 3 / 1 / 5 and the skill names in Go. The caps equal game.exe's petmax at high level, and :132 `min(s.Level, max)` matches `min(lvl, par)` of rows 221 / 227 / 237. Which level the spawn path feeds petmax (base or with items) is U. The comment at :98-99 can now say the per-level count is V run at the calc level. Skill names and caps in a function body are a build-flexibility leftover (AGENTS.md "Builds are data"); not changed here.
- **B2 (D + I):** druid.go:107 and :133-134 plan up to three spirits (Oak Sage 1, Heart of Wolverine 2, Spirit of Barbs 3, "the n-th spirit skill is wanted while fewer than n spirits stand"). In `Skills.txt` all three have `pettype=totem` and `petmax=1` (D, V run petmax 1), so the game keeps one spirit per pettype (I: the replace-on-recast rule is in the spawn path, U for game.exe). The plan would recast spirits that replace each other; whether `capSeen` (:139) stops that loop is not checked.

## 5. Ledger rows

`docs/coverage/ledger.tsv` row `skills.dru 221-250 TESTED-0-DIFF 27800` (rule-coverage shows 30 / 30 run 0 diff, 100 %). **Earned for the calc aspect, with a note:** the note already says "calc cells only; effects not compared", but rows 224 Shape Shifting and 238 Rabies have no calc cell at all, so they were not compared on anything. Proposed change (for the ledger owner, not applied): split the row into `221-223,225-237,239-250 TESTED-0-DIFF 58,480` (27,800 + this verifier's 30,680, note "calc cells; blvl/lvl split by verifyskdruid") and `224,238 UNTESTED` (note "no calc cell; form and proc effects not run").

## 6. Drill recipe (not run; lab binary only, docs/d2d-fork.md, docs/forkcheats.md)

1. Spawn-path cap: Druid with Raven, Spirit Wolf, Fenris, Grizzly at hard level 3 and +2 from items; cast each 6 times, count pets: whether the cap follows base 3 or total 5 settles the U of B1.
2. Totems: cast Oak Sage, then Heart of Wolverine: count totems (expect 1, the first replaced) for B2.
3. Armageddon / Hurricane with Eruption / Cyclone Armor hard points 10: state duration 750 frames.

## 7. Glossary

No missing term found.

## Shared-file lines for the merger

See openspec/changes/verifyskdruid/merge-lines.txt.
