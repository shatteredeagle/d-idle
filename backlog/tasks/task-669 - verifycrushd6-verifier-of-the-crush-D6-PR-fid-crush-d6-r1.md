---
id: TASK-669
title: 'verifycrushd6: verifier of the crush D6 PR (fid/crush-d6-r1)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-19T17:16:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - d2d
  - readonly
  - d2d-combat
  - fidelity
dependencies: []
priority: high
ordinal: 328000
card: "verifycrushd6"
owns:
  - "docs/upstream/verifier-crushd6-2026-11-19.md"
exclusive: []
contracts: []
result: "r1 FAIL (stale base da98b7f8; m8 missile, m9 marked, Fight::land call site and a life mutant escape its pins and oracle), r3 PASS WITH CHANGES: fid/crush-d6-r3 oracle f2a7007c + claim 8867eb7f on upstream d5de50ca, pushed to the private fork (r2 629e3126 superseded by 4 tidy fixes); oracle now through Fight::land with missile, fire and life: parent 65,145/65,544/65,316 of 103,520 differ, claim 0, controls 0; 13/13 non-equivalent mutants abort the 16 pins; ctest 407/408 both trees (smoke_d2d, as upstream); tidy 0 (control caught); D5+D6 combined 0 differ on both oracles; body changes C1-C8; siblings F1 (MonProp crush on monster/pet) and F2 (Life Tap); offline, not verified live"
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: independent Opus check of fid/crush-d6-r1 (decomp42) before posting upstream. Consumer: the posting agent (docs/upstream/pr-queue.md). Done when: the verdict doc names the branch and commits to post and the body changes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Rule re-derived from the decompilation for every path through FUN_0057c6c0; d2d's other crush and get-hit users checked
- [x] #2 Rebased on current upstream main and on D5; build, full ctest vs parent, oracle, mutants incl. missile and element, tidy, pr_guard, presubmit on the builder
- [x] #3 Verdict doc with required body changes and the exact branch and commits (pushed to the private fork)
<!-- AC:END -->
