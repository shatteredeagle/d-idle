---
id: TASK-372
title: 'prwarp: upstream-ready warp arrival branch fid/warp-arrival-r2 (private fork), PR draft'
status: Proposed
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 fid/warp-arrival-r2 pushed to the private fork: oracle commit first, then the claim (test, drlg.md, refusal, caller citation)
- [ ] #2 differential numbers on seeds 5 82 7 11 23 4242 (Act 1) and Acts 2-5 seed 5, presubmit --quick, pr_guard, leakcheck, ctest
- [ ] #3 PR title and body draft in docs/upstream/drafts/warp-arrival.md; queue row 'ready for second verifier pass'
<!-- AC:END -->
