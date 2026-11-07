# Gate 2 g2g: combined five-seed verdict (waves 1 and 2)

> status: current, 2026-11-07. Task TASK-71. Docs only. Wave 1 (seeds 11, 13, 17, tag g2g): `docs/soaks/gate2-multiseed-20261107-0150-g2g.md` (merged, not rewritten). Wave 2 (seeds 19, 21, tag g2g2): `docs/soaks/gate2-multiseed-20261107-0953-g2g2.md`. Diagnosis of wave 1: `docs/research/g2g-diagnosis.md`.

Tags: V = counted from the committed logs (`docs/probes/soak-logs/*g2g*`), the per-seed records, the full bot logs on the worker host (read-only) or a `d2dsoak compare` run; S = secondary; U = unverified.

## 1. Setup (V)
- Same build for all five seeds: bot 2a6cf2b, binary `d2d.lab3-labverbs-f75816c6-linuxbuild`, ladder off, 8 wall h, fault at 60 wall min. Seeds 11 and 13 are tuning seeds, 17, 19 and 21 are holdout seeds (16 to 23).
- Judge: `d2dsoak` rebuilt from origin/main abea7be1 (`go build ./cmd/d2dsoak` on the worker host under nice 19), which carries the belt rule of #300 (`BeltWindow.Funded()`: a window counts only when pack plus stash gold at its FIRST minute covers a potion). The old binary inside the run directory was not used for the wave 2 compare; the head-side judge (`gate-judge.py`) is the ops copy. Baselines: `tools/d2dsoak/baselines-early.json` at the same commit.
- Restart-containing NOKILL and BELT windows are kept literal (user decision 2026-11-06/07). Andariel interim floor stays clvl 17 (the hold is on). Gate 3 not started.
- The collect step ships the judge that was copied into each run directory at start, so the wave 2 extras logs still lack the merc `revived`/`seen` and early-death lines (the ops change c027d65 is newer than the run start). The merc and early-death numbers below therefore come from the full bot logs on the worker host (read-only; the same script reproduces the wave 1 numbers of the diagnosis exactly).

## 2. Pass rule (contract)
`docs/contracts/soak-gates.md`, "Gate 2 and Gate 3 on d2d": the run is judged by the Gate 1 criteria, the m24 limits and `INVARIANTS: PASS`; the invariants set "is VIOLATED when any seed is, PASS only when every seed is PASS" (same file, invariants requirement). A violated invariant is not a pass even if the m24 limits hold. There is no k-of-n or holdout exception. XP/h is advisory (`docs/decisions.md`, 2026-11-03: at least 5 seeds, 2 of them holdout). So Gate 2 passes only if all five seeds pass every criterion.

## 3. Five-seed table (full runs; V)
| item | 11 (tune) | 13 (tune) | 17 (holdout) | 19 (holdout) | 21 (holdout) |
|---|---|---|---|---|---|
| verdict (all criteria, belt rule) | PASS | FAIL (NOKILL) | FAIL (NOKILL) | PASS | PASS |
| virtual min / final clvl | 1963 / 16 | 2414 / 17 | 1699 / 16 | 2237 / 16 | 2986 / 17 |
| XP / kills | 249706 / 12522 | 289572 / 15123 | 243718 / 10696 | 260421 / 14142 | 305340 / 20001 |
| deaths / writes after start | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| longest NOKILL (limit 10) | 6 | 11 | 14 | 9 | 7 |
| longest funded BELT, new rule (limit 5) | 3 | 1 | 3 | 2 | 4 |
| fault recovery (path A, limit 180 s) | 3.4 s | 2.2 s | 2.2 s | 2.2 s | 2.4 s |
| Andariel interim rule (hold on) | PASS | PASS | PASS | PASS | PASS |
| merc: died / revived / alive share (time-weighted) | 17 / 16 / 11.5% | 17 / 16 / 4.2% | 16 / 15 / 16.8% | 17 / 16 / 14.4% | 15 / 15 / 30.2% |
| merc alive at the end | no | no | no | no | yes |
| fresh games (d2dsoak) | 41 | 88 | 25 | 60 | 29 |
| `mouthRef` summed on minute lines | 422 | 468 | 206 | 652 | 205 |
| XP/h full run (advisory, baseline 9172) | 7627 | 7193 | 8600 | 6982 | 6134 |
| XP/h over the first 1579 min (same length for all) | 8533 | 8451 | 8742 | 8568 | 8670 |
| dominant `fsWhy` (minute lines) | task_step 19, np_clearing 15, recovery 6 | np_clearing 50, task_step 20, recovery 11 | np_clearing 11, task_step 8, recovery 4 | recovery 23, task_step 23, np_clearing 21 | task_step 9, np_clearing 4 |

