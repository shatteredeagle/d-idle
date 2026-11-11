---
id: TASK-300
title: 'aggression: Paladin pack aggression knobs and evidence'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T14:40:00Z'
labels:
  - docs
  - chore
dependencies: []
priority: medium
ordinal: 209000
card: "aggression"
owns:
  - "docs/aggression.md"
exclusive: []
contracts: []
result: "Root cause: sweepPass opened a 90 s futile window after Clear killed 0 with remaining 0 (target died in between); now 3 s. Test fails without fix. Knobs changed in sandbox tuning.json. Not verified live after the fix deploy."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Paladin looked passive in big packs. Done when: evidence, root cause (futile window after a gone-target Clear) fixed and knobs documented in docs/aggression.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/aggression.md has evidence and before/after knobs
<!-- AC:END -->
