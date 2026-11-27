---
id: TASK-1067
title: 'gearcopy: gear builds read internal/plandata, no second copy'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-27T21:30:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 514000
card: "gearcopy"
lane: next
owns:
  - "internal/gear"
  - "docs/gear.md"
result: "Offline only, not verified live. Gear scoring now reads the plan tables from internal/plandata; the 15 copies under internal/gear/builds (which had drifted from the SME-reviewed tables) are deleted."
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/gear/builds removed; gear.BuildJSON reads plandata
- [x] #2 Test fails if a copy returns or the build list drifts from plandata
<!-- AC:END -->
