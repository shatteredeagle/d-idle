# Re-verification: Energy Shield absorb order r2 (fork branch local/es-absorb-order-r2)

> status: verifier verdict, 2026-11-04 (job esverify2, docs and harness only; no game, no Go change, nothing pushed). CLAIM: head `229aeca7` (one commit on upstream main `561814df`) makes d2d's Energy Shield equal game.exe 1.14d's own `FUN_005ca6c0`, including the flat / mdr reduction on a field the shield leaves under it (reduction before the resistance, the negative kept and summed, the record total clamped once). Previous verdict: `verdict.md` (head `12b78b60`, caveat C1). Tags: **V run** (run here, command below), **V read** (disassembly or bytes read here), **S** (another doc), **U** (not checked).

## Verdict

**CONFIRMED WITH CAVEATS.** C1 is fixed: on every new comparable case the head equals game.exe's native `FUN_005ca6c0` on life and mana, and on all shield cases on mana. What is left is outside the shield's own arithmetic and is named in the PR body's limits (K1 to K3 below). Ready to open after the body edits in "Recommended actions"; no code change needed.

## Upstream drift (V run)

Upstream main is now `0d424ede7` (3 commits after `561814df`). Of the five files the commit touches only `components/game/ai.hpp` changed upstream (a new `UnitState::filed` and `file_first_think`, NPC code), in a different hunk from the commit's `Foe::take` change; `git merge-tree --write-tree 0d424ede7 229aeca7` is clean. Builds and runs below are on `229aeca7` and `561814df` as they are.

## Harness

The merged esverify harness (`verdict.md`, `tools/emu/esv_*`), unchanged in what it measures, plus (this job):

| Piece | What it adds |
|---|---|
| `esv_build.sh`, `esv_tidy.sh` | find the new worktrees `verifier2-head` / `verifier2-base`; clean Release builds of `229aeca7` and `561814df` on the worker host |
| `esv_probe.cpp`, `esv_game.py`, `esv_cmp.py` | an optional sixth number a blow, a poison roll (d2d: `monster_blow` element type 3; game: record +0x28 with a length at +0x2c, the rows 8 and 7 of the resistance table `0x732980`). Five-number lines read as before |
| `esv2_cases.py` | a new edge grid (13,176 lines with twins: all six damage types, resistance -100 / 0 / 50 / 75 / 100, rolls 1..632, (flat, mdr) in (0,0) (15,0) (0,10) (15,10) (3,6) (40,40), levels 1 / 40 / 99, mana 0 / 1/256 / 300 points, and two-field records where a small field goes under the reduction next to a big one) and `rand_case2`: flat and mdr each in half the cases up to 60, resistance -100 / 75 / 100 often, poison in 15 % of the blows |
| `esv2_hand.py`, `esv2_handgame.py`, `esv2_show.py` | hand cases: the no-shield order, negatives, poison, a field the shield leaves untouched; the test's own numbers natively |
| `esv2_resid.py` | per run: comparable cases (control twin exact), head and base differences on them, and why any head difference is left |
| `esv2_tables.py` | the two field tables read from the image |

Runs (V run, the worker host): `tools/emu/run-on-worker.sh --ttl 2900 esv2_run.sh GEN SEED 20000` for `1 21`, `1 22`, `1 23` (the first generator, new seeds; the author used 7 / 8 / 9 of it) and `2 0` (the new grid plus 20,000 `rand_case2` frames); `esv2_run.sh 3 0 0` (hand cases); `esv2_resid.py` on the run dirs. Self-checks on every run: event 11 fired with the raw fields in every call (`snapshot_not_raw 0`, 95,453 shield calls on the four big runs), no hook called with unexpected arguments, the calc tables of base and head equal.

## Counts (V run)

| Run | Shield cases | Comparable (control exact) | Base life differs (pt / 256ths) | Head life differs (pt / 256ths) | Mana differs, all shield cases: base / head |
|---|---|---|---|---|---|
| gen 1, seeds 21 + 22 + 23 | 50,977 | 23,931 | 4,655 / 6,768 | **0 / 0** | 4,483 / **0** |
| gen 2, new grid + 20,000 | 24,655 (4,418 with poison: mana only) | 11,006 | 1,457 / 2,314 | 7 / 9 (K2) | 3,136 / **0** |
| hand cases | 17 + twins | | | see K1 | 0 on head |

