# Verification: the no-shield damage reduction order (fork branch local/noshield-reduction-order-r2)

> status: verifier verdict, 2026-11-05 (job esverify3, docs and harness only; no game, no Go change, nothing pushed). CLAIM: fork head `ca7872ab` (one commit on upstream main `4b402827`, the merge of #198) makes `rules::monster_blow` take the flat reduction and the mdr off a field BEFORE the resistance, keep the fields signed, sum them into one record total and clamp it once, as game.exe 1.14d's `FUN_0057bf80` / `FUN_0057c1e0` do. Earlier verdicts: `verdict.md`, `verdict2.md`. Tags: **V run** (run here), **V read** (read here), **S** (another doc), **U** (not checked).

## Verdict

**CONFIRMED WITH CAVEATS** (N1 to N4 below). On 135,724 frames with new seeds and a new edge grid, the head equals game.exe's own code on life (whole points and 256ths) and mana in every frame except 24 C2 frames (physical above 4,096 points with a percent reduction), where the head equals the base (pre-existing, not counted). Ready to open after the body edits under "Recommended actions"; no code change needed.

## Harness (V run, the worker host)

The merged esverify harness, unchanged in what it measures (game side: game.exe's real `FUN_0057c6c0`, which runs `FUN_0057c1e0` and `FUN_0057bf80`, and with a shield `FUN_005ca6c0`, in the emulator; d2d side: `esv_probe` through the real `monster_blow`, `Foe::take`, `Fight::absorb`), plus this job's pieces:

| Piece | What it adds |
|---|---|
| `esv_build.sh` (`ESV_TREE`), `esv3_build.sh` | clean Release builds of `verifier3-base` (`4b402827`) and `verifier3-head` (`ca7872ab`), probes plus `test_combat`, `test_energy_shield`, `test_game` |
| `esv3_cases.py` | a new edge grid, 21,670 frames with the twins: flat 0 / 3 / 15 / 60, mdr 0 / 10 / 50 / 100, resistance -100 / 0 / 75 / 100 / 110, all six damage types, physical percent 0 / 20 / 50, shield level 0 or 99 with mana 0 or 300 points, two- and three-field records with a field under the reduction, three-blow frames, physical 4,097..12,000 with a percent (C2); the flat 0 / mdr 0 rows are the zero-flat controls |
| `esv3_cmp.py` | compares EVERY frame (control twins included; no twin filter, as the claim is the no-shield path) for base and head against game.exe, by class: shield / no-shield, reduction present or not, poison, C2, resistance above 100 |
| `esv3_run.sh` | one run: cases, both probes, game.exe, comparison, the list of head differences |
| `esv3_tidy.sh`, `esv3_tidy2.sh` | clang-tidy (the fork's `.clang-tidy`) on TUs that include `combat.hpp` and `ai.hpp`, base and head; the three tests with the 1.14d data |

Runs: `tools/emu/run-on-worker.sh --ttl 2400 esv3_run.sh 2 41 20000` (and 42, 43: the second generator `rand_case2`, new seeds; the author used 31 / 32 / 33 of the first), `esv3_run.sh 4 0 0` (the grid). Self-checks on every run: calc tables of base and head equal, event 11 saw the raw fields (`snapshot_not_raw 0`), no hook called with unexpected arguments, no case missing.

## Counts (V run)

| Run | Frames | Base life differs (pt / 256ths) | Head life differs (pt / 256ths) | Mana differs base / head |
|---|---|---|---|---|
| seeds 41 + 42 + 43, all | 114,054 | 36,449 / 45,942 | **0 / 0** | 0 / 0 |
| of them no shield | 60,000 | 20,029 / 26,156 | **0 / 0** | 0 / 0 |
| no shield, flat or mdr non-zero, no poison | 35,173 | 18,227 / 21,242 | **0 / 0** | 0 / 0 |
| no shield, no reduction, no poison | 11,615 | 1,802 / 4,914 | **0 / 0** | 0 / 0 |
| with a shield | 54,054 | 16,420 / 19,786 | **0 / 0** | 0 / 0 |
| edge grid, all | 21,670 | 3,447 / 4,092 | 12 / 24, all C2 | 0 / 0 |
| of them resistance 110 (above the cap) | 3,952 | 516 / 516 | **0 / 0** | 0 / 0 |
| of them C2 | 54 | 12 / 24 | 12 / 24 (head == base on all 54) | 0 / 0 |

Poison frames (25,132 in the seeds, 1,616 in the grid) are compared on mana only: the emulator hooks the poison handler, so game.exe's life there is not the frame's (as in `verdict2.md`). Head ids that differ: only the grid's C2 ids 40012768..40012803 and their twins, e.g. 40012768 (physical 4,097, percent 1): game 1,038,312 / 256, base and head 1,038,343.

## Break attempts (V run)

| Attempt | Result |
|---|---|
| mdr 100 and flat 60 against rolls 1..2000: every field under the reduction, alone and next to others | head equal on all; the total is clamped once (sum of negatives, life 0) |
| resistance 110 (above `min(res, 100)`), 100, -100 on every element | head equal |
| three fields with one under the mdr, mixed resistances (-100 / 75 / 100), physical percent 30, shield 99 or 40 | head equal |
| three blows in one frame, each with fields under the reduction | head equal |
| shield up at mana 0 and 20 points (the untouched fields after the mana runs out: K1 of `verdict2.md`) | head equal: K1 closed |
| zero-flat control (flat 0, mdr 0): must stay equal to game.exe | head equal; the base differed in 256ths here (it resisted each element in whole points) |
| physical above 4,096 with a percent (C2) | head == base != game.exe (MulDiv over 0x100000), pre-existing |

## Tests and checks (V run unless tagged)

- Release builds of base and head with the 1.14d data on the worker host: `test_combat`, `test_energy_shield` (2,400 cases) and `test_game` return 0 on BOTH (`esv3_tidy.sh`), so the two tests the author left at exit 77 do pass on `ca7872ab`.
- `python3 tools/pr_guard.py 4b402827 ca7872ab` (the fork's tool, run in the head worktree): 0 FAIL, 0 WARN. Author and committer the numeric noreply identity; body ends `Agent: Claude Code`.
- clang-tidy, LLVM 23.1.4: `tests/test_combat.cpp` no finding on base or head; `components/game/ai.cpp` and `components/game/world/tick.cpp` give the SAME 10 / 4 `readability-identifier-length` errors on base and head, all structured bindings `x`, `y` in `drlg/outdoor.hpp`, `rules/merc.hpp`, `rules/pets.hpp`; none in `combat.hpp` or `ai.hpp`. CI's "Names and includes" job (`.github/workflows/ci.yml`, V read) is changed-FILE, not changed-line: a PR tidies its changed `.cpp` / `.hpp` and `tools/tidy-pr-select.py` adds every TU that includes a changed header, with `HeaderFilterRegex` covering `components/`, so those headers' findings WOULD be reported on this PR if CI's clang-tidy produced them. They are identical on upstream main, whose push sweep tidies everything, so they are a local-version artefact (U: CI's clang-tidy version not checked); the change adds none.

## Agreement with the author's body (`docs/research/pr-noshield-order-body.md`, read last)

Agrees: head 0 differences on the no-shield frames with and without flat / mdr, on shield frames, mana never differs, the K1/K2 frames closed, the 256ths-sum effect on no-reduction frames. The new seeds and grid reproduce "after 0 / 0" with larger samples and wider axes (mdr 50 / 100, flat 60, resistance 110, three fields, three blows) than the author's.

## Caveats

- **N1 (medium, V run). The change moves frames with NO reduction too**, and that is a large share: no-shield frames without flat or mdr differ base vs game.exe in whole points on 1,802 of 11,615 (16 %), because the base resisted each element in whole points and the head sums 256ths. The body says so in one paragraph under the table; the title and first paragraph speak only of the order. Say up front that the commit does two things `FUN_0057c1e0` does (the order and the 256ths sum).
- **N2 (low, V run). C2 remains** (physical above 4,096 points with a percent: the head keeps 64-bit arithmetic where game.exe's MulDiv divides first). The body says "the grid has none by design"; this grid has 54 such frames: head == base on all, 24 differ from game.exe by under one point. Pre-existing, outside the claim.
- **N3 (low, U).** Bone Armor and Cyclone Armor with negative fields or with a shield are not in any differential (neither the author's nor this one: the harness cannot put an armor state up). `absorb_blows`' elemental path takes whole points off the total without updating `parts[].fine`, so a shield after Cyclone Armor reads stale fields (V read; the same staleness existed with `dealt` on the base, the "Cyclone Armor cut is not put back through the resistance" ponytail).
- **N4 (low, S).** Poison life, the monster-only fields (C4), the aura and burn callers that bypass `monster_blow`, the state at 0 mana (C3), MSVC / clang++ and the parity lane: not run here either.

## Recommended actions

1. Body edits, then open: (a) status line: an independent verification is done (this doc) and the r2 head `ca7872ab` is measured, not only `fe434d09`; (b) Tests: `test_game` and `test_energy_shield` pass with data on `ca7872ab` (V run here), drop the exit 77 sentence; (c) first paragraph: name the 256ths sum next to the order (N1) with its share; (d) C2 line: "pre-existing, measured: head == base on 54 frames" instead of "none by design"; (e) the tidy line: findings identical on base and head, none in a touched header; (f) keep "the harness is not in this PR; only test_combat is repeatable upstream".
2. Follow-ups (not blocking): C2, the aura / burn callers through the same order, an armor-state differential (N3).

## Reproduce

`git worktree add --detach` the fork at `ca7872ab` and `4b402827` as `verifier3-head` / `verifier3-base` next to this worktree, `labrun <worker> -- true` from each to mirror them, then `tools/emu/run-on-worker.sh --ttl 2400 esv3_build.sh`, the runs above, `esv3_tidy.sh`, `esv3_tidy2.sh`. Outputs stay in the worker's scratch (`run3-GEN-SEED/`), not in the repo.

## Shared-file lines for the merger

In `backlog/merge-lines/agent-esverify3.txt`.
