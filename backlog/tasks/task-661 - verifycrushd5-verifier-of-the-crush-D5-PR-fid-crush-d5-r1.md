---
id: TASK-661
title: 'verifycrushd5: verifier of the crush D5 PR (fid/crush-d5-r1)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-19T14:49:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - d2d
  - readonly
  - d2d-combat
  - fidelity
dependencies: []
priority: high
ordinal: 327000
card: "verifycrushd5"
owns:
  - "docs/upstream/verifier-crushd5-2026-11-19.md"
exclusive: []
contracts: []
result: "r1 FAIL (stale base 83 commits; 4 mutants pass its pins, 2 of them in Monster::target pass its oracle too; writer scan wording overclaims: a second setter FUN_004ac810, 0x5a4599 is a re-make from a saved record), r2 PASS WITH CHANGES: fid/crush-d5-r2 oracle 3d51e67e + claim 5824a42f on upstream d5de50ca, pushed to the private fork; parent 5,632 of 29,360 differ (all unique), claim 0; 17/17 non-equivalent mutants abort r2's pins; ctest 407/408 + 13 skip, smoke_d2d fails identically on upstream d5de50ca; tidy 0 findings (control caught); pr_guard 0/0; presubmit 0 FAIL 4 WARN; body changes C1-C8; offline, not verified live"
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: independent Opus check of fid/crush-d5-r1 (decomp41) before posting upstream. Consumer: the posting agent (docs/upstream/pr-queue.md). Done when: the verdict doc names the branch and commits to post and the body changes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Rule re-derived from the decompilation; boss classes mapped both sides; bit-2 writers scanned beyond the author's scan
- [x] #2 Rebased on current upstream main; build, full ctest vs parent, oracle, 10+ mutants, tidy, pr_guard, presubmit on the builder
- [x] #3 Verdict doc with required body changes and the exact branch and commits (pushed to the private fork)
<!-- AC:END -->