No-shield frames, base `561814df` output == head output: 60,000 + 26,588 + all hand twins, every one (the change is off the no-shield path). Head wrong where the base was right: 74 (gen 1) and 45 (gen 2), every one in a frame whose control already differs (coincidental agreement before); head right where the base was wrong: 11,023 and 3,323.

The author's figure (comparable 24,174, head 0) is reproduced on new seeds: 23,931 comparable, head 0 in whole points and in 256ths, mana 0.

## Break attempts (V run)

| Attempt | Result |
|---|---|
| flat AND mdr both non-zero, up to 40 / 40 and 60 / 60, small fields driven under 0 | comparable cases: head equal except K2 |
| two fields, one under the reduction (fire 5 + cold 100 mdr 6; physical 10 flat 15 + fire 50; magic 5 + lightning 100 mdr 10) | game.exe sums the negative (93, 45, 85 points; V run). Under a shield the head does the same; without one it does not (K1) |
| resistance -100, 75, 100 (immune) on every element | equal on comparable cases; game.exe keeps the resistance off a negative field (fire 5 res -100 mdr 10 next to cold 100: 175, not 170) and the head does too |
| poison (all damage types) | `0x6e30e0` has no +0x28 row (V read, `esv2_tables.py`); game.exe takes no mana for a poison roll: mana equal on base, head and game.exe in all 4,418 poison shield cases and in the hand cases (poison alone, mana kept; poison + fire 100, mana the same as fire alone) |
| shield level 0 / calc1 0 (the twins), mana 0, mana 1/256 | equal; at mana 0 the field goes through the no-shield path on both sides and differs only by K1 |
| the PR test's own numbers natively (`esv2_handgame.py`) | 100 fire, 20 % shield, mdr 10, res 50: game.exe 35 points; 10 fire + 100 cold, 50 % shield, mdr 6: 43; same with mdr 0: 55. All three equal the test |

## The no-shield finding (brief item: confirm and judge the test)

