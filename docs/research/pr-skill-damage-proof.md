# Proof: a skill's element damage with synergies and masteries as game.exe computes it (fix/skill-damage)

> status: current (agent DECOMPSKILL, 2026-10-31). Applies docs/process/upstream-pr-standard.md. Branch `fix/skill-damage` in `~/d2d-lab/pr/skilldmg` on the worker host, three commits on upstream/main f8aab0e8: `20441196` (tooling: `tools/emu/skill_damage.py`, `tests/test_game.cpp --skill-damage`), `d72ee7f3` (claim 1 + doc note in `docs/research/re/skills.md`, `tools/emu/README.md`), `c6be9247` (claim 2); author and committer the user, body `Agent: Claude Code`, no trailer. Nothing pushed or posted. Trace: docs/research/skill-damage-trace.md. Tags: V = run or read by this agent (command given), S = upstream doc, U = unverified.

## Claims (two, one per commit)

1. **The min's synergy gate.** FUN_00644d50 (a skill's min element damage in 256ths) adds the EDmgSymPerCalc share only when `(EMin + brackets) << HitShift` is over 0x100 or the row's EMinLev1 is not 0 (0x644db3: `cmp esi, 0x100; jg`; 0x644dbb: `cmp [edi+0x1e8], 0; je skip`); FUN_00644e40 (the max) has no gate (0x644e9e..0x644eac). d2d's `elem_damage` (components/rules/skills.hpp:544) added the synergy to the min unconditionally, so the "1-x" lightning skills (Lightning, Chain Lightning, Lightning Bolt, Charged Strike, Lightning Strike, Lightning Fury, Power Strike, Holy Shock, Shock Field, Charged Bolt Sentry, Claws of Thunder, Lightning Sentry, Death Sentry: EMin 1 << 8 or 2 << 7 = 0x100, EMinLev1 0, D) had a min above 1 with synergies. Fix: `(max || damage > 0x100 || skill.emin_lev[0] != 0)`.
2. **MulDiv shares.** The synergy share (0x644e0b: `MulDiv(esi, calc, 100)`) and the mastery share (FUN_00644c90: `MulDiv(dmg, stat, 100)`) are FUN_00483360, which above 0x100000 divides first (`a / c x b`). d2d used `damage * calc / 100` and `int64 * stat / 100`. Fix: the existing `mul_div` port (combat.hpp:297, PR #135).

## The chain, tagged (listings by `disasm.py --exe /tmp/casttrace/game.exe <addr>`, in /tmp/decompskill/*.txt on the worker host)

| Link | What it shows | Tag |
|---|---|---|
| FUN_0059f900 -> FUN_0064b860 | a missile's damage struct: 0x64ba30 FUN_00644d50, 0x64ba43 FUN_00644e40, 0x64ba56 FUN_00644f20 (flag 1 pushed: with the mastery) at the missile's level | V |
| FUN_0064ac60 | sets the missile's stats from the struct (FUN_00627260): 0x30..0x3b the elements, no 0x149..0x14c, so FUN_005a8910's mastery read on the missile at roll time is 0: the mastery counts once | V |
| FUN_00644d50 | base, gate, synergy by MulDiv through FUN_006c0bc0 (calc machine, callback FUN_00646be0 -> FUN_00646460), mastery FUN_00644c90 | V (listing + oracle) |
| FUN_00644e40 | the max: no gate | V |
| FUN_00644b70 | the level brackets (= d2d `level_bonus`) | V |
| FUN_00644c90 | byte table 0x644d38 by EType: 1 -> 0x149, 2 -> 0x14a, 4 and 12 -> 0x14b, 5 -> 0x14c, else none; MulDiv(dmg, stat, 100) | V (dumped with emu) |
| FUN_00646c00 skill() | the other skill's record FUN_00643810, level FUN_006442a0 flag 1, then FUN_00646460; operand 41 `blvl` = FUN_006442a0 flag 0 | V |
| FUN_00483360 | MulDiv's branches | V (PR #135) |
| skills.txt rows (EMin, EMinLev1, HitShift, EDmgSymPerCalc) | read from the loaded tables by the oracle (`rows`) and with mpq-cat | D |

No U link. Hooked in the oracle (not the claim; docstring): FUN_00625480 (stat getter: 0xc clvl, 0x149..0x14c masteries, else 0), FUN_00643810 (the unit's skill record: a fake record per id), FUN_006442a0 (its level: the case's, base = with items).

## The differential (V)

`tools/emu/skill_damage.py gen 100000 1` makes 116,000 cases: every skill with an EType at levels 1..40 x {alone; its neighbours (id +-15) at 20; the same with masteries 100 at clvl 50; masteries 37 at clvl 99} (16,000), plus 100,000 random (levels 1..60 with the bracket edges, clvl 1..99, masteries 0..600, 0..6 other skills at 0..40). `game` runs the real functions; `test_game --skill-damage` runs d2d's `elem_damage(min / max, mastery)` and `elem_length` with the same `CalcEnv`; joined by `diff`.

| Build | Lines differ (of 116,000) | min / max / len | Skills |
|---|---|---|---|
| upstream/main f8aab0e8 + the test mode (BEFORE) | 4,549 | 4,509 / 340 / 0 | 13 lightning skills (the gate); 20 skills over 0x100000 (MulDiv) |
| + claim 1 (d72ee7f3) | 344 | MulDiv only | Fire Claws, Meteor, Blizzard, Lightning Sentry, ... at levels 23+ with big shares |
| + claim 2 (c6be9247, AFTER) | 0 | md5 4440bed4... both sides | |

Outputs on the worker host: /tmp/decompskill/{cases,g,d-before,d-gate,d-final}.tsv.

## Their checks (V, the worker host, `~/d2d-lab/pr/skilldmg/build`, D2D_WERROR OFF)

`ninja test_combat test_game mpq-cat` ok; `test_combat` ok; `test_game` with D2_MPQ_DIR and D2_PATCH_INSTALLER rc 0, 27 OK; `pr_guard.py upstream/main` 0 FAIL 0 WARN; `presubmit.py --quick` 0 FAIL, 4 WARN: ruff absent; siblings ("skill, not melee, missile": the change is the skill's own calc, the missile and melee paths read it through `missile_damage` / `skill_element`: answered in the body); doc-claims ("the max always": read at 0x644e9e..0x644eac, no gate); build skipped by --quick (built by hand).

## Callers of `elem_damage` (what changes for them)

`missile_damage` (a missile's element), `skill_element` (a melee skill's element on the swing), the aura paths (fight.cpp:3238, 3597, 5010) and the tooltip / calc operands 17 / 18 / 38 / 39 / 49 / 50 / 52 / 53 (`calc_operand`). All take the gated min and the MulDiv shares; the tooltip now shows Lightning's "1-x" with synergies as the game does (FUN_004edea0 calls the same FUN_00644d50 through the operand table, V: FUN_00646460 cases 38 / 39 at 0x646808 / 0x646820). `elem_length` and `skill_phys` are unchanged (lengths matched in all cases; the physical part is outside the claims).

## Residual risks and limits (disclosed in the body)

- One level per skill in the cases (base = with items); game.exe's `blvl` / `lvl` split (FUN_006442a0 flags 0 / 1) matches d2d's `base_level` / `level` by reading, not by a case where they differ.
- The oracle's unit is a zeroed block with three hooks; a calc operand that reads other unit fields (none in the EDmgSymPerCalc of the elemental skills, D) would see 0 on both sides.
- Synergy skills in the cases are within id +-15 of the skill (a class's own skills); cross-class ids are not generated (no calc references one, D).
- `elem_length`'s and `skill_phys`'s `* x / 100` are left as they are (0 mismatches; outside the claims).
- Static Field (FUN_005c96a0: difficulty floor, at most life - 1, negative resistance neutralised, 256ths through the resistance) differs in d2d by reading: a separate candidate, not in this PR.
- `tools/emu/skill_damage.py` and the test mode touch `tools/` and `tests/`: CODEOWNERS review by the maintainer; the two fix commits stand without them.

## Verifier (independent, second agent, Fable; report verbatim)

VERIFIER_REPORT_PLACEHOLDER
