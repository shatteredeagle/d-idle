# fid-skill-damage-pr (TASK-89, matrix row SK02): player skill damage, d2d against game.exe

> status: finished 2026-11-07 by agent fidskilldmg (Sonnet). Result: **no PR candidate, because every proven skill-damage deviation is already merged upstream** (PR #146, merge commit 5475650e, 2026-10-31). The 116,000-case differential, re-run on current upstream main, is 0. Nothing was pushed, posted or filed. Earlier material: docs/research/skill-damage-trace.md, pr-skill-damage-proof.md (with the independent verifier's report), pr-skill-damage-draft.md, pr-skill-damage-evidence-pack.md.

Tags: **V run** game.exe's own code run under the emulator and compared with d2d byte for byte; **V read** read in the disassembly; **D** from the 1.14d tables; **S** shown by an earlier proof; **I** inferred; **U** unverified.

## 1. Why the card is a negative result

TASK-89 was parked with the premise "upstream skill damage min-synergy gate and MulDiv are d2d deviations, PR draft ready". Checking the newest upstream main first (fetch of the public repository, main = 9c78f391, 2026-11-07) shows the premise is out of date:

| Fact | Evidence | Tag |
|---|---|---|
| The three PR commits (20441196 tooling, d72ee7f3 gate, c6be9247 MulDiv) are ancestors of upstream main | `git merge-base --is-ancestor c6be9247 upstream/main` true; merge commit 5475650e "Merge pull request #146 from shatteredeagle/fix/skill-damage", 2026-10-31 | V run (git) |
| Upstream `rules::elem_damage` already holds both rules | components/rules/skills.hpp `elem_damage`: `(max \|\| damage > 0x100 \|\| skill.emin_lev[0] != 0)` gate and `mul_div` for the synergy and mastery shares; the comment cites FUN_00644d50 at 0x644db3 | V read |
| The oracle tool and the test mode are already in the upstream tree | tools/emu/skill_damage.py, `test_game --skill-damage` (tests/game/test_game.cpp) | V read |

So a replay of the old branch as `local/fid-skill-damage-r1` would be an empty diff (the cherry-pick of 20441196 conflicts only because the test file moved to tests/game/ and the hunk is already there). No branch was created on the private remote and no push was made (nothing to push).

## 2. Questions answered

**Q1. Does d2d's `elem_damage` / `elem_length` still equal game.exe's FUN_00644d50 / FUN_00644e40 / FUN_00644f20 (flag 1, with the mastery) on current main?** Yes (V run). Build: upstream main 9c78f391 plus nothing (scratch clone on the worker host, Release, `-DD2D_WERROR=ON`, `test_game` built). Oracle: `tools/emu/skill_damage.py` (hooks only the stat getter FUN_00625480, the unit's skill record FUN_00643810 and its level FUN_006442a0; the level brackets, the calc machine, `skill()`, MulDiv and the mastery run natively).

| Run | Cases | Lines differing | md5 of game side = d2d side |
|---|---|---|---|
| seed 1 (16,000 grid + 100,000 random) | 116,000 | **0** | 4440bed408f78c55d8be3991140df9e1 (the same value the PR's AFTER run had, so the result is unchanged since the merge) |
| seed 7 | 116,000 | **0** | 990c6e6899a8e5d4806209f025e2f10f |

The grid covers every skill that has an element (EType), levels 1..40, alone, with its class neighbours at 20, with masteries 100 at clvl 50, and masteries 37 at clvl 99; the random part has levels with the bracket edges, clvl 1..99, masteries 0..600 and 0..6 other skills. The BEFORE counts of the merged PR (4,549 lines differing before, 344 after claim 1, 0 after claim 2) are the PR's own and are not re-measured here: re-measuring needs a build of the pre-PR parent (f8aab0e8); the rerun here proves "now 0", which is what AC 1 asks for. Charged Bolt, Fire Bolt, Ice Bolt, Frost Nova (Nova), Static Field's calc donors and the 13 formerly deviating lightning skills are all inside the 116,000 cases and are in the 0 (AC 2).

**Q2. Is the split between a skill's base level (`blvl`, FUN_006442a0 flag 0) and its level with item bonuses (flag 1) the same in d2d?** Yes for the element damage (V run, new in this card; this was the verifier's open limit "no case distinguishes blvl from lvl"). The oracle hook was extended (scratch patch `tools/emu/sdmg_items_patch.py`: the hook answers the base level for flag 0 and `base + 1 + id % 3` for flag 1 on any skill the unit has points in; the damage call itself is made at the with-items level) and `test_game --skill-damage` gets the same rule through `env.base_level` / `env.level` (environment switch, scratch only, not a candidate). 116,000 cases (seed 1): **0 lines differ**, md5 672dbe6e2d0b9a1fbf9a1ce4929c4ba5 on both sides, and it differs from the no-items md5 (4440bed4...), so the bonus is really exercised. Limits: the item-bonus function FUN_00644180 and the level cap FUN_00611830 are bypassed by the hook (synthetic bonus), so this proves the calc machine's operand split (operand 41 `blvl` flag 0, every other operand flag 1), not the item bonus sum itself.

**Q3. Which other skill-damage pieces are proven or unproven?** See section 3.

## 3. Deviation list (d2d against game.exe)

| ID | Rule | game.exe | d2d | Tag | State | Differential |
|---|---|---|---|---|---|---|
| D1 | the min takes its synergy only above one point or with EMinLev1 | FUN_00644d50 at 0x644db3 / 0x644dbb; the max FUN_00644e40 has no gate | was unconditional | V run (116,000 cases: 13 lightning skills) | **MERGED upstream (#146)**; now 0 | yes, `skill_damage.py` (in the upstream tree) |
| D2 | the synergy and mastery shares are MulDiv (divide first above 0x100000) | FUN_00483360 | was `a * b / 100` | V run (344 lines before claim 2) | **MERGED upstream (#146)**; now 0 | yes, same |
| D3 | Static Field (difficulty floor, at most life - 1, negative resistance, 256ths through the resistance) | FUN_005c96a0 | was a plain share of life | V run in its own PR | **MERGED upstream (#155, 2026-11-01)**; not re-run in this card (the re-run script of the CodeRabbit round did not apply as-is: its case files are fed differently, 4 minutes spent, stopped, no number claimed) | yes, upstream `tools/emu/static_field.py` |
| D4 | Cold length / chill application | FUN_0057c6c0 chill record | `blow.chill_ticks` | V run in its own PR | **MERGED with #155**; not re-run here | yes, upstream `chill_length.py` |
| D5 | EType 12 (freeze) takes the COLD mastery (stat 0x14b) | byte table at 0x644d38 (`00 01 04 02 03 04 04 04 04 04 04 02`: index 12 -> 2 = cold) | `elem_damage` applies the mastery only for `etype` 0..3, no freeze bucket | V read (verifier, pr-skill-damage-proof.md); no stock row has EType 12 (rows use 1, 2, 3, 4, 5, 9), so a stock game cannot show it | **unproven, no stock effect, not a candidate** (only a mod with EType=frze and damage would differ) | yes if wanted: add a synthetic row with EType 12 to the oracle's table load (needs a modified Skills.txt in a scratch MPQ directory); not worth a PR |
| D6 | The element mastery counts once per hit | FUN_0064ac60 sets the missile stats 0x30..0x42 etc. but no 0x149..0x14c, so FUN_005a8910's roll-time mastery read on the missile is 0 | `missile_blow` rolls `range(elo, ehi)` without a second mastery | V read (verifier re-checked 0x64ac60 only) | holds for the creation path read; other missile creators / stat-list copies not traced | no (would need a whole-missile oracle: a missile created through FUN_0059f900 and rolled through FUN_005a8910) |
| D7 | `skill_phys` (a skill's own physical part, FUN_00647bc0 / 00647d00): `damage * calc / 100` then `<< HitShift` | `MulDiv`? not read for this function in this card | plain integers | Barbarian rows: V run 0 of 27,800 cases (skbarb-trace.md, weapon share off); other classes: **U** | unproven for Sorceress / Paladin / others; physical share is below 0x100000 for every stock row (I), so MulDiv cannot differ in stock data | yes: skbarb_calc.py takes a skill list; widening `BAR` to all ids and building `skbarb_test` against current main is the recipe (hand build, about half a day; not done, the Sorceress has no physical skill) |
| D8 | `elem_length`'s share is a plain `length * calc / 100` | FUN_00644f20 uses MulDiv | plain | V run: 0 of 116,000 (and the verifier's 104,000 adversarial cases) | **no deviation observed**; MulDiv would matter only above 0x100000 (a length that big does not occur) | covered |

Nothing in D1..D8 is both proven and unmerged, so there is nothing to put into a candidate.

## 4. Bot findings

- **B1. The bot does not compute element damage.** It reads the game's memory and plans casts per kill with knobs (`staticCasts` in `DefaultCombatTuning`, internal/combat/sim_test.go:520). A d2d fix to the lightning min or the MulDiv rounding therefore changes the bot's casts-per-kill only through the game it plays (about 2 percent more average damage for the lightning plan on d2d, skill-damage-trace.md "Effect on the bot", I), not through any bot code. No change proposed.
- **B2.** The static-field plan was written against game.exe's rule; with #155 merged the bot's numbers and d2d agree (S, pr-static-*.md). Nothing to change.

## 5. PR candidate

**None.** Reason: all proven deviations (D1 to D4) are merged upstream; the remaining items (D5 to D7) are either not reachable with stock data (D5, D7 for the Sorceress) or not proven by a differential (D6). The brief's rule for this case: "If a deviation cannot be proven, document the negative result and stop short of a candidate." So also: no `local/fid-skill-damage-r1` branch exists, nothing was pushed to the private remote, no PR draft was written, no verifier brief is needed, and no queue entry for verification is added. The old draft (pr-skill-damage-draft.md) has an empty `posted:`; the merge lines below record that it was posted and merged as #146.

Checks run on the scratch build of upstream main 9c78f391 (worker host, nice 19): `ninja test_game test_combat` rc 0 with `-DD2D_WERROR=ON` and `-Wshadow -Wunused -Wsign-compare`. test_game, pr_guard and presubmit were not run: there is no diff to check.

## 6. Matrix-cell update (proposed text for docs/research/fidelity-matrix.md, row SK02)

Replace the Notes cell with: "skill_damage 116,000 cases x 2 seeds on upstream main 9c78f391: 0 differ (md5 4440bed4... and 990c6e68...); the min-synergy gate and MulDiv were fixed by upstream PR #146 (merged 2026-10-31); blvl / lvl split also 0 of 116,000 with a synthetic item bonus; open and unproven: EType 12 freeze mastery (no stock row), skill_phys outside the Barbarian (V run 0 of 27,800 there). No PR pending (docs/probes/fid/fid-skill-damage-pr.md)". Status cell: DEV -> OK (no remaining d2d deviation for the element damage). SK09 (Fire Bolt) follows the same 0.

## 7. Drill recipe for the fork verbs (nothing was run)

Only one live-visible consequence is left to confirm (U): that the running d2d shows Lightning's minimum at 1 with synergies. In the lab binary (docs/d2d-fork.md, docs/forkcheats.md, verbs in docs/forkdbg.md) give a throwaway Sorceress Lightning at level 1 and Charged Bolt at 10, read the skill tooltip's damage line and expect the range to start at 1 (without the gate it would be 1.8 with this synergy). No drill is needed to close the card: the oracle covers the same function.

## 8. How to re-run (the commands, for the next agent)

```sh
# worker host; scratch clone of upstream main, test_game built (Release, -DD2D_WERROR=ON), mpq-cat from any build
tools/emu/sdmg_run.sh SEED 100000 /path/to/build/tests/test_game TAG    # prints counts, differing lines, md5 of both sides
SD_ITEMS=i tools/emu/sdmg_run.sh 1 100000 ...                           # after applying tools/emu/sdmg_items_patch.py to the scratch tree
```
`sdmg_run.sh` needs `/tmp/casttrace/game.exe` (the 1.14d executable), the MPQ directory and the patch installer as its comments say; the environment values in it are the worker host's and are for that machine only.

## 9. Dead ends

- Replaying the old branch: empty (already merged).
- Re-running the Static Field and chill-length oracles from the CodeRabbit round: the script I reused fed the d2d side a count where the oracle expects its own case file (218 lines against 24,312); stopped, no result claimed. Their proofs are in the upstream PR (#155) and the repo's pr-static-*.md.
- A whole-repo clang-tidy: not run (no diff).

## Shared-file lines for the merger

See `backlog/merge-lines/agent-fidskilldmg.txt`.
