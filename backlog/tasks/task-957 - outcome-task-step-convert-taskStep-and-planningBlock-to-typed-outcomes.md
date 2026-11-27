---
id: TASK-957
title: 'outcome-task-step: convert taskStep and planningBlock to typed outcomes'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:41:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: low
ordinal: 456000
card: "outcome-task-step"
owns:
  - "docs/outcomes.md"
exclusive: []
contracts: []
result: "The quest runner planning block returns typed refusals for a quest that is not ready and for a ban wait. Offline tests on two classes, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Size: S. Follow-up of outcome1 (docs/outcomes.md, not yet converted list). Needs outcome1 merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 path converted behind bot.typedOutcome, row added to docs/outcomes.md, must-fail test on two builds
<!-- AC:END -->