Pass count: 3 of 5 (11, 19, 21). Holdout: 2 of 3 (17 fails). Mean XP/h of the five full runs 7307 (-20.3% vs 9172, tolerance 1834); XP/h falls with run length (clvl 16 to 17 slows), so the first-1579-min column is the like-for-like one: mean 8593 (-6.3%). The `tp` field reads 0 on all minute lines in this build (fix #286 is newer); the town-portal criterion is judged from log text (390 trips on seed 19).

### d2dsoak over all five seeds (cut to the shortest, 1699 virtual min = 28.3 h each, 141.5 virtual h; V)
`xp_per_hour` 8302 (sd 182, -9.5%, tolerance 1834, PASS band), `kills_per_hour` 378 (PASS), `deaths_per_hour` 0 (PASS), `deaths_per_1000_kills` 0 vs baseline 1.02, tolerance 1.0: FAIL. `INVARIANTS: VIOLATED(seed 13: NOKILL,seed 17: NOKILL)`, `OVERALL: FAIL (exit 3)`. The only failing fidelity row is zero deaths being flagged against a 1.02 baseline (U: the metric looks two-sided at this tolerance; a gate-tool question for the maintainers, not a bot defect; XP/h stays advisory). Group holdout (17, 19, 21): 84.9 virtual h, 8369 XP/h, VIOLATED(NOKILL). COVERAGE: act 1, clvl 1 to 16, class and difficulty unknown, 141.5 virtual h.

## 4. Verdict
**Gate 2 is NOT PASSED on bot 2a6cf2b.** Five seeds were run (the real-claim size), 3 pass and 2 fail one criterion each:
- Seed 13 (tuning): NOKILL 11 min (limit 10) at minute 1190, Cave Level 1; cause F1 in `g2g-diagnosis.md`: equip failure leaves the game, then a nav mouth-gate refusal walks the failsafe ladder to a game restart (3 fresh games inside the window).
- Seed 17 (holdout): NOKILL 14 min at minute 1036, Black Marsh; cause F2: no town portal scroll, nav pocket, walk to town fails; only game restarts free it. Its BELT failure of wave 1 is gone under the new rule (section 5).
- Seeds 11, 19, 21 pass every criterion; wave 2 shows no new failure type. NOKILL on 19 (9) and 21 (7) is close to the limit on 19.
Both failing windows contain game restarts and are kept literal. Causes are from the wave 1 diagnosis (V there); whether other seeds would hit the same two mechanisms is U.

## 5. Addendum: wave 1 re-judged with the rebuilt d2dsoak (belt rule), rows that change (V)
Compared with the merged wave 1 record (cut to 1699 min, `d2dsoak compare -gate v1 -hours 28.32`), same logs, rebuilt binary:
| row | merged wave 1 record | re-judge |
|---|---|---|
| seed 17 funded BELT window | 8 min (pack+stash 16..31, counted as funded) | 3 min; the 8 min window is now poverty (first minute 16 < potion price) |
| seed 17 d2dsoak verdict | VIOLATED(BELT,NOKILL) | VIOLATED(NOKILL) |
| seed 17 per-seed criterion "Belt" | FAIL | PASS (full-run single judge: funded BELT 3) |
| group holdout seed 17 | VIOLATED(BELT,NOKILL) | VIOLATED(NOKILL) |
| overall invariants | VIOLATED(seed 13: NOKILL, seed 17: BELT, seed 17: NOKILL) | VIOLATED(seed 13: NOKILL, seed 17: NOKILL) |
Unchanged: seed 11 (funded 3, poverty 8 and 6), seed 13 (1), every NOKILL value (6, 11, 14), XP/h 8339, deaths, fault recovery. The review table in the wave 1 record lists funded BELT 0 for seeds 11 and 13; the d2dsoak block of the same record, and this re-judge, give 3 and 1 (the review table counted differently; U which). Seed 21 full-run funded BELT is 4 (limit 5, the cut run showed 1).

## 6. Mercenary (V, full bot logs)
Revives per wall hour are still capped at 2 on the wall clock (M1 of the diagnosis): `revive skipped (revive cap)` 52, 52, 58, 52, 18 lines on seeds 11, 13, 17, 19, 21, `(gold)` 1, 0, 3, 0, 2. Gold paid for revives: 3030, 3030, 4500, 3870, 10950. Alive share 4 to 17% on four seeds, 30% on seed 21 (the only seed whose merc was alive at the end; it also had the fewest cap skips). Seed 21's much higher revive spend (10950 for 15 revives) shows a different read quote (U why). Merc hired on every seed.

## 7. Early-death rules (V, full logs)
They fired on every seed, with 0 deaths in 5 runs: `guard: potion burst` 8 / 5 / 4 / 5 / 7, `stale pack noted` 37 / 24 / 31 / 25 / 36, `field gate: stale pack` 14 / 6 / 9 / 14 / 19 (seeds 11, 13, 17, 19, 21). Whether they prevented deaths is U (no control run). The extras logs of both waves do not carry these lines (judge copy older than c027d65).

## 8. Paired comparison with g2f (seeds 11, 13, 17; g2f = bot 65a7013, g2g = 2a6cf2b; V)
| seed | NOKILL f / g | funded BELT f / g | deaths f / g | XP/h first 1579 min f / g | fresh games f / g | mouthRef f / g |
|---|---|---|---|---|---|---|
| 11 | 8 / 6 | 4 / 3 | 0 / 0 | 9267 / 8533 | 34 / 41 | 430 / 422 |
| 13 | 9 / 11 | 2 / 1 | 1 / 0 | 8646 / 8451 | 68 / 88 | 536 / 468 |
| 17 | 6 / 14 | 3 / 3 | 0 / 0 | 9050 / 8742 | 20 / 25 | 159 / 206 |
g2f passed all three seeds; g2g loses NOKILL on 13 and 17, and XP/h is lower on all three (-8%, -2%, -3%). Deaths and belt are equal or better. One run per seed per build: the NOKILL difference is the two restart-heavy windows above; whether it is a regression or seed variance is U (the diagnosis points at known mechanisms, not at a g2g-specific change).

## 9. What the next wave must prove (fixes not in bot 2a6cf2b)
- #296 merc clock/quote: revives follow the virtual clock (more than 2 per wall hour, fewer `revive cap` skips), gold-farm need uses the read quote, alive share clearly above 4 to 30%, less time in the gold-farm zone list; no BELT regression from the larger revive spend.
- #298 mouth gate: `mouthRef` sum and the repeated "no safe click" holds fall well below 160 to 650 per seed; no NOKILL window caused by the failsafe ladder (the seed 13 pattern); fresh games per seed down from 25 to 88.
- #299 scroll-first: no `town trip abandoned [no town portal scroll]` and no nav-pocket NOKILL window (the seed 17 pattern); `tp` counts trips (also needs #286).
- Every seed: NOKILL at most 10 with the literal restart rule, funded BELT at most 5, 0 deaths, 0 writes; early-death lines still fire; five seeds again with at least two holdout seeds.

## 10. Not done
No run was started; Gate 3 not started. Not verified: whether the two NOKILL seeds recur on the same build (needs a rerun), the cause of the seed 21 revive price, the gate-tool treatment of zero deaths.
