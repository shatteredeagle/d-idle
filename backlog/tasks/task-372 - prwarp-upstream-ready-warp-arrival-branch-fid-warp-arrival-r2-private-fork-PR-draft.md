---
id: TASK-372
title: 'prwarp: upstream-ready warp arrival branch fid/warp-arrival-r2 (private fork), PR draft'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-12T18:07:00Z'
labels:
  - d2d
  - upstream-pr
dependencies: []
priority: high
ordinal: 239000
card: "prwarp"
owns:
  - "docs/upstream/drafts/warp-arrival.md"
  - "docs/upstream/pr-queue.md"
  - "tools/emu/pw"
exclusive:
  - "fork-branch:fid/warp-arrival-r2"
contracts: []
result: "fid/warp-arrival-r2 pushed to the private fork (24e8ac4f, oracle 688e53ed first, on upstream c9b4c06e), each commit builds alone (all targets). V run: synthetic 20,000 cases: 11,645 (seed 1) and 11,249 (seed 31) differ before, 0 after; full use_warp on real Act 1, seeds 5 82 7 11 23 4242: 25-26 of 52 before, 0 after; Acts 2-5 seed 5: 92 of 140 before, 2 after (shut sewer stairs; arrive's second nearest_free after an object made on arrival: both stated as limits, out of scope). pr_guard 0/0, presubmit --quick 0 FAIL 1 WARN, leakcheck 0, ctest the same 4 data-set failures. Draft docs/upstream/drafts/warp-arrival.md; ready for the second verifier pass; nothing posted. Offline, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Prepare the upstream-ready branch fid/warp-arrival-r2 (private fork only) from the TASK-366 verifier's required edits: oracle commit first, then the claim (test, drlg.md, refusal instead of the nearest_free fallback, FUN_005550b0 citation), commit format. Done when it is pushed to the fork, built and checked, and the PR draft and queue row say ready for the second verifier pass. Nothing is posted upstream.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 fid/warp-arrival-r2 pushed to the private fork: oracle commit first, then the claim (test, drlg.md, refusal, caller citation)
- [x] #2 differential numbers on seeds 5 82 7 11 23 4242 (Act 1) and Acts 2-5 seed 5, presubmit --quick, pr_guard, leakcheck, ctest
- [x] #3 PR title and body draft in docs/upstream/drafts/warp-arrival.md; queue row 'ready for second verifier pass'
<!-- AC:END -->
