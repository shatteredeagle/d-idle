---
id: TASK-340
title: 'prverify: independent verification of upstream PR candidates Lightning last unit and Cold Enchanted length'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T20:50:00Z'
labels:
  - d2d
  - readonly
dependencies: []
priority: high
ordinal: 225000
card: "prverify"
owns:
  - "docs/upstream/verification-2026-11-11.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Independent Opus verifier before the push go for slot A and B of docs/upstream/pr-queue.md (TASK-332, branch agent/prprep). Consumer: TASK-45 (upstream-next, posting card). Done when the doc carries a verdict per candidate.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Re-run both differentials on clean branches rebased on current upstream main, parent and claim
- [ ] #2 Run the fork test suite (ctest) on both claim builds against a parent baseline
- [ ] #3 Read the cited game.exe functions independently and check scope, overclaim and hygiene
- [ ] #4 Verdict per candidate with required fixes in docs/upstream/verification-2026-11-11.md (consumer TASK-45, the PR-posting card)
<!-- AC:END -->
