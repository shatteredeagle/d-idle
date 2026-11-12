---
id: TASK-332
title: 'prprep: upstream PR prep: audit of local candidates, Lightning and Cold Enchanted rebased with differentials, drafts, pr-queue'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:35:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-skills
dependencies: []
priority: high
ordinal: 218000
card: "prprep"
owns:
  - "docs/upstream/"
  - "tools/emu/pp_*"
exclusive:
  - "fork-branch:fid/missile-last-unit-r3"
  - "fork-branch:fid/cold-enchanted-length-r4"
  - "upstream-pr:lightning-last-unit"
  - "upstream-pr:cold-enchanted-length"
contracts: []
result: "Audit and two candidates prepared (2026-11-11, agent prprep): fid/missile-last-unit-r3 (5effaf98) and fid/cold-enchanted-length-r4 (30713b60) rebased on upstream b6b59ae9, differentials re-run (V run: Lightning 44 of 1192 casts differ before, 0 after; Cold Enchanted cold-only 7869 mismatches before, 0 after), drafts and docs/upstream/pr-queue.md written; verifier round 1 NEEDS-WORK applied (r3 / r4: srv_do == 1 scope, oracle commit builds alone, 3 seeds, ctest same 4 data-set failures as the parent); ready for the verifier's second pass; nothing posted; no live game."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Audit every local PR candidate against current upstream main (b6b59ae9, 2026-11-11), finish the two closest to the standard, write drafts and the ranked queue. Nothing posted. Done when: docs/upstream/pr-queue.md and the drafts are merged and the independent verifier has the two branches.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every local candidate audited against upstream main in docs/upstream/pr-queue.md
- [x] #2 Lightning last-unit and Cold Enchanted rebased, differentials re-run (before > 0, after 0), pushed to the private fork remote
- [x] #3 Drafts in docs/upstream/drafts/ and the ranked queue with the bundle plan
- [ ] #4 Independent verifier clears the two branches (later; live-check style box)
<!-- AC:END -->
