# Evidence pack: fix/missile-damage-fine-v2 (for the user's go / no-go)

> status: current (agent PRPROOF, 2026-10-30). One page. Detail: docs/research/pr-missile-damage-fine-proof.md; body: docs/research/pr-missile-damage-fine-draft-v2.md; standard: docs/process/upstream-pr-standard.md.

**Claim.** A missile's element damage on a monster keeps its 256ths exactly as game.exe 1.14d: resistance on the fine roll by game.exe's MulDiv, life in 256ths, dead under one point and set to 0, heals capped at max life.

**Proof summary.** game.exe's own code (FUN_0057c6c0 -> FUN_0057c1e0 -> FUN_0057bf80 / FUN_0057be00 / FUN_00483360 / FUN_0057bf10, life block LAB_0057c852..0057c874, heal FUN_0057a980) run in upstream's emulator (`tools/emu/life_apply.py`, in the branch) against d2d's real `missile_blow` + `hurt` + `heal` (`test_game --life-apply`), same generated cases, joined by `diff`. Every link in the chain was read in the disassembly (V); the one secondary fact (stats 6 / 7 in 256ths) is upstream's doc plus the 0x100 death threshold in the code.

**Numbers.**

| Scenario | Cases | Mismatches before (old d2d arithmetic) | after (v2) | real d2d binary vs game.exe |
|---|---|---|---|---|
| Boundaries, exhaustive (rolls 1..1100 x 13 resistances x lives 1..4; MulDiv 0x100000 edge) | 57,304 | in the total below | 0 | diff empty |
| Random hit / heal sequences, seed 1 | 100,000 sequences, 500,425 events | 307,385 | 0 | diff empty (157,304 lines) |
| Seed 7 after the rebase on 32768ddb | 3,000 sequences, 70,523 events | 37,993 | 0 | identical (md5) |
| Seed 99, final commit 4e6d9d39 (+ rolls 0, -1, -256, 0x7fffffff) | 3,000 sequences, 70,698 events | 38,030 | 0 | diff empty |
| Verifier's own runs (seed 99; 36,292 variant events) | | | 0 | all match |
| Roll formula FUN_005a89a0 (UPSTREAMGAPS, missile_roll.py) | 3,000 | n/a | 0 | ok |

Their checks: `ninja test_combat test_game` with D2D_WERROR on, `test_combat` ok, `test_game` rc 0 (MPQs + LODPatch_114d.exe), `pr_guard.py` 0 FAIL 0 WARN, `presubmit.py --quick` 0 FAIL 3 WARN (answered in the body). Commits authored by the user, no trailer, bodies of 3 and 1 lines, `Agent: Claude Code` in the body.

**Commands (reviewer re-runs on their copy).**

```sh
python3 tools/pr_guard.py && cmake --build build && ctest --test-dir build --output-on-failure
cd tools/emu && uv run python life_apply.py 100000 1 > g.txt
cd ../.. && build/tests/test_game --life-apply 1 100000 | grep -E '^[BMR][0-9]* ' | diff tools/emu/g.txt -    # empty
```

**What the patch needed.** PRBUILD's version was not exact: (1) the resistance arithmetic differed from MulDiv above 0x100000 256ths (4096 points a hit); (2) a dead monster kept a negative life and a stale fraction; (3) a capped heal left `life_frac` stale (up to 255/256 of a point). v2 fixes all three (`resisted_fine`, dead = 0, `heal()`), plus the verifier's two edge cases; the differential is zero only with them.

**Residual risks.**
- Domain of the proof: plain monsters (no hireling), no magic damage reduction or absorb stats, resistance -100..150, pierce only against a non-immune, whole-point heals. Pierce through an immunity is a separate d2d gap found on the way (game.exe LAB_0057be20): candidate issue, not in this PR.
- The stat getter / setter, max life, unit-kind predicates and side effects are hooked (listed in the script); the proof is of the arithmetic chain, not of the stat list.
- Scenario state hashes (tools/d2ddet, det.py) on a patched lab binary were not rerun in this job: callers other than the element branch pass whole points as before, and the only visible change is a dead monster's `hit_points` 0 instead of negative.
- Rebased on upstream/main 32768ddb (our #127, no overlap) and rechecked; fetch again right before the push (first command in the draft).
- `tools/emu/life_apply.py` is a `tools/` change: CODEOWNERS review by the maintainer; the fix commit stands alone if they decline it.
- Statistics (PRMEASURE): CB 8 casts per kill 0.60 -> 0.52, CB 4 within noise, CB 1 unusable: corroboration only.

**Verifier.** Independent second agent (Fable): **VERDICT: holds with limits.** Its two mismatches outside the domain (a roll at or under 0; a 32-bit wrap above 0x100000) are fixed in the final commit and covered by the differential; its third (a heal on a corpse) is an oracle hook, guarded in both. Its four limits (`life_frac` not replicated to co-op clients; DoT paths and partial Bone-Armor absorbs keep whole points; the Lightning Enchanted release check) are pre-existing whole-point paths, disclosed in the body. Full report in the proof doc.

**PROOF: complete** for the claim as stated (differential zero over exhaustive boundaries and 500k random events, chain fully V/S, their checks green, verifier report attached), with the residual risks above disclosed in the body.

**Recommendation.** Go, in this order and each with the user's go: (1) fetch, rebase if main moved again, rebuild, rerun the commands (expect READY); (2) push `fix/missile-damage-fine-v2`; (3) open the PR with the v2 body; at least one hour after our last PR (#128 cast-ticks). If the verifier's verdict is "broken", stop and send the finding back to a worker first.

## Posted log

- posted: 2026-10-31 PR #135 opened (head e8dd9efd at open; created 03:37Z) https://github.com/psi29a/d2decomped/pull/135 go: user, ref not recorded
- posted: 2026-10-31 comment-only commit 8a3cb678 pushed to PR #135 (04:08Z) https://github.com/psi29a/d2decomped/pull/135/commits/8a3cb6789d8954031267442fc44f5a2e976f2cbc go: user, ref not recorded
- posted: 2026-10-31 reply on PR #135 (04:09Z) https://github.com/psi29a/d2decomped/pull/135#issuecomment-6538822940 go: user, ref not recorded
- Verified 2026-10-31 by read-only `gh api` GET (pulls/135, pulls/135/commits, issues/135/comments, issues/135/events).