Confirmed natively (V run, `esv2_handgame.py`, game.exe's own `FUN_0057c6c0` / `FUN_0057c1e0` / `FUN_0057bf80`, no shield): 100 fire, 50 % resistance, mdr 10 deals **45** (d2d base and head 40); res 75: 22.5 (d2d 15); res -100: 180 (d2d 190); a field under the reduction is kept negative and summed (fire 5 + cold 100 with mdr 6: 93, d2d 94; physical 10 with flat 15 + fire 50: 45, d2d 50). V read, `FUN_0057bf80` (0x57bf80..0x57c04d): field at or under 0 stored as 0; else the reduction (stat 34 physical, 35 the elements) comes off, then, only if the rest is above 0 and the resistance is not 0, `MulDiv(rest, 100 - min(res, 100), 100)`; the result is stored even under 0. On the random no-shield frames with flat or mdr, d2d differs from game.exe in whole points on 593 of 1,449 (gen 2) and 414 of 1,097 (gen 1, seed 21).

The test: the file `tests/game/test_energy_shield.cpp` is new in this commit, so no upstream test changes; the "30 to 35" is a change against the first draft only. 35 is game.exe's answer (V run above), so the expectation is right and the body's sentence about it is accurate.

Split: the no-shield order belongs in a SEPARATE PR. It changes `rules::monster_blow` for every monster hit on every unit (41 % of the no-shield frames with flat or mdr move), touches other tests and the parity lane, and is a different claim; this PR is the shield's. The body already names it as a limit; it should say plainly that it is a known, measured d2d deviation with the native numbers and that a follow-up will fix it (K1).

## Caveats

- **K1 (medium, V run). Fields the shield does not take from stay on d2d's no-shield path** (resistance then mdr, clamped per part), also inside a shield frame: once the mana runs out, or at mana 0, the later fields of the same record and frame differ from game.exe exactly as without a shield (hand case: physical 200 + fire 100, mdr 10, res 50, mana for part of the physical: game.exe 244.414, head 239.414). One record can now mix the two orders. The body's limit says this ("Only the fields the shield took from follow game.exe's order"); it should add "including the fields after the mana runs out".
- **K2 (low, V run).** The 9 comparable gen-2 cases the head still gets wrong (7 in whole points) are all frames where the mana ran out (game.exe leaves under one point): the untouched fields carry K1 and the twin's K1 errors cancelled (case 11,255: lightning 520 res -100 plus cold 503 res 74, mdr 3; the twin is 1,164 on all three by +3 / -3). Not a shield arithmetic error; it shows the "comparable" filter can be fooled by cancelling no-shield errors.
- **K3 (low, V read).** The emulated wearer is a monster unit (`life_apply.Game`): `FUN_0057be00` then skips the player's difficulty penalty and the 75 (+max, at most 95) cap and the physical 50 cap, so cases at resistance 100 or physical reduction above 50 are not reachable on a real player. The order claim is not affected (the cap only changes the percent fed to `FUN_0057bf80`); d2d's player resistances are already the capped values.
- Carried over from `verdict.md`: C2 (physical above 4,096 points with a percent reduction; none in this job's new grid by design, not re-measured), C3 (the state is not removed at 0 mana), C4 (monster-only fields untested). The poison life in the emulator is not the frame's (the poison handler `FUN_0057ac50` is hooked, so +0x28 stays in the record total); only mana is compared on poison cases.

## PR rules and text (V run unless tagged)

- Identity and trailer: author and committer the numeric noreply identity; body 2 lines plus `Agent: Claude Code` (lines 242 and 162 characters: long, allowed). `python3 tools/pr_guard.py 561814df 229aeca7`: 0 FAIL, 0 WARN.
- Tests: Release build of `229aeca7`: `test_energy_shield` (2,400 cases, "life and mana as game.exe's order") and `test_combat` return 0. Asserts are live in Release: `tests/CMakeLists.txt` strips NDEBUG for the tests (V read).
- clang-tidy (the repo's `.clang-tidy`) on `test_energy_shield.cpp` and `fight/states.cpp`: no finding in a touched file; the only errors are `readability-identifier-length` in `outdoor.hpp` and `merc.hpp` (untouched, pre-existing). Risk for the CI tidy lane: none from this change.
- Body overreach and stale lines (`docs/research/pr-esabsorb-body.md`):
  1. Status line: "An independent re-verify of r2 has not been done": now done (this doc).
  2. "No-shield frames ... the base `2d051d77` equals them too": the r2 base is `561814df`; this run: base `561814df` equals head on every no-shield frame.
  3. Event 11 order is tagged S; it is V run here (every shield call saw the raw fields).
  4. The limit on untouched blows should name K1's case (after the mana runs out, in the same record) and state the no-shield order as a known, measured deviation for a follow-up, with 45 vs 40.
  5. Base and upstream: say the commit applies cleanly on upstream `0d424ede7` (only `ai.hpp` moved upstream, another hunk), or rebase and rerun `pr_guard` before opening.
  6. Repeatability: the differential counts come from this repo's harness (`tools/emu/esv_*`), which is not in the PR; say so, and that `test_energy_shield` is the part upstream can run.
- Nothing found that overstates the shield claim itself.

## Recommended actions

1. Edit the PR body (items 1 to 6 above); no code change. Then open.
2. Follow-up PR: the no-shield order in `monster_blow` (reduction before the resistance, a field kept negative, the record total clamped once), with the native numbers above.
3. Follow-ups as before: the state at 0 mana (C3), magic percent resistance (stat 0x25), element 256ths in `monster_blow`.

## Reproduce

`git worktree add` the fork at `229aeca7` and `561814df` as `verifier2-head` / `verifier2-base` next to this worktree, `labrun <worker> -- true` from each to mirror them, then `tools/emu/run-on-worker.sh --ttl 2400 esv_build.sh head test_energy_shield test_combat` (and `base test_combat`), the runs above, `esv_tidy.sh`, `esv2_tidyall.sh`. Outputs stay in the worker's scratch (`run-GEN-SEED/`), not in the repo.

## Shared-file lines for the merger

- BACKLOG: esverify2: Energy Shield r2 head 229aeca7 confirmed with caveats against game.exe's native FUN_005ca6c0 (docs/research/esverify/verdict2.md); ready to open after body edits (re-verify done, base 561814df, untouched fields after the mana runs out, harness not in the PR); follow-up PR: no-shield mdr/flat order and negative fields in monster_blow (game.exe 45 vs d2d 40).
- INDEX: research/esverify/verdict2.md: re-verification of the Energy Shield absorb-order PR r2 (new seeds, flat+mdr grid, poison, no-shield order confirmed natively)
- CHANGELOG: docs: Energy Shield r2 re-verified against game.exe's own FUN_005ca6c0; the no-shield mdr-before-resistance order confirmed natively (docs/research/esverify/verdict2.md, tools/emu/esv2_*)
