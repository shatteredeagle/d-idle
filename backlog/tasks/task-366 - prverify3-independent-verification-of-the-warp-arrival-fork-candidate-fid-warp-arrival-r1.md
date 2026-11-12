---
id: TASK-366
title: 'prverify3: independent verification of the warp-arrival fork candidate (fid/warp-arrival-r1)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-12T16:22:00Z'
labels:
  - d2d
  - research
dependencies: []
priority: high
ordinal: 237000
card: "prverify3"
owns:
  - "docs/upstream/verification-warp-arrival-2026-11-12.md"
exclusive: []
contracts: []
result: "docs/upstream/verification-warp-arrival-2026-11-12.md: fid/warp-arrival-r1 (feba9539) NEEDS-WORK. Rule and code proven: full use_warp path, Act 1 seeds 5 82 7 11 23 4242: 25-26 of 52 differ before, 0 after (as pushed and rebased on c9b4c06e); Acts 2-5 seed 5: 92 of 140 to 2 (one shut sewer stairs, one object made on arrival); game.exe chain read (rings 1-49, ties, plus footprint, refusal). Missing for upstream: a test, the oracle in the PR, drlg.md row, faithful refusal instead of the nearest_free fallback, caller citation, commit format (new r2 branch). ctest same 4 data-set failures; builds, pr_guard, presubmit ok. Offline, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Opus independent verifier for the private fork candidate fid/warp-arrival-r1 (feba9539, TASK-350 decomp12, docs/probes/fid/fid-warp-arrival.md) against psi29a/d2decomped's rules and the standard of docs/upstream/verification-2026-11-11.md. Consumer: TASK-45 (upstream PR posting). Done when the doc carries a PASS / NEEDS-WORK / FAIL verdict with the exact edits.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Differential re-run on clean worktrees, seeds 5 82 7 and three new seeds, through the full use_warp path, before and after
- [x] #2 game.exe chain FUN_005550b0 / FUN_00554ea0 / FUN_0064e7b0 / FUN_0064dea0 read here: ring order, ties, range 49, blocked definition, fallthrough
- [x] #3 Regressions: other callers, d2d ctest on parent and claim, as pushed and rebased
- [x] #4 Every commit builds alone, pr_guard, presubmit --quick, no private or fork-only content
- [x] #5 One topic, minimal diff, commit conventions; verdict and exact edits in docs/upstream/verification-warp-arrival-2026-11-12.md
<!-- AC:END -->
