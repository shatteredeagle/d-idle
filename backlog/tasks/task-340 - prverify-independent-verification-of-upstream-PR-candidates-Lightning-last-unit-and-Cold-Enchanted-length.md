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
result: "docs/upstream/verification-2026-11-11.md: both candidates NEEDS-WORK, nothing cleared for posting. Lightning (89059e2d): rule and lightningbolt proof reproduce (44/43 to 0 of 1192, seed 13 too; Bone Spear 12 to 0), but the change also reaches Lightning Fury, Inferno, Bone Spear, Arctic Blast and the standing srvdofunc 3 rows (Blizzard, poison clouds) which are unproven, and the 0 depends on d2d unit order (18 of 1192 with path order). Cold Enchanted (1b4036a3): proven (cold-only 9636 to 0, edge levels and the 110 cap = MonLvl rows read), but the oracle commit does not compile alone. Offline only (emulator, ctest on the worker host), not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Independent Opus verifier before the push go for slot A and B of docs/upstream/pr-queue.md (TASK-332, branch agent/prprep). Consumer: TASK-45 (upstream-next, posting card). Done when the doc carries a verdict per candidate.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Re-run both differentials on clean branches rebased on current upstream main, parent and claim
- [x] #2 Run the fork test suite (ctest) on both claim builds against a parent baseline
- [x] #3 Read the cited game.exe functions independently and check scope, overclaim and hygiene
- [x] #4 Verdict per candidate with required fixes in docs/upstream/verification-2026-11-11.md (consumer TASK-45, the PR-posting card)
<!-- AC:END -->
