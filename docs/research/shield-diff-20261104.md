# Shield / off-weapon damage differential, D1 and D2 (job shielddiff)

> status: current (2026-11-04, Sonnet, plan-account runner). Offline: no game, no memory writes, nothing pushed. Follows docs/research/skshield-trace.md (D1 off-weapon enhanced damage, D2 broken item props). Tags: **V run** = game.exe's own code under the emulator on the worker host, compared with d2d's real build; **S** = existing trace; **I** = inferred; **U** = unverified. d2d = the scratch build of upstream used by SKMELEE2 (make_fighter + player_blow, `tools/emu/sk_melee2_test.cpp`, built by hand with `tools/emu/shield_diff_build.sh`).

## Result

| claim | tag | counts |
|---|---|---|
| D1: game.exe adds the off-weapon ED (player stats 17 / 18) to the damage percentage; d2d ignores it | **V run** (formula FUN_0057b420 on game.exe vs d2d's real fighter), the input "a worn shield / armor reaches 17 / 18" is S (skshield-trace Q2/Q4, 382 scenarios, 0 differ from the model) | 20,000 O cases per seed, seeds 7 and 1 (5 off values x 4,000); "weapon" cases with off 15 / 40 / 100 / 300: nearly all differ in whole points, with off 0: 0 differ |
| D1 proposed rule (phys_pct += off-weapon ED) closes it | V run for the formula, **I** for the code change (rule emulated through d2d's input, see below; no fix compiled) | weapon cases, all four off values, both seeds: **0** differ in whole points after |
| D2: broken worn item's props are off in game.exe | V run game side (skshield_run.py: 382 scenarios, 0 differ from model; every "broken" row has an empty player list for the shield) | d2d side V read only (fight.cpp:505-511 has no broken test): **no differential run**, no 20,000-case count |

## D1 counts (seeds 7 / 1; "weapon" = armed cases; mean = game minus d2d, points)

Before (d2d unchanged), cases that differ in whole points, per cell (compared cases in brackets):

| off ED | seed 7 | seed 1 |
|---|---|---|
| 0 | 0 of 3198 | 0 of 3235 |
| 15 | 3207 of 3245 (mean +9.3) | 3208 of 3256 (+8.5) |
| 40 | 3265 of 3292 (+22.1) | 3187 of 3214 (+26.8) |
| 100 | 3192 of 3207 (+64.6) | 3194 of 3215 (+63.0) |
| 300 | 3254 of 3254 (+184.7) | 3215 of 3217 (+188.0) |

After (the off-weapon ED added to d2d's percentage): weapon cells off 0 / 15 / 40 / 100 / 300: **0 differ in whole points in all ten cells** (seed 7: 3198, 3245, 3292, 3207, 3254 compared; seed 1: 3235, 3256, 3214, 3215, 3217); the leftover is the known +0.50 point 256ths rule (SKMELEE2 D1, unrelated). Whole-set totals: seed 7 15,537 differ before, 2,508 after; seed 1 15,477 before, 2,530 after. The remainder after is the **barehanded cases** (about 500 of 590 per cell differ with off 0 too: SKMELEE D4, a separate deviation, not changed by this rule).

First differing case (the driver prints the first compared case that differs in whole points; the case may be barehanded, so it is a D4 example, not specific to D1): seed 7 id 2 (game 409, d2d 3), seed 1 id 4 (game 5563, d2d 8) in 256ths-shifted units of the out file's columns (I: column meaning not re-checked).

How the "after" was made (I): d2d's make_fighter puts stat 25 into the one percentage `phys_pct` (combat.hpp:117), the same place game.exe adds 17 / 18. The driver builds a second case file with the off value added to the stat-25 column and off 0, and runs the unchanged d2d binary on it. That emulates `phys_pct += other worn ED` for items without their own flat 21 / 22 (skshield D3). A real code change in fight.cpp (a separate off-weapon ED sum per item) is not compiled here, so **"after 0" is a formula result, not a fix result**.

Caveat (honest limit): the game side hooks the player's stat reads (sk_melee2.py `game`), so the 17 / 18 values are fed in; that a worn shield's ED does reach the player's 17 / 18 is skshield-trace (S, run on the real equip code).

## Harness commands

```sh
tools/emu/run-on-worker.sh --ttl 1500 shield_diff_build.sh > build.txt 2>&1        # d2d test binary
tools/emu/run-on-worker.sh --ttl 2400 shield_diff_run.py 4000 7 1 > out.txt 2>&1   # D1: 2 seeds x 20,000 cases, before and after
tools/emu/run-on-worker.sh skshield_run.py > run.txt 2>&1                          # D2 game side: "scenarios 382, differ from the model 0"
```
Files: `tools/emu/shield_diff_{build.sh,gen.py,run.py}`; it reuses `sk_melee2.py` and `sk_melee2_test.cpp`.

## Verdict per deviation (for SKSHIELDFIX)

- **D1: PR candidate (proven with a differential on the existing harness): before > 0 (about 3,200 of 3,250 whole-point mismatches per armed cell with off ED), after 0 (rule emulated).** It still needs the real change in fight.cpp, the harness rerun on that build, and the independent verifier; the Fighter needs the off-weapon ED sum without items that hold their own flat 21 / 22 (D3). Not a branch from here (no fork checkout on this runner).
- **D2: not yet a PR candidate**: no d2d-side differential exists. Build: a test that makes a character with a worn item carrying stats 17 / 25 and flat 21 / 22, flag 0x100 set, runs the fight.cpp:500 builder, prints `sum`; oracle = skshield_run broken rows (all zero). U for the d2d count.

## Bot model

No off-weapon enhanced damage in the bot's gear or combat model: `grep -rn -i "damage"` over `internal/gear` (non-test) only hits a comment on bow ranking by cost (`internal/gear/ammo.go:141`); `internal/combat` has only test fields `meleeDmg` (e.g. `internal/combat/sim_test.go:52`). So the bot neither follows game.exe nor d2d here: it models no shield or armor damage stats at all (consistent with skshield B1/B2). Nothing to correct until a damage estimate exists (I).

## Matrix cell

sorc-skill-matrix.md 3.12 (b): append "D1 off-weapon ED: V run, 20,000 cases x 2 seeds, armed cells with off 15..300 differ in whole points (about 98 percent), after adding it to phys_pct 0 (formula); D2 game side V run, d2d side V read".

## Drill recipe (not run)

U to confirm live: with the lab binary wear a shield with +x% enhanced damage (forkcheats item verbs), read player stats 17 / 18 (forkdbg stat read) and a melee hit's damage with and without the shield (expect the percentage in the hit); then set item flag 0x100 and re-read the stats (expect 17 / 18 and flats gone).

## Shared-file lines for the merger

- BACKLOG: SKSHIELDFIX: D1 proven by the harness (shield-diff-20261104.md); D2 needs a d2d-side test
- INDEX: docs/research/shield-diff-20261104.md - D1/D2 shield differential, 20,000 cases x 2 seeds
- CHANGELOG: Added docs/research/shield-diff-20261104.md (D1 off-weapon ED differential run on the emulator, tools/emu/shield_diff_*)
